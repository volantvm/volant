#include "vmlinux.h"
#include <bpf/bpf_endian.h>
#include <bpf/bpf_helpers.h>

#ifndef XDP_PASS
#define XDP_PASS 2
#endif

#ifndef ETH_P_IP
#define ETH_P_IP 0x0800
#define ETH_HLEN 14
#define IP_HLEN 20
#endif

// Portmap: maps (protocol, host_port) -> (backend_ip, backend_port)
struct portmap_key {
	__u8 proto;
	__u8 pad;
	__be16 port;
};

struct portmap_value {
	__be32 dst_ip;   // Stored in network byte order
	__be16 dst_port; // Stored in network byte order
	__u16 pad;
};

struct {
	__uint(type, BPF_MAP_TYPE_HASH);
	__uint(max_entries, 4096);
	__type(key, struct portmap_key);
	__type(value, struct portmap_value);
} portmap SEC(".maps");

// Conntrack: tracks active connections for reverse NAT
struct conntrack_key {
	__be32 src_ip;   // Client IP (network byte order)
	__be32 dst_ip;   // Backend IP (network byte order)
	__be16 src_port; // Client port (network byte order)
	__be16 dst_port; // Backend port (network byte order)
	__u8 proto;
	__u8 pad[3];
};

struct conntrack_value {
	__be32 orig_dst_ip;   // Original host IP (network byte order)
	__be16 orig_dst_port; // Original host port (network byte order)
	__u16 pad;
	__u64 last_seen;
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

static __always_inline void update_stats(__u32 idx, __u64 bytes)
{
	__u64 *count = bpf_map_lookup_elem(&stats, &idx);
	if (count)
		__sync_fetch_and_add(count, bytes);
}

static __always_inline __u16 bpf_csum_fold(__u32 csum)
{
	csum = (csum & 0xffff) + (csum >> 16);
	csum = (csum & 0xffff) + (csum >> 16);
	return (__u16)~csum;
}

static __always_inline __u32 bpf_csum_unfold(__u16 csum)
{
	return (__u32)csum;
}

// XDP ingress: rewrites destination for incoming packets
SEC("xdp")
int drift_l4_ingress(struct xdp_md *ctx)
{
	void *data = (void *)(long)ctx->data;
	void *data_end = (void *)(long)ctx->data_end;

	struct ethhdr *eth = data;
	if ((void *)(eth + 1) > data_end)
		return XDP_PASS;

	if (eth->h_proto != bpf_htons(ETH_P_IP))
		return XDP_PASS;

	struct iphdr *iph = (struct iphdr *)(eth + 1);
	if ((void *)(iph + 1) > data_end)
		return XDP_PASS;

	if (iph->ihl < 5)
		return XDP_PASS;

	__u8 proto = iph->protocol;

	if (proto == IPPROTO_TCP) {
		// Use struct pointer arithmetic (assumes no IP options)
		struct tcphdr *tcph = (struct tcphdr *)(iph + 1);
		if ((void *)(tcph + 1) > data_end)
			return XDP_PASS;

		// Lookup port mapping
		struct portmap_key key = {
			.proto = proto,
			.port = tcph->dest,
		};

		struct portmap_value *val = bpf_map_lookup_elem(&portmap, &key);
		if (!val)
			return XDP_PASS;

		// Store original destination for conntrack
		__be32 orig_dst_ip = iph->daddr;
		__be16 orig_dst_port = tcph->dest;

		// Rewrite destination
		__be32 new_dst_ip = val->dst_ip;
		__be16 new_dst_port = val->dst_port;

		// Save old values for checksum calculation
		__be32 old_dst_ip = iph->daddr;
		__be16 old_dst_port = tcph->dest;

		// Update TCP checksum using incremental update
		__u32 csum_diff = bpf_csum_diff(&old_dst_ip, sizeof(old_dst_ip), &new_dst_ip, sizeof(new_dst_ip), 0);
		csum_diff = bpf_csum_diff(&old_dst_port, sizeof(old_dst_port), &new_dst_port, sizeof(new_dst_port), csum_diff);
		tcph->check = bpf_csum_fold(bpf_csum_unfold(tcph->check) + csum_diff);

		// Update IP checksum using incremental update
		csum_diff = bpf_csum_diff(&old_dst_ip, sizeof(old_dst_ip), &new_dst_ip, sizeof(new_dst_ip), 0);
		iph->check = bpf_csum_fold(bpf_csum_unfold(iph->check) + csum_diff);

		// Rewrite packet
		iph->daddr = new_dst_ip;
		tcph->dest = new_dst_port;

		// Create conntrack entry
		struct conntrack_key ct_key = {
			.src_ip = iph->saddr,
			.dst_ip = new_dst_ip,
			.src_port = tcph->source,
			.dst_port = new_dst_port,
			.proto = proto,
		};

		struct conntrack_value ct_val = {
			.orig_dst_ip = orig_dst_ip,
			.orig_dst_port = orig_dst_port,
			.last_seen = bpf_ktime_get_ns(),
		};

		bpf_map_update_elem(&conntrack, &ct_key, &ct_val, BPF_ANY);

		update_stats(STAT_INGRESS_PACKETS, 1);
		// Skip byte counting for now - verifier rejects pointer arithmetic
		// update_stats(STAT_INGRESS_BYTES, packet_size);

		return XDP_PASS; // Let kernel route the rewritten packet

	} else if (proto == IPPROTO_UDP) {
		// Use struct pointer arithmetic (assumes no IP options)
		struct udphdr *udph = (struct udphdr *)(iph + 1);
		if ((void *)(udph + 1) > data_end)
			return XDP_PASS;

		// Lookup port mapping
		struct portmap_key key = {
			.proto = proto,
			.port = udph->dest,
		};

		struct portmap_value *val = bpf_map_lookup_elem(&portmap, &key);
		if (!val)
			return XDP_PASS;

		// Store original destination
		__be32 orig_dst_ip = iph->daddr;
		__be16 orig_dst_port = udph->dest;

		// Rewrite destination
		__be32 new_dst_ip = val->dst_ip;
		__be16 new_dst_port = val->dst_port;

		// Save old values for checksum calculation
		__be32 old_dst_ip = iph->daddr;
		__be16 old_dst_port = udph->dest;

		// Update UDP checksum if present
		if (udph->check) {
			__u32 csum_diff = bpf_csum_diff(&old_dst_ip, sizeof(old_dst_ip), &new_dst_ip, sizeof(new_dst_ip), 0);
			csum_diff = bpf_csum_diff(&old_dst_port, sizeof(old_dst_port), &new_dst_port, sizeof(new_dst_port), csum_diff);
			udph->check = bpf_csum_fold(bpf_csum_unfold(udph->check) + csum_diff);
		}

		// Update IP checksum using incremental update
		__u32 csum_diff = bpf_csum_diff(&old_dst_ip, sizeof(old_dst_ip), &new_dst_ip, sizeof(new_dst_ip), 0);
		iph->check = bpf_csum_fold(bpf_csum_unfold(iph->check) + csum_diff);

		// Rewrite packet
		iph->daddr = new_dst_ip;
		udph->dest = new_dst_port;

		// Create conntrack entry
		struct conntrack_key ct_key = {
			.src_ip = iph->saddr,
			.dst_ip = new_dst_ip,
			.src_port = udph->source,
			.dst_port = new_dst_port,
			.proto = proto,
		};

		struct conntrack_value ct_val = {
			.orig_dst_ip = orig_dst_ip,
			.orig_dst_port = orig_dst_port,
			.last_seen = bpf_ktime_get_ns(),
		};

		bpf_map_update_elem(&conntrack, &ct_key, &ct_val, BPF_ANY);

		update_stats(STAT_INGRESS_PACKETS, 1);
		// Skip byte counting for now - verifier rejects pointer arithmetic
		// update_stats(STAT_INGRESS_BYTES, packet_size);

		return XDP_PASS;
	}

	return XDP_PASS;
}

char _license[] SEC("license") = "GPL";
