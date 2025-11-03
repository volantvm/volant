//go:build linux

package dataplane

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"sync"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
)

type manager struct {
	logger        *slog.Logger
	xdpProgram    *ebpf.Program
	egressProgram *ebpf.Program
	portmap       *ebpf.Map
	conntrack     *ebpf.Map
	stats         *ebpf.Map
	xdpLink       link.Link
	egressLink    link.Link
	extInterface  string
	brInterface   string
	mu            sync.Mutex
	closed        bool
	xdpColl       *ebpf.Collection
	egressColl    *ebpf.Collection
}

func newManager(opts Options) (Interface, error) {
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	if opts.ObjectPath == "" {
		return nil, fmt.Errorf("dataplane: bpf object path required")
	}
	if opts.Interface == "" {
		return nil, fmt.Errorf("dataplane: bridge interface name required")
	}

	// Determine paths for XDP and egress programs
	xdpPath := opts.ObjectPath + "_xdp.bpf.o"
	egressPath := opts.ObjectPath + "_egress.bpf.o"

	// Load XDP collection (ingress)
	xdpColl, err := ebpf.LoadCollection(xdpPath)
	if err != nil {
		return nil, fmt.Errorf("dataplane: load xdp collection: %w", err)
	}

	// Load egress collection
	egressColl, err := ebpf.LoadCollection(egressPath)
	if err != nil {
		xdpColl.Close()
		return nil, fmt.Errorf("dataplane: load egress collection: %w", err)
	}

	// Get XDP program
	xdpProg, ok := xdpColl.Programs["drift_l4_ingress"]
	if !ok {
		xdpColl.Close()
		egressColl.Close()
		return nil, errors.New("dataplane: program drift_l4_ingress not found in xdp collection")
	}

	// Get egress program
	egressProg, ok := egressColl.Programs["drift_l4_egress"]
	if !ok {
		xdpColl.Close()
		egressColl.Close()
		return nil, errors.New("dataplane: program drift_l4_egress not found in egress collection")
	}

	// Get shared maps from XDP collection (ingress creates them)
	portmap, ok := xdpColl.Maps["portmap"]
	if !ok {
		xdpColl.Close()
		egressColl.Close()
		return nil, errors.New("dataplane: portmap not found")
	}

	conntrack, ok := xdpColl.Maps["conntrack"]
	if !ok {
		xdpColl.Close()
		egressColl.Close()
		return nil, errors.New("dataplane: conntrack not found")
	}

	stats, ok := xdpColl.Maps["stats"]
	if !ok {
		xdpColl.Close()
		egressColl.Close()
		return nil, errors.New("dataplane: stats not found")
	}

	// Pin shared maps and update egress collection to use them
	if err := pinAndReuseMap(conntrack, egressColl, "conntrack"); err != nil {
		xdpColl.Close()
		egressColl.Close()
		return nil, fmt.Errorf("dataplane: reuse conntrack map: %w", err)
	}

	if err := pinAndReuseMap(stats, egressColl, "stats"); err != nil {
		xdpColl.Close()
		egressColl.Close()
		return nil, fmt.Errorf("dataplane: reuse stats map: %w", err)
	}

	// Determine external interface for XDP attachment
	extIface := opts.ExternalInterface
	if extIface == "" {
		extIface = opts.Interface // Fallback to bridge if not specified
	}

	// Attach XDP to external interface
	extIfaceObj, err := net.InterfaceByName(extIface)
	if err != nil {
		xdpColl.Close()
		egressColl.Close()
		return nil, fmt.Errorf("dataplane: lookup external interface %s: %w", extIface, err)
	}

	xdpLink, err := link.AttachXDP(link.XDPOptions{
		Program:   xdpProg,
		Interface: extIfaceObj.Index,
	})
	if err != nil {
		xdpColl.Close()
		egressColl.Close()
		return nil, fmt.Errorf("dataplane: attach xdp: %w", err)
	}

	opts.Logger.Info("drift dataplane XDP attached", "interface", extIface)

	// Attach TC egress to bridge
	brIfaceObj, err := net.InterfaceByName(opts.Interface)
	if err != nil {
		xdpLink.Close()
		xdpColl.Close()
		egressColl.Close()
		return nil, fmt.Errorf("dataplane: lookup bridge interface %s: %w", opts.Interface, err)
	}

	egressLink, err := link.AttachTCX(link.TCXOptions{
		Program:   egressProg,
		Interface: brIfaceObj.Index,
		Attach:    ebpf.AttachTCXEgress,
	})
	if err != nil {
		xdpLink.Close()
		xdpColl.Close()
		egressColl.Close()
		return nil, fmt.Errorf("dataplane: attach tc egress: %w", err)
	}

	opts.Logger.Info("drift dataplane TC egress attached", "interface", opts.Interface)

	return &manager{
		logger:        opts.Logger.With("component", "dataplane"),
		xdpProgram:    xdpProg,
		egressProgram: egressProg,
		portmap:       portmap,
		conntrack:     conntrack,
		stats:         stats,
		xdpLink:       xdpLink,
		egressLink:    egressLink,
		extInterface:  extIface,
		brInterface:   opts.Interface,
		xdpColl:       xdpColl,
		egressColl:    egressColl,
	}, nil
}

// pinAndReuseMap pins a map from one collection and updates another collection to use it
func pinAndReuseMap(m *ebpf.Map, destColl *ebpf.Collection, name string) error {
	pinPath := fmt.Sprintf("/sys/fs/bpf/drift_%s", name)

	// Pin the map
	if err := m.Pin(pinPath); err != nil {
		// If already exists, that's fine
		if !errors.Is(err, ebpf.ErrMapAlreadyExists) {
			return err
		}
	}

	// Update destination collection's map to point to the pinned map
	destMap, ok := destColl.Maps[name]
	if ok {
		destMap.Close()
	}

	// Load pinned map
	pinnedMap, err := ebpf.LoadPinnedMap(pinPath, nil)
	if err != nil {
		return err
	}

	destColl.Maps[name] = pinnedMap
	return nil
}

func (m *manager) ApplyBridge(_ context.Context, proto uint8, hostPort uint16, destIP net.IP, destPort uint16) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return errors.New("dataplane: manager closed")
	}

	ip4 := destIP.To4()
	if ip4 == nil {
		return fmt.Errorf("dataplane: destination ip %s not ipv4", destIP)
	}

	// CRITICAL: Store IP in network byte order (big-endian)
	// Convert IP bytes to uint32 in big-endian order
	ipBigEndian := binary.BigEndian.Uint32(ip4)

	key := portmapKey{
		Proto: proto,
		Port:  htons(hostPort),
	}

	value := portmapValue{
		DestIP:   ipBigEndian,   // Already in network byte order
		DestPort: htons(destPort),
	}

	if err := m.portmap.Put(&key, &value); err != nil {
		return fmt.Errorf("dataplane: portmap update: %w", err)
	}

	m.logger.Info("route applied", "proto", protoName(proto), "host_port", hostPort, "dest_ip", destIP.String(), "dest_port", destPort)
	return nil
}

func (m *manager) ConfigureLoadBalancer(_ context.Context, config LoadBalancerConfig) error {
	return errors.New("dataplane: load balancer not yet implemented")
}

func (m *manager) SetBackendHealth(_ context.Context, proto uint8, hostPort uint16, backendIdx uint32, healthy bool) error {
	return errors.New("dataplane: backend health not yet implemented")
}

func (m *manager) Remove(_ context.Context, proto uint8, hostPort uint16) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return errors.New("dataplane: manager closed")
	}

	key := portmapKey{
		Proto: proto,
		Port:  htons(hostPort),
	}

	if err := m.portmap.Delete(&key); err != nil {
		return fmt.Errorf("dataplane: portmap delete: %w", err)
	}

	m.logger.Info("route removed", "proto", protoName(proto), "host_port", hostPort)
	return nil
}

func (m *manager) Stats(_ context.Context) (Stats, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return Stats{}, errors.New("dataplane: manager closed")
	}

	var result Stats
	var ingressPackets, ingressBytes, egressPackets, egressBytes uint64

	key0 := uint32(0) // STAT_INGRESS_PACKETS
	key1 := uint32(1) // STAT_INGRESS_BYTES
	key2 := uint32(2) // STAT_EGRESS_PACKETS
	key3 := uint32(3) // STAT_EGRESS_BYTES

	m.stats.Lookup(&key0, &ingressPackets)
	m.stats.Lookup(&key1, &ingressBytes)
	m.stats.Lookup(&key2, &egressPackets)
	m.stats.Lookup(&key3, &egressBytes)

	result.IngressPackets = ingressPackets
	result.IngressBytes = ingressBytes
	result.EgressPackets = egressPackets
	result.EgressBytes = egressBytes

	return result, nil
}

func (m *manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return nil
	}
	m.closed = true

	if m.xdpLink != nil {
		_ = m.xdpLink.Close()
	}
	if m.egressLink != nil {
		_ = m.egressLink.Close()
	}
	if m.xdpColl != nil {
		m.xdpColl.Close()
	}
	if m.egressColl != nil {
		m.egressColl.Close()
	}

	m.logger.Info("drift dataplane closed")
	return nil
}

type portmapKey struct {
	Proto uint8
	_     uint8
	Port  uint16
}

type portmapValue struct {
	DestIP   uint32
	DestPort uint16
	_        uint16
}

func htons(v uint16) uint16 {
	return (v << 8) | (v >> 8)
}

func protoName(proto uint8) string {
	if proto == 6 {
		return "tcp"
	}
	if proto == 17 {
		return "udp"
	}
	return fmt.Sprintf("%d", proto)
}
