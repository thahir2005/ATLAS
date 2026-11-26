package routing

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestParsePeerSummary(t *testing.T) {
	input := `{
		"ipv4Unicast": {
			"peers": {
				"172.20.12.2": {
					"state": "Established",
					"localAs": 65001,
					"remoteAs": 65002,
					"peerUptime": "00:22:53",
					"pfxRcd": 2
				}
			}
		}
	}`

	peers, err := ParsePeerSummary([]byte(input))
	if err != nil {
		t.Fatal(err)
	}

	peer, exists := peers["172.20.12.2"]
	if !exists {
		t.Fatal("expected Mumbai-Singapore peer")
	}

	if peer.State != "Established" {
		t.Fatalf("expected Established, got %s", peer.State)
	}

	if peer.LocalAS != 65001 || peer.RemoteAS != 65002 {
		t.Fatalf("unexpected AS numbers: %+v", peer)
	}

	if peer.PrefixesReceived != 2 {
		t.Fatalf("expected two received prefixes, got %d", peer.PrefixesReceived)
	}
}

func TestParseBestPath(t *testing.T) {
	input := `{
		"paths": [
			{
				"aspath": {"string": "65003 65002"},
				"peer": {"peerId": "172.20.31.1"},
				"nexthops": [
					{"ip": "172.20.31.1", "accessible": true, "used": true}
				]
			},
			{
				"aspath": {"string": "65002"},
				"peer": {"peerId": "172.20.12.2"},
				"bestpath": {"overall": true},
				"nexthops": [
					{"ip": "172.20.12.2", "accessible": true, "used": true}
				]
			}
		]
	}`

	best, err := ParseBestPath([]byte(input))
	if err != nil {
		t.Fatal(err)
	}

	if best == nil {
		t.Fatal("expected a best path")
	}

	if best.Peer != "172.20.12.2" {
		t.Fatalf("unexpected best peer: %s", best.Peer)
	}

	if best.ASPath != "65002" {
		t.Fatalf("unexpected AS path: %s", best.ASPath)
	}

	if len(best.NextHops) != 1 || best.NextHops[0].IP != "172.20.12.2" {
		t.Fatalf("unexpected next hops: %+v", best.NextHops)
	}
}

func TestParseBestPathWhenNoBestPathExists(t *testing.T) {
	best, err := ParseBestPath([]byte(`{"paths":[]}`))
	if err != nil {
		t.Fatal(err)
	}

	if best != nil {
		t.Fatalf("expected no best path, got %+v", best)
	}
}

func TestParseKernelRoute(t *testing.T) {
	output := "10.10.2.1 via 172.20.12.2 dev ms0 src 172.20.12.1 uid 0 cache"

	route, err := ParseKernelRoute("10.10.2.1", output)
	if err != nil {
		t.Fatal(err)
	}

	if route.NextHop != "172.20.12.2" {
		t.Fatalf("unexpected next hop: %s", route.NextHop)
	}

	if route.Interface != "ms0" {
		t.Fatalf("unexpected interface: %s", route.Interface)
	}

	if route.Source != "172.20.12.1" {
		t.Fatalf("unexpected source: %s", route.Source)
	}
}

func TestParseKernelRouteRejectsEmptyOutput(t *testing.T) {
	_, err := ParseKernelRoute("10.10.2.1", "")
	if err == nil {
		t.Fatal("expected error for empty route output")
	}
}

func TestParsePeerSummaryRejectsInvalidJSON(t *testing.T) {
	_, err := ParsePeerSummary([]byte("not-json"))
	if err == nil {
		t.Fatal("expected JSON parsing error")
	}
}

type fakeRunner struct {
	responses map[string][]byte
	errors    map[string]error
}

func (f fakeRunner) Run(
	_ context.Context,
	name string,
	args ...string,
) ([]byte, error) {
	key := name + " " + strings.Join(args, " ")

	if err := f.errors[key]; err != nil {
		return nil, err
	}

	output, exists := f.responses[key]
	if !exists {
		return nil, errors.New("unexpected command: " + key)
	}

	return output, nil
}

func TestObserverSnapshot(t *testing.T) {
	runner := fakeRunner{
		responses: map[string][]byte{
			"vtysh -N atlas-mumbai -c show ip bgp summary json": []byte(`{
				"ipv4Unicast": {
					"peers": {
						"172.20.12.2": {
							"state": "Established",
							"localAs": 65001,
							"remoteAs": 65002,
							"pfxRcd": 2
						}
					}
				}
			}`),
			"vtysh -N atlas-mumbai -c show ip bgp 10.10.2.1/32 json": []byte(`{
				"paths": [{
					"aspath": {"string": "65002"},
					"bestpath": {"overall": true},
					"peer": {"peerId": "172.20.12.2"},
					"nexthops": [{
						"ip": "172.20.12.2",
						"accessible": true,
						"used": true
					}]
				}]
			}`),
			"ip -n atlas-mumbai -o route get 10.10.2.1": []byte(
				"10.10.2.1 via 172.20.12.2 dev ms0 src 172.20.12.1",
			),
		},
	}

	observer := Observer{Runner: runner}

	snapshot, err := observer.Snapshot(
		context.Background(),
		"atlas-mumbai",
		"10.10.2.1/32",
		"10.10.2.1",
	)
	if err != nil {
		t.Fatal(err)
	}

	if snapshot.Peers["172.20.12.2"].State != "Established" {
		t.Fatal("peer state was not captured")
	}

	if snapshot.BestPath == nil || snapshot.BestPath.Peer != "172.20.12.2" {
		t.Fatal("best path was not captured")
	}

	if snapshot.KernelRoute == nil || snapshot.KernelRoute.Interface != "ms0" {
		t.Fatal("kernel route was not captured")
	}
}

func TestObserverRecordsMissingKernelRoute(t *testing.T) {
	runner := fakeRunner{
		responses: map[string][]byte{
			"vtysh -N atlas-mumbai -c show ip bgp summary json": []byte(
				`{"ipv4Unicast":{"peers":{}}}`,
			),
			"vtysh -N atlas-mumbai -c show ip bgp 10.10.2.1/32 json": []byte(
				`{"paths":[]}`,
			),
		},
		errors: map[string]error{
			"ip -n atlas-mumbai -o route get 10.10.2.1": errors.New("route unavailable"),
		},
	}

	observer := Observer{Runner: runner}

	snapshot, err := observer.Snapshot(
		context.Background(),
		"atlas-mumbai",
		"10.10.2.1/32",
		"10.10.2.1",
	)
	if err != nil {
		t.Fatal(err)
	}

	if snapshot.KernelRoute != nil {
		t.Fatal("expected missing kernel route")
	}

	if snapshot.KernelRouteError == "" {
		t.Fatal("expected kernel route error")
	}
}
