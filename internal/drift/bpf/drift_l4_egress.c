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

// Conntrack: shared with ingress
struct conntrack_key {
	__be32 src_ip;
	__be32 dst_ip;
	__be16 src_port;
	__be16 dst_port;
	__u8 proto;
	__u8 pad[3];
};

struct conntrack_value {
	__be32 orig_dst_ip;
	__be16 orig_dst_port;
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

#define STAT_EGRESS_PACKETS  2
#define STAT_EGRESS_BYTES    3

static __always_inline void update_stats(__u32 idx, __u64 bytes)
{
	__u64 *count = bpf_map_lookup_elem(&stats, &idx);
	if (count)
		__sync_fetch_and_add(count, bytes);
}

// TC egress: rewrites source for return packets
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
	const __u32 l3_off = ETH_HLEN;
	const __u32 l4_off = ETH_HLEN + IP_HLEN;

	if (proto == IPPROTO_TCP) {
		void *l4 = data + l4_off;
		if (l4 + sizeof(struct tcphdr) > data_end)
			return TC_ACT_OK;

		struct tcphdr *tcph = l4;

		// Lookup conntrack: reply has src=VM, dst=Client
		struct conntrack_key ct_key = {
			.src_ip = iph->daddr,    // Client IP (destination in reply)
			.dst_ip = iph->saddr,    // VM IP (source in reply)
			.src_port = tcph->dest,  // Client port
			.dst_port = tcph->source, // VM port
			.proto = proto,
		};

		struct conntrack_value *ct_val = bpf_map_lookup_elem(&conntrack, &ct_key);
		if (!ct_val)
			return TC_ACT_OK;

		// Rewrite source to original destination (host IP:port)
		__be32 new_src_ip = ct_val->orig_dst_ip;
		__be16 new_src_port = ct_val->orig_dst_port;

		__be16 old_port = tcph->source;
		__be32 old_ip = iph->saddr;

		// Update TCP checksum
		if (bpf_l4_csum_replace(skb, l4_off + offsetof(struct tcphdr, check),
		                        old_port, new_src_port, sizeof(new_src_port)))
			return TC_ACT_OK;

		if (bpf_l4_csum_replace(skb, l4_off + offsetof(struct tcphdr, check),
		                        old_ip, new_src_ip, sizeof(new_src_ip) | BPF_F_PSEUDO_HDR))
			return TC_ACT_OK;

		// Update IP checksum
		if (bpf_l3_csum_replace(skb, l3_off + offsetof(struct iphdr, check),
		                        old_ip, new_src_ip, sizeof(new_src_ip)))
			return TC_ACT_OK;

		// Write new values
		if (bpf_skb_store_bytes(skb, l4_off + offsetof(struct tcphdr, source),
		                        &new_src_port, sizeof(new_src_port), 0))
			return TC_ACT_OK;

		if (bpf_skb_store_bytes(skb, l3_off + offsetof(struct iphdr, saddr),
		                        &new_src_ip, sizeof(new_src_ip), 0))
			return TC_ACT_OK;

		update_stats(STAT_EGRESS_PACKETS, 1);
		update_stats(STAT_EGRESS_BYTES, skb->len);

	} else if (proto == IPPROTO_UDP) {
		void *l4 = data + l4_off;
		if (l4 + sizeof(struct udphdr) > data_end)
			return TC_ACT_OK;

		struct udphdr *udph = l4;

		// Lookup conntrack
		struct conntrack_key ct_key = {
			.src_ip = iph->daddr,
			.dst_ip = iph->saddr,
			.src_port = udph->dest,
			.dst_port = udph->source,
			.proto = proto,
		};

		struct conntrack_value *ct_val = bpf_map_lookup_elem(&conntrack, &ct_key);
		if (!ct_val)
			return TC_ACT_OK;

		__be32 new_src_ip = ct_val->orig_dst_ip;
		__be16 new_src_port = ct_val->orig_dst_port;

		__be16 old_port = udph->source;
		__be32 old_ip = iph->saddr;

		// Update UDP checksum if present
		if (udph->check) {
			if (bpf_l4_csum_replace(skb, l4_off + offsetof(struct udphdr, check),
			                        old_port, new_src_port, sizeof(new_src_port)))
				return TC_ACT_OK;

			if (bpf_l4_csum_replace(skb, l4_off + offsetof(struct udphdr, check),
			                        old_ip, new_src_ip, sizeof(new_src_ip) | BPF_F_PSEUDO_HDR))
				return TC_ACT_OK;
		}

		// Update IP checksum
		if (bpf_l3_csum_replace(skb, l3_off + offsetof(struct iphdr, check),
		                        old_ip, new_src_ip, sizeof(new_src_ip)))
			return TC_ACT_OK;

		// Write new values
		if (bpf_skb_store_bytes(skb, l4_off + offsetof(struct udphdr, source),
		                        &new_src_port, sizeof(new_src_port), 0))
			return TC_ACT_OK;

		if (bpf_skb_store_bytes(skb, l3_off + offsetof(struct iphdr, saddr),
		                        &new_src_ip, sizeof(new_src_ip), 0))
			return TC_ACT_OK;

		update_stats(STAT_EGRESS_PACKETS, 1);
		update_stats(STAT_EGRESS_BYTES, skb->len);
	}

	return TC_ACT_OK;
}

char _license[] SEC("license") = "GPL";
