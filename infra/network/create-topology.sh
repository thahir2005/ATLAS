#!/usr/bin/env bash
set -euo pipefail

echo "===== CLEANING PREVIOUS ATLAS TOPOLOGY ====="

for ns in atlas-mumbai atlas-singapore atlas-frankfurt; do
    ip netns delete "$ns" 2>/dev/null || true
done

echo "===== CREATING REGIONAL NAMESPACES ====="

ip netns add atlas-mumbai
ip netns add atlas-singapore
ip netns add atlas-frankfurt

for ns in atlas-mumbai atlas-singapore atlas-frankfurt; do
    ip -n "$ns" link set lo up
    ip netns exec "$ns" sysctl -qw net.ipv4.ip_forward=1
done

echo "===== CONFIGURING REGIONAL LOOPBACKS ====="

ip -n atlas-mumbai addr add 10.10.1.1/32 dev lo
ip -n atlas-singapore addr add 10.10.2.1/32 dev lo
ip -n atlas-frankfurt addr add 10.10.3.1/32 dev lo

create_link() {
    local ns1="$1"
    local if1="$2"
    local ns2="$3"
    local if2="$4"
    local tmp1="$5"
    local tmp2="$6"

    ip link add "$tmp1" type veth peer name "$tmp2"

    ip link set "$tmp1" netns "$ns1"
    ip link set "$tmp2" netns "$ns2"

    ip -n "$ns1" link set "$tmp1" name "$if1"
    ip -n "$ns2" link set "$tmp2" name "$if2"

    ip -n "$ns1" link set "$if1" up
    ip -n "$ns2" link set "$if2" up
}

echo "===== CREATING MUMBAI-SINGAPORE LINK ====="

create_link atlas-mumbai ms0 atlas-singapore ms0 veth-ms-m veth-ms-s

ip -n atlas-mumbai addr add 172.20.12.1/30 dev ms0
ip -n atlas-singapore addr add 172.20.12.2/30 dev ms0

echo "===== CREATING SINGAPORE-FRANKFURT LINK ====="

create_link atlas-singapore sf0 atlas-frankfurt sf0 veth-sf-s veth-sf-f

ip -n atlas-singapore addr add 172.20.23.1/30 dev sf0
ip -n atlas-frankfurt addr add 172.20.23.2/30 dev sf0

echo "===== CREATING FRANKFURT-MUMBAI LINK ====="

create_link atlas-frankfurt fm0 atlas-mumbai fm0 veth-fm-f veth-fm-m

ip -n atlas-frankfurt addr add 172.20.31.1/30 dev fm0
ip -n atlas-mumbai addr add 172.20.31.2/30 dev fm0

echo "===== CONFIGURING STATIC ROUTES ====="

ip -n atlas-mumbai route add 10.10.2.1/32 via 172.20.12.2 dev ms0
ip -n atlas-mumbai route add 10.10.3.1/32 via 172.20.31.1 dev fm0

ip -n atlas-singapore route add 10.10.1.1/32 via 172.20.12.1 dev ms0
ip -n atlas-singapore route add 10.10.3.1/32 via 172.20.23.2 dev sf0

ip -n atlas-frankfurt route add 10.10.1.1/32 via 172.20.31.2 dev fm0
ip -n atlas-frankfurt route add 10.10.2.1/32 via 172.20.23.1 dev sf0

echo "===== REGIONAL NETWORK INTERFACES ====="

for ns in atlas-mumbai atlas-singapore atlas-frankfurt; do
    echo "--- $ns ---"
    ip -n "$ns" -br address
done

echo "===== REGIONAL ROUTING TABLES ====="

for ns in atlas-mumbai atlas-singapore atlas-frankfurt; do
    echo "--- $ns ---"
    ip -n "$ns" route
done

echo "===== TESTING REGIONAL CONNECTIVITY ====="

ip netns exec atlas-mumbai ping -c 3 -W 2 -I 10.10.1.1 10.10.2.1
ip netns exec atlas-mumbai ping -c 3 -W 2 -I 10.10.1.1 10.10.3.1
ip netns exec atlas-singapore ping -c 3 -W 2 -I 10.10.2.1 10.10.3.1

echo "===== ATLAS REGIONAL NETWORK CREATED ====="
