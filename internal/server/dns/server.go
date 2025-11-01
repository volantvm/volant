// Copyright (c) 2025 HYPR. PTE. LTD.
//
// Business Source License 1.1
// See LICENSE file in the project root for details.

package dns

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"strings"

	"github.com/miekg/dns"
	"github.com/volantvm/volant/internal/server/db"
)

// Server provides DNS resolution for VM and deployment service discovery.
// Resolves <vm-name>.volant → VM IP and <deployment-name>.volant → round-robin IPs.
type Server struct {
	store  db.Store
	listen string      // e.g., "192.168.127.1:53"
	domain string      // e.g., "volant"
	logger *slog.Logger
	server *dns.Server
}

// New creates a new DNS server instance.
func New(store db.Store, listen, domain string, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	return &Server{
		store:  store,
		listen: listen,
		domain: domain,
		logger: logger,
	}
}

// Start launches the DNS server and blocks until context is cancelled.
func (s *Server) Start(ctx context.Context) error {
	mux := dns.NewServeMux()
	mux.HandleFunc(s.domain+".", s.handleQuery)
	mux.HandleFunc(".", s.handleQuery) // Catch-all for short names

	s.server = &dns.Server{
		Addr:    s.listen,
		Net:     "udp",
		Handler: mux,
	}

	// Shutdown on context cancellation
	go func() {
		<-ctx.Done()
		s.logger.Info("dns server shutting down")
		if err := s.server.Shutdown(); err != nil {
			s.logger.Error("dns server shutdown error", "error", err)
		}
	}()

	s.logger.Info("dns server starting", "listen", s.listen, "domain", s.domain)
	if err := s.server.ListenAndServe(); err != nil {
		return fmt.Errorf("dns server listen: %w", err)
	}
	return nil
}

// handleQuery processes DNS queries for VM and deployment names.
func (s *Server) handleQuery(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)
	m.Authoritative = true

	for _, q := range r.Question {
		if q.Qtype == dns.TypeA {
			// Extract service name from query
			// e.g., "my-vm.volant." → "my-vm"
			name := strings.TrimSuffix(q.Name, "."+s.domain+".")
			name = strings.TrimSuffix(name, ".")

			// Resolve to IP(s)
			ips, err := s.resolveService(name)
			if err != nil || len(ips) == 0 {
				s.logger.Debug("dns query not found", "name", name, "query", q.Name)
				m.SetRcode(r, dns.RcodeNameError)
				if err := w.WriteMsg(m); err != nil {
					s.logger.Error("dns write response error", "error", err)
				}
				return
			}

			s.logger.Debug("dns query resolved", "name", name, "ips", ips)

			// Add A records for each IP
			for _, ip := range ips {
				rr := &dns.A{
					Hdr: dns.RR_Header{
						Name:   q.Name,
						Rrtype: dns.TypeA,
						Class:  dns.ClassINET,
						Ttl:    10, // Short TTL (10s) for dynamic updates
					},
					A: net.ParseIP(ip),
				}
				m.Answer = append(m.Answer, rr)
			}
		}
	}

	if err := w.WriteMsg(m); err != nil {
		s.logger.Error("dns write response error", "error", err)
	}
}

// resolveService looks up a VM or deployment by name and returns IP address(es).
// Returns single IP for VM, multiple IPs for deployment (round-robin).
func (s *Server) resolveService(name string) ([]string, error) {
	ctx := context.Background()

	// Try VM lookup first
	vm, err := s.store.WithTx(ctx, func(q db.Queries) (interface{}, error) {
		return q.VirtualMachines().GetByName(ctx, name)
	})
	if err == nil && vm != nil {
		if vmRecord, ok := vm.(*db.VM); ok && vmRecord.IPAddress != "" && vmRecord.Status == db.VMStatusRunning {
			return []string{vmRecord.IPAddress}, nil
		}
	}

	// Try deployment lookup (returns all running VM IPs for round-robin)
	deployment, err := s.store.WithTx(ctx, func(q db.Queries) (interface{}, error) {
		return q.Deployments().GetByName(ctx, name)
	})
	if err == nil && deployment != nil {
		if depRecord, ok := deployment.(*db.Deployment); ok {
			vms, err := s.store.WithTx(ctx, func(q db.Queries) (interface{}, error) {
				return q.VirtualMachines().ListByGroupID(ctx, depRecord.ID)
			})
			if err == nil && vms != nil {
				if vmList, ok := vms.([]*db.VM); ok {
					var ips []string
					for _, vm := range vmList {
						if vm.IPAddress != "" && vm.Status == db.VMStatusRunning {
							ips = append(ips, vm.IPAddress)
						}
					}
					if len(ips) > 0 {
						return ips, nil
					}
				}
			}
		}
	}

	return nil, fmt.Errorf("service not found: %s", name)
}
