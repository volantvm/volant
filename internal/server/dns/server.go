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
	"time"

	"github.com/miekg/dns"
	"github.com/volantvm/volant/internal/server/db"
)

// Server provides DNS resolution for VM and deployment service discovery.
// Resolves <vm-name>.volant → VM IP and <deployment-name>.volant → round-robin IPs.
// Forwards external queries to upstream DNS servers (e.g., 1.1.1.1).
type Server struct {
	store     db.Store
	listen    string      // e.g., "192.168.127.1:53"
	domain    string      // e.g., "volant"
	upstreams []string    // Upstream DNS servers (e.g., "1.1.1.1:53", "8.8.8.8:53")
	logger    *slog.Logger
	server    *dns.Server
	client    *dns.Client // DNS client for upstream queries
}

// New creates a new DNS server instance.
// upstreams: comma-separated list of upstream DNS servers (e.g., "1.1.1.1:53,8.8.8.8:53")
func New(store db.Store, listen, domain string, upstreams []string, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	if len(upstreams) == 0 {
		upstreams = []string{"1.1.1.1:53", "8.8.8.8:53"} // Default to Cloudflare and Google DNS
	}
	return &Server{
		store:     store,
		listen:    listen,
		domain:    domain,
		upstreams: upstreams,
		logger:    logger,
		client:    &dns.Client{Net: "udp", Timeout: 5 * time.Second},
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
// If not found internally, forwards to upstream DNS servers.
func (s *Server) handleQuery(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)
	m.Authoritative = true

	for _, q := range r.Question {
		if q.Qtype == dns.TypeA {
			// Check if this is a .volant domain query
			isVolantDomain := strings.HasSuffix(q.Name, "."+s.domain+".") ||
				strings.HasSuffix(q.Name, s.domain+".")

			if isVolantDomain {
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
			} else {
				// External domain - forward to upstream DNS
				s.logger.Debug("forwarding external dns query", "query", q.Name)
				response := s.forwardToUpstream(r)
				if response != nil {
					if err := w.WriteMsg(response); err != nil {
						s.logger.Error("dns write upstream response error", "error", err)
					}
					return
				}
				// If all upstreams fail, return server failure
				m.SetRcode(r, dns.RcodeServerFailure)
			}
		}
	}

	if err := w.WriteMsg(m); err != nil {
		s.logger.Error("dns write response error", "error", err)
	}
}

// forwardToUpstream forwards a DNS query to upstream resolvers.
// Tries each upstream in order until one succeeds.
func (s *Server) forwardToUpstream(r *dns.Msg) *dns.Msg {
	for _, upstream := range s.upstreams {
		response, _, err := s.client.Exchange(r, upstream)
		if err != nil {
			s.logger.Warn("upstream dns query failed", "upstream", upstream, "error", err)
			continue
		}
		if response != nil {
			s.logger.Debug("upstream dns query succeeded", "upstream", upstream, "answers", len(response.Answer))
			return response
		}
	}
	s.logger.Error("all upstream dns servers failed")
	return nil
}

// resolveService looks up a VM or deployment by name and returns IP address(es).
// Returns single IP for VM, multiple IPs for deployment (round-robin).
func (s *Server) resolveService(name string) ([]string, error) {
	ctx := context.Background()

	// Try VM lookup first
	vm, err := s.store.Queries().VirtualMachines().GetByName(ctx, name)
	if err == nil && vm != nil && vm.IPAddress != "" && vm.Status == db.VMStatusRunning {
		return []string{vm.IPAddress}, nil
	}

	// Try deployment lookup (returns all running VM IPs for round-robin)
	group, err := s.store.Queries().VMGroups().GetByName(ctx, name)
	if err == nil && group != nil {
		vms, err := s.store.Queries().VirtualMachines().ListByGroupID(ctx, group.ID)
		if err == nil {
			var ips []string
			for _, vm := range vms {
				if vm.IPAddress != "" && vm.Status == db.VMStatusRunning {
					ips = append(ips, vm.IPAddress)
				}
			}
			if len(ips) > 0 {
				return ips, nil
			}
		}
	}

	return nil, fmt.Errorf("service not found: %s", name)
}
