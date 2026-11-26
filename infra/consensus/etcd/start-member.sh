#!/usr/bin/env bash
set -euo pipefail

member="${1:?Usage: atlas-etcd-member mumbai|singapore|frankfurt}"

config_dir="/etc/atlas-etcd"

source "${config_dir}/cluster.env"
source "${config_dir}/${member}.env"

exec ip netns exec "${ETCD_NAMESPACE}" \
    runuser -u etcd -- \
    env "ETCD_UNSUPPORTED_ARCH=${ETCD_UNSUPPORTED_ARCH}" \
    /usr/bin/etcd \
    --name="${ETCD_NAME}" \
    --data-dir="${ETCD_DATA_DIR}" \
    --listen-peer-urls="http://${ETCD_IP}:2380" \
    --initial-advertise-peer-urls="http://${ETCD_IP}:2380" \
    --listen-client-urls="http://${ETCD_IP}:2379" \
    --advertise-client-urls="http://${ETCD_IP}:2379" \
    --initial-cluster="${ETCD_INITIAL_CLUSTER}" \
    --initial-cluster-token="${ETCD_CLUSTER_TOKEN}" \
    --initial-cluster-state="${ETCD_CLUSTER_STATE}"
