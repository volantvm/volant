#include "vmlinux.h"
#include <bpf/bpf_endian.h>
#include <bpf/bpf_helpers.h>

#ifndef TC_ACT_OK
#define TC_ACT_OK 0
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
	__u32 dst_ip;    // Stored in HOST byte order (converted to network with bpf_htonl)
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

static __always_inline void update_stats(__u32 idx, __u64 bytes)
{
	__u64 *count = bpf_map_lookup_elem(&stats, &idx);
	if (count)
		__sync_fetch_and_add(count, bytes);
}

// TC ingress: rewrites destination for incoming packets
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
	const __u32 l3_off = ETH_HLEN;
	const __u32 l4_off = ETH_HLEN + IP_HLEN;

	if (proto == IPPROTO_TCP) {
		struct tcphdr *tcph = (struct tcphdr *)(iph + 1);
		if ((void *)(tcph + 1) > data_end)
			return TC_ACT_OK;

		// Lookup port mapping
		struct portmap_key key = {
			.proto = proto,
			.port = tcph->dest,
		};

		struct portmap_value *val = bpf_map_lookup_elem(&portmap, &key);
		if (!val)
			return TC_ACT_OK;

		// CRITICAL: Save all values BEFORE modifying packet
		// (packet-modifying helpers invalidate all packet pointers)
		__be32 orig_dst_ip = iph->daddr;
		__be16 orig_dst_port = tcph->dest;
		__be32 client_src_ip = iph->saddr;
		__be16 client_src_port = tcph->source;

		// Rewrite destination
		// Map stores in host byte order; on little-endian, memory bytes are already correct for packet
		__u32 new_dst_ip = val->dst_ip;
		__be16 new_dst_port = val->dst_port;

		// Update TCP checksum
		if (bpf_l4_csum_replace(skb, l4_off + offsetof(struct tcphdr, check),
		                        orig_dst_port, new_dst_port, sizeof(new_dst_port)))
			return TC_ACT_OK;

		if (bpf_l4_csum_replace(skb, l4_off + offsetof(struct tcphdr, check),
		                        orig_dst_ip, new_dst_ip, sizeof(new_dst_ip) | BPF_F_PSEUDO_HDR))
			return TC_ACT_OK;

		// Update IP checksum
		if (bpf_l3_csum_replace(skb, l3_off + offsetof(struct iphdr, check),
		                        orig_dst_ip, new_dst_ip, sizeof(new_dst_ip)))
			return TC_ACT_OK;

		// Write new values - bpf_skb_store_bytes copies memory bytes as-is
		if (bpf_skb_store_bytes(skb, l4_off + offsetof(struct tcphdr, dest),
		                        &new_dst_port, sizeof(new_dst_port), 0))
			return TC_ACT_OK;

		if (bpf_skb_store_bytes(skb, l3_off + offsetof(struct iphdr, daddr),
		                        &new_dst_ip, sizeof(new_dst_ip), 0))
			return TC_ACT_OK;

		// Create conntrack entry using saved values
		struct conntrack_key ct_key = {
			.src_ip = client_src_ip,
			.dst_ip = new_dst_ip,
			.src_port = client_src_port,
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

	} else if (proto == IPPROTO_UDP) {
		struct udphdr *udph = (struct udphdr *)(iph + 1);
		if ((void *)(udph + 1) > data_end)
			return TC_ACT_OK;

		// Lookup port mapping
		struct portmap_key key = {
			.proto = proto,
			.port = udph->dest,
		};

		struct portmap_value *val = bpf_map_lookup_elem(&portmap, &key);
		if (!val)
			return TC_ACT_OK;

		// CRITICAL: Save all values BEFORE modifying packet
		__be32 orig_dst_ip = iph->daddr;
		__be16 orig_dst_port = udph->dest;
		__be32 client_src_ip = iph->saddr;
		__be16 client_src_port = udph->source;

		// Rewrite destination
		// Map stores in host byte order; on little-endian, memory bytes are already correct for packet
		__u32 new_dst_ip = val->dst_ip;
		__be16 new_dst_port = val->dst_port;

		// Update UDP checksum if present
		if (udph->check) {
			if (bpf_l4_csum_replace(skb, l4_off + offsetof(struct udphdr, check),
			                        orig_dst_port, new_dst_port, sizeof(new_dst_port)))
				return TC_ACT_OK;

			if (bpf_l4_csum_replace(skb, l4_off + offsetof(struct udphdr, check),
			                        orig_dst_ip, new_dst_ip, sizeof(new_dst_ip) | BPF_F_PSEUDO_HDR))
				return TC_ACT_OK;
		}

		// Update IP checksum
		if (bpf_l3_csum_replace(skb, l3_off + offsetof(struct iphdr, check),
		                        orig_dst_ip, new_dst_ip, sizeof(new_dst_ip)))
			return TC_ACT_OK;

		// Write new values - bpf_skb_store_bytes copies memory bytes as-is
		if (bpf_skb_store_bytes(skb, l4_off + offsetof(struct udphdr, dest),
		                        &new_dst_port, sizeof(new_dst_port), 0))
			return TC_ACT_OK;

		if (bpf_skb_store_bytes(skb, l3_off + offsetof(struct iphdr, daddr),
		                        &new_dst_ip, sizeof(new_dst_ip), 0))
			return TC_ACT_OK;

		// Create conntrack entry using saved values
		struct conntrack_key ct_key = {
			.src_ip = client_src_ip,
			.dst_ip = new_dst_ip,
			.src_port = client_src_port,
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
	}

	return TC_ACT_OK;
}

char _license[] SEC("license") = "GPL";
