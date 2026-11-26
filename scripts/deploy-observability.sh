#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SSH_CONFIG="/tmp/atlas-colima-ssh-config"

cd "$ROOT"

echo "===== PREPARE COLIMA CONNECTION ====="
colima ssh-config > "$SSH_CONFIG"

echo "===== TRANSFER REPOSITORY CONFIGS ====="
scp -F "$SSH_CONFIG" observability/prometheus/prometheus.yml colima:atlas-prometheus.yml
scp -F "$SSH_CONFIG" observability/prometheus/atlas-alerts.yml colima:atlas-alerts.yml
scp -F "$SSH_CONFIG" observability/grafana/provisioning/datasources/atlas-prometheus.yml colima:atlas-grafana-datasource.yml
scp -F "$SSH_CONFIG" observability/grafana/provisioning/dashboards/atlas.yml colima:atlas-grafana-provider.yml
scp -F "$SSH_CONFIG" observability/grafana/dashboards/atlas-overview.json colima:atlas-overview.json

echo "===== DEPLOY AND VALIDATE ON COLIMA ====="
colima ssh -- sudo bash -s <<'REMOTE'
set -euo pipefail

echo "--- Check regional namespace ---"
ip netns list | grep -q '^atlas-mumbai' || {
  echo "ERROR: atlas-mumbai namespace does not exist"
  exit 1
}

echo "--- Recreate monitoring veth if missing ---"
if ! ip link show atlas-mon0 >/dev/null 2>&1; then
  if ip netns exec atlas-mumbai ip link show atlas-mon1 >/dev/null 2>&1; then
    echo "ERROR: atlas-mon1 exists but root-side atlas-mon0 is missing"
    exit 1
  fi

  ip link add atlas-mon0 type veth peer name atlas-mon1
  ip link set atlas-mon1 netns atlas-mumbai
fi

ip address replace 172.30.90.1/30 dev atlas-mon0
ip link set atlas-mon0 up

ip netns exec atlas-mumbai ip address replace 172.30.90.2/30 dev atlas-mon1
ip netns exec atlas-mumbai ip link set atlas-mon1 up
ip netns exec atlas-mumbai ip link set lo up

echo "--- Install Prometheus configuration ---"
install -D -m 0644 /home/thahir.guest/atlas-prometheus.yml /etc/prometheus/prometheus.yml
install -D -m 0644 /home/thahir.guest/atlas-alerts.yml /etc/prometheus/atlas-alerts.yml

echo "--- Install Grafana provisioning and dashboard ---"
install -D -m 0644 /home/thahir.guest/atlas-grafana-datasource.yml /etc/grafana/provisioning/datasources/atlas-prometheus.yml
install -D -m 0644 /home/thahir.guest/atlas-grafana-provider.yml /etc/grafana/provisioning/dashboards/atlas.yml
install -D -m 0644 /home/thahir.guest/atlas-overview.json /var/lib/grafana/dashboards/atlas-overview.json

echo "--- Validate configurations before restarting ---"
promtool check config /etc/prometheus/prometheus.yml
promtool check rules /etc/prometheus/atlas-alerts.yml
python3 -m json.tool /var/lib/grafana/dashboards/atlas-overview.json >/dev/null

echo "--- Restart monitoring services ---"
systemctl restart prometheus
systemctl restart grafana-server

echo "--- Verify services ---"
systemctl is-active prometheus
systemctl is-active grafana-server

echo "--- Verify monitoring network ---"
ip -brief address show atlas-mon0
ip netns exec atlas-mumbai ip -brief address show atlas-mon1

echo "--- Wait for Grafana readiness ---"
grafana_ready=0
for attempt in $(seq 1 15); do
  if curl -fsS --max-time 3 http://127.0.0.1:3000/api/health; then
    echo
    grafana_ready=1
    break
  fi
  echo "Grafana is starting (attempt $attempt/15)..."
  sleep 2
done

if [ "$grafana_ready" -ne 1 ]; then
  echo "ERROR: Grafana did not become healthy"
  systemctl status grafana-server --no-pager
  exit 1
fi

echo "OBSERVABILITY DEPLOYMENT COMPLETED"
REMOTE

echo "===== DEPLOYMENT FINISHED ====="
