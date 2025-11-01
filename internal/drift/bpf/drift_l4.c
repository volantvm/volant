#include "vmlinux.h"
#include <bpf/bpf_endian.h>
#include <bpf/bpf_helpers.h>

#ifndef TC_ACT_OK
#define TC_ACT_OK 0
#endif

#ifndef ETH_P_IP
#define ETH_P_IP 0x0800
#endif

// Portmap: maps (protocol, host_port) -> (backend_ip, backend_port)
struct portmap_key {
	__u8 proto;
	__u8 pad;
	__be16 port;
};

struct portmap_value {
	__be32 dst_ip;
	__be16 dst_port;
	__u16 pad;
};

struct {
	__uint(type, BPF_MAP_TYPE_HASH);
	__uint(max_entries, 4096);
	__type(key, struct portmap_key);
	__type(value, struct portmap_value);
} portmap SEC(".maps");

// Conntrack: tracks active connections for reverse NAT
// Key: 5-tuple of the connection (client -> backend)
struct conntrack_key {
	__be32 src_ip;   // Client IP
	__be32 dst_ip;   // Backend IP
	__be16 src_port; // Client port
	__be16 dst_port; // Backend port
	__u8 proto;
	__u8 pad[3];
};

// Value: original destination as seen by client
struct conntrack_value {
	__be32 orig_dst_ip;   // Host IP
	__be16 orig_dst_port; // Host port
	__u16 pad;
	__u64 last_seen;      // Timestamp for cleanup
};

struct {
	__uint(type, BPF_MAP_TYPE_LRU_HASH);
	__uint(max_entries, 65536);
	__type(key, struct conntrack_key);
	__type(value, struct conntrack_value);
} conntrack SEC(".maps");

// Metrics
struct {
	__uint(type, BPF_MAP_TYPE_PERCPU_ARRAY);
	__uint(max_entries, 4);
	__type(key, __u32);
	__type(value, __u64);
} stats SEC(".maps");

#define STAT_INGRESS_PACKETS 0
#define STAT_INGRESS_BYTES   1
#define STAT_EGRESS_PACKETS  2
#define STAT_EGRESS_BYTES    3

// Load balancing configuration
#define MAX_BACKENDS_PER_PORT 4  // Limited to 4 to avoid clang complexity issues

#define LB_ALGORITHM_ROUND_ROBIN 0
#define LB_ALGORITHM_LEAST_CONN  1
#define LB_ALGORITHM_IP_HASH     2

// Backend: maps (protocol, host_port, backend_index) -> backend details
struct backend_key {
	__u8 proto;
	__u8 pad;
	__be16 port;
	__u32 backend_idx;
};

struct backend_value {
	__be32 dst_ip;
	__be16 dst_port;
	__u8 health_status; // 0=unhealthy, 1=healthy
	__u8 pad;
	__u64 conn_count;   // For least-conn algorithm
};

struct {
	__uint(type, BPF_MAP_TYPE_HASH);
	__uint(max_entries, 4096 * MAX_BACKENDS_PER_PORT);
	__type(key, struct backend_key);
	__type(value, struct backend_value);
} backends SEC(".maps");

// Backend config: maps (protocol, host_port) -> load balancing config
struct backend_config_key {
	__u8 proto;
	__u8 pad;
	__be16 port;
};

struct backend_config_value {
	__u32 backend_count;
	__u32 lb_algorithm;
	__u32 next_backend; // For round-robin
};

struct {
	__uint(type, BPF_MAP_TYPE_HASH);
	__uint(max_entries, 4096);
	__type(key, struct backend_config_key);
	__type(value, struct backend_config_value);
} backend_config SEC(".maps");

static __always_inline void update_stats(__u32 idx, __u64 bytes)
{
	__u64 *count = bpf_map_lookup_elem(&stats, &idx);
	if (count)
		__sync_fetch_and_add(count, bytes);
}

// Select backend using round-robin algorithm
static __always_inline __u32 select_backend_rr(struct backend_config_value *config)
{
	// BPF doesn't allow direct use of XADD return value
	// Must store to variable first, then use it
	__u32 counter = __sync_fetch_and_add(&config->next_backend, 1);
	__u32 idx = counter % config->backend_count;
	return idx;
}

// Select backend using least-connections algorithm
static __always_inline __u32 select_backend_least_conn(__u8 proto, __be16 port, __u32 backend_count)
{
	__u32 min_idx = 0;
	__u64 min_conns = 0xFFFFFFFFFFFFFFFF;

	struct backend_key key = {
		.proto = proto,
		.port = port,
		.backend_idx = 0,
	};

	// Check all backends and find the one with least connections
	struct backend_value *b;

	key.backend_idx = 0;
	b = bpf_map_lookup_elem(&backends, &key);
	if (b && b->health_status == 1 && b->conn_count < min_conns) {
		min_conns = b->conn_count;
		min_idx = 0;
	}

	if (backend_count > 1) {
		key.backend_idx = 1;
		b = bpf_map_lookup_elem(&backends, &key);
		if (b && b->health_status == 1 && b->conn_count < min_conns) {
			min_conns = b->conn_count;
			min_idx = 1;
		}
	}

	if (backend_count > 2) {
		key.backend_idx = 2;
		b = bpf_map_lookup_elem(&backends, &key);
		if (b && b->health_status == 1 && b->conn_count < min_conns) {
			min_conns = b->conn_count;
			min_idx = 2;
		}
	}

	if (backend_count > 3) {
		key.backend_idx = 3;
		b = bpf_map_lookup_elem(&backends, &key);
		if (b && b->health_status == 1 && b->conn_count < min_conns) {
			min_conns = b->conn_count;
			min_idx = 3;
		}
	}

	return min_idx;
}

// Select backend using IP hash algorithm (consistent hashing)
static __always_inline __u32 select_backend_ip_hash(__be32 src_ip, __u32 backend_count)
{
	// Simple hash: use client IP as seed
	__u32 hash = (__u32)src_ip;
	hash ^= (hash >> 16);
	hash *= 0x85ebca6b;
	hash ^= (hash >> 13);
	hash *= 0xc2b2ae35;
	hash ^= (hash >> 16);

	return hash % backend_count;
}

// Look up backend for load balancing
static __always_inline struct backend_value *lookup_backend(__u8 proto, __be16 port, __be32 src_ip, __u32 lb_algorithm, struct backend_config_value *config)
{
	struct backend_key key = {
		.proto = proto,
		.port = port,
		.backend_idx = 0,
	};

	__u32 selected_idx = 0;
	if (lb_algorithm == LB_ALGORITHM_ROUND_ROBIN) {
		selected_idx = select_backend_rr(config);
	} else if (lb_algorithm == LB_ALGORITHM_LEAST_CONN) {
		selected_idx = select_backend_least_conn(proto, port, config->backend_count);
	} else if (lb_algorithm == LB_ALGORITHM_IP_HASH) {
		selected_idx = select_backend_ip_hash(src_ip, config->backend_count);
	}

	// Clamp to valid range
	if (selected_idx >= config->backend_count)
		selected_idx = 0;

	// Try selected backend
	key.backend_idx = selected_idx;
	struct backend_value *backend = bpf_map_lookup_elem(&backends, &key);
	if (backend && backend->health_status == 1)
		return backend;

	// Simple fallback - try backend 0
	if (selected_idx != 0) {
		key.backend_idx = 0;
		backend = bpf_map_lookup_elem(&backends, &key);
		if (backend && backend->health_status == 1)
			return backend;
	}

	return NULL;
}

static __always_inline int rewrite_tcp(struct __sk_buff *skb, struct iphdr *iph, struct tcphdr *tcph, __be32 new_ip, __be16 new_port, __u32 l3_off, __u32 l4_off)
{
	__be16 old_port = tcph->dest;
	__be32 old_ip = iph->daddr;

	if (bpf_l4_csum_replace(skb, l4_off + offsetof(struct tcphdr, check), old_port, new_port, sizeof(new_port)))
		return TC_ACT_OK;

	if (bpf_l4_csum_replace(skb, l4_off + offsetof(struct tcphdr, check), old_ip, new_ip, sizeof(new_ip) | BPF_F_PSEUDO_HDR))
		return TC_ACT_OK;

	if (bpf_l3_csum_replace(skb, l3_off + offsetof(struct iphdr, check), old_ip, new_ip, sizeof(new_ip)))
		return TC_ACT_OK;

	if (bpf_skb_store_bytes(skb, l4_off + offsetof(struct tcphdr, dest), &new_port, sizeof(new_port), 0))
		return TC_ACT_OK;
	if (bpf_skb_store_bytes(skb, l3_off + offsetof(struct iphdr, daddr), &new_ip, sizeof(new_ip), 0))
		return TC_ACT_OK;

	return TC_ACT_OK;
}

static __always_inline int rewrite_udp(struct __sk_buff *skb, struct iphdr *iph, struct udphdr *udph, __be32 new_ip, __be16 new_port, __u32 l3_off, __u32 l4_off)
{
	__be16 old_port = udph->dest;
	__be32 old_ip = iph->daddr;

	if (udph->check) {
		if (bpf_l4_csum_replace(skb, l4_off + offsetof(struct udphdr, check), old_port, new_port, sizeof(new_port)))
			return TC_ACT_OK;
		if (bpf_l4_csum_replace(skb, l4_off + offsetof(struct udphdr, check), old_ip, new_ip, sizeof(new_ip) | BPF_F_PSEUDO_HDR))
			return TC_ACT_OK;
	}

	if (bpf_l3_csum_replace(skb, l3_off + offsetof(struct iphdr, check), old_ip, new_ip, sizeof(new_ip)))
		return TC_ACT_OK;

	if (bpf_skb_store_bytes(skb, l4_off + offsetof(struct udphdr, dest), &new_port, sizeof(new_port), 0))
		return TC_ACT_OK;
	if (bpf_skb_store_bytes(skb, l3_off + offsetof(struct iphdr, daddr), &new_ip, sizeof(new_ip), 0))
		return TC_ACT_OK;

	return TC_ACT_OK;
}

SEC("tc")
int drift_l4_ingress(struct __sk_buff *skb)
{
	void *data = (void *)(long)skb->data;
	void *data_end = (void *)(long)skb->data_end;

	struct ethhdr *eth = data;
	if ((void *)(eth + 1) > data_end)
		return TC_ACT_OK;

	if (eth->h_proto != bpf_htons(ETH_P_IP))
		return TC_ACT_OK;

	struct iphdr *iph = (struct iphdr *)(eth + 1);
	if ((void *)(iph + 1) > data_end)
		return TC_ACT_OK;

	if (iph->ihl < 5)
		return TC_ACT_OK;

	__u8 proto = iph->protocol;
	__u32 ihl_bytes = iph->ihl * 4;
	void *l4 = (void *)iph + ihl_bytes;
	if (l4 + sizeof(__be16) > data_end)
		return TC_ACT_OK;

	__u32 l3_off = (void *)iph - data;
	__u32 l4_off = (void *)l4 - data;

	if (proto == IPPROTO_TCP) {
		struct tcphdr *tcph = l4;
		if ((void *)(tcph + 1) > data_end)
			return TC_ACT_OK;

		__be32 dst_ip = 0;
		__be16 dst_port = 0;

		// Try simple portmap first (single backend, backwards compatible)
		struct portmap_key pm_key = {
			.proto = proto,
			.port = tcph->dest,
		};
		struct portmap_value *pm_value = bpf_map_lookup_elem(&portmap, &pm_key);
		if (pm_value) {
			dst_ip = pm_value->dst_ip;
			dst_port = pm_value->dst_port;
		} else {
			// Try load balancing
			struct backend_config_key bc_key = {
				.proto = proto,
				.port = tcph->dest,
			};
			struct backend_config_value *bc_value = bpf_map_lookup_elem(&backend_config, &bc_key);
			if (!bc_value || bc_value->backend_count == 0)
				return TC_ACT_OK; // No backends configured

			// Select backend using configured algorithm
			struct backend_value *backend = lookup_backend(proto, tcph->dest, iph->saddr, bc_value->lb_algorithm, bc_value);
			if (!backend)
				return TC_ACT_OK; // No healthy backend found

			dst_ip = backend->dst_ip;
			dst_port = backend->dst_port;

			// Increment connection count for least-conn algorithm
			if (bc_value->lb_algorithm == LB_ALGORITHM_LEAST_CONN)
				__sync_fetch_and_add(&backend->conn_count, 1);
		}

		// Track connection for reverse NAT
		struct conntrack_key ct_key = {
			.src_ip = iph->saddr,
			.dst_ip = dst_ip,
			.src_port = tcph->source,
			.dst_port = dst_port,
			.proto = proto,
		};
		struct conntrack_value ct_val = {
			.orig_dst_ip = iph->daddr,
			.orig_dst_port = tcph->dest,
			.last_seen = bpf_ktime_get_ns(),
		};
		bpf_map_update_elem(&conntrack, &ct_key, &ct_val, BPF_ANY);

		// Update metrics
		update_stats(STAT_INGRESS_PACKETS, 1);
		update_stats(STAT_INGRESS_BYTES, skb->len);

		return rewrite_tcp(skb, iph, tcph, dst_ip, dst_port, l3_off, l4_off);
	}

	if (proto == IPPROTO_UDP) {
		struct udphdr *udph = l4;
		if ((void *)(udph + 1) > data_end)
			return TC_ACT_OK;

		__be32 dst_ip = 0;
		__be16 dst_port = 0;

		// Try simple portmap first (single backend, backwards compatible)
		struct portmap_key pm_key = {
			.proto = proto,
			.port = udph->dest,
		};
		struct portmap_value *pm_value = bpf_map_lookup_elem(&portmap, &pm_key);
		if (pm_value) {
			dst_ip = pm_value->dst_ip;
			dst_port = pm_value->dst_port;
		} else {
			// Try load balancing
			struct backend_config_key bc_key = {
				.proto = proto,
				.port = udph->dest,
			};
			struct backend_config_value *bc_value = bpf_map_lookup_elem(&backend_config, &bc_key);
			if (!bc_value || bc_value->backend_count == 0)
				return TC_ACT_OK; // No backends configured

			// Select backend using configured algorithm
			struct backend_value *backend = lookup_backend(proto, udph->dest, iph->saddr, bc_value->lb_algorithm, bc_value);
			if (!backend)
				return TC_ACT_OK; // No healthy backend found

			dst_ip = backend->dst_ip;
			dst_port = backend->dst_port;

			// Increment connection count for least-conn algorithm
			if (bc_value->lb_algorithm == LB_ALGORITHM_LEAST_CONN)
				__sync_fetch_and_add(&backend->conn_count, 1);
		}

		// Track connection for reverse NAT
		struct conntrack_key ct_key = {
			.src_ip = iph->saddr,
			.dst_ip = dst_ip,
			.src_port = udph->source,
			.dst_port = dst_port,
			.proto = proto,
		};
		struct conntrack_value ct_val = {
			.orig_dst_ip = iph->daddr,
			.orig_dst_port = udph->dest,
			.last_seen = bpf_ktime_get_ns(),
		};
		bpf_map_update_elem(&conntrack, &ct_key, &ct_val, BPF_ANY);

		// Update metrics
		update_stats(STAT_INGRESS_PACKETS, 1);
		update_stats(STAT_INGRESS_BYTES, skb->len);

		return rewrite_udp(skb, iph, udph, dst_ip, dst_port, l3_off, l4_off);
	}

	return TC_ACT_OK;
}

// Egress handler: rewrites reply packets from VMs back to clients
SEC("tc")
int drift_l4_egress(struct __sk_buff *skb)
{
	void *data = (void *)(long)skb->data;
	void *data_end = (void *)(long)skb->data_end;

	struct ethhdr *eth = data;
	if ((void *)(eth + 1) > data_end)
		return TC_ACT_OK;

	if (eth->h_proto != bpf_htons(ETH_P_IP))
		return TC_ACT_OK;

	struct iphdr *iph = (struct iphdr *)(eth + 1);
	if ((void *)(iph + 1) > data_end)
		return TC_ACT_OK;

	if (iph->ihl < 5)
		return TC_ACT_OK;

	__u8 proto = iph->protocol;
	__u32 ihl_bytes = iph->ihl * 4;
	void *l4 = (void *)iph + ihl_bytes;
	if (l4 + sizeof(__be16) > data_end)
		return TC_ACT_OK;

	__u32 l3_off = (void *)iph - data;
	__u32 l4_off = (void *)l4 - data;

	if (proto == IPPROTO_TCP) {
		struct tcphdr *tcph = l4;
		if ((void *)(tcph + 1) > data_end)
			return TC_ACT_OK;

		// Lookup connection in conntrack
		// Reply packet has: src=VM, dst=Client (reverse of original)
		struct conntrack_key ct_key = {
			.src_ip = iph->daddr,    // Client IP (dest in reply)
			.dst_ip = iph->saddr,    // VM IP (src in reply)
			.src_port = tcph->dest,  // Client port
			.dst_port = tcph->source, // VM port
			.proto = proto,
		};

		struct conntrack_value *ct_val = bpf_map_lookup_elem(&conntrack, &ct_key);
		if (!ct_val)
			return TC_ACT_OK; // No conntrack entry, pass through

		// Rewrite source to original dest (host IP:port)
		__be32 new_src_ip = ct_val->orig_dst_ip;
		__be16 new_src_port = ct_val->orig_dst_port;

		// Rewrite packet
		__be16 old_port = tcph->source;
		__be32 old_ip = iph->saddr;

		// Update TCP checksum
		if (bpf_l4_csum_replace(skb, l4_off + offsetof(struct tcphdr, check), old_port, new_src_port, sizeof(new_src_port)))
			return TC_ACT_OK;
		if (bpf_l4_csum_replace(skb, l4_off + offsetof(struct tcphdr, check), old_ip, new_src_ip, sizeof(new_src_ip) | BPF_F_PSEUDO_HDR))
			return TC_ACT_OK;

		// Update IP checksum
		if (bpf_l3_csum_replace(skb, l3_off + offsetof(struct iphdr, check), old_ip, new_src_ip, sizeof(new_src_ip)))
			return TC_ACT_OK;

		// Write new values
		if (bpf_skb_store_bytes(skb, l4_off + offsetof(struct tcphdr, source), &new_src_port, sizeof(new_src_port), 0))
			return TC_ACT_OK;
		if (bpf_skb_store_bytes(skb, l3_off + offsetof(struct iphdr, saddr), &new_src_ip, sizeof(new_src_ip), 0))
			return TC_ACT_OK;

		// Update metrics
		update_stats(STAT_EGRESS_PACKETS, 1);
		update_stats(STAT_EGRESS_BYTES, skb->len);

		return TC_ACT_OK;
	}

	if (proto == IPPROTO_UDP) {
		struct udphdr *udph = l4;
		if ((void *)(udph + 1) > data_end)
			return TC_ACT_OK;

		// Lookup connection in conntrack
		struct conntrack_key ct_key = {
			.src_ip = iph->daddr,    // Client IP
			.dst_ip = iph->saddr,    // VM IP
			.src_port = udph->dest,  // Client port
			.dst_port = udph->source, // VM port
			.proto = proto,
		};

		struct conntrack_value *ct_val = bpf_map_lookup_elem(&conntrack, &ct_key);
		if (!ct_val)
			return TC_ACT_OK;

		// Rewrite source to original dest
		__be32 new_src_ip = ct_val->orig_dst_ip;
		__be16 new_src_port = ct_val->orig_dst_port;

		// Rewrite packet
		__be16 old_port = udph->source;
		__be32 old_ip = iph->saddr;

		// Update UDP checksum (if present)
		if (udph->check) {
			if (bpf_l4_csum_replace(skb, l4_off + offsetof(struct udphdr, check), old_port, new_src_port, sizeof(new_src_port)))
				return TC_ACT_OK;
			if (bpf_l4_csum_replace(skb, l4_off + offsetof(struct udphdr, check), old_ip, new_src_ip, sizeof(new_src_ip) | BPF_F_PSEUDO_HDR))
				return TC_ACT_OK;
		}

		// Update IP checksum
		if (bpf_l3_csum_replace(skb, l3_off + offsetof(struct iphdr, check), old_ip, new_src_ip, sizeof(new_src_ip)))
			return TC_ACT_OK;

		// Write new values
		if (bpf_skb_store_bytes(skb, l4_off + offsetof(struct udphdr, source), &new_src_port, sizeof(new_src_port), 0))
			return TC_ACT_OK;
		if (bpf_skb_store_bytes(skb, l3_off + offsetof(struct iphdr, saddr), &new_src_ip, sizeof(new_src_ip), 0))
			return TC_ACT_OK;

		// Update metrics
		update_stats(STAT_EGRESS_PACKETS, 1);
		update_stats(STAT_EGRESS_BYTES, skb->len);

		return TC_ACT_OK;
	}

	return TC_ACT_OK;
}

char LICENSE[] SEC("license") = "Dual BSD/GPL";
