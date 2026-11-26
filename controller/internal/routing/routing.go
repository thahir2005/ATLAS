package routing

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type CommandRunner interface {
	Run(ctx context.Context, name string, args ...string) ([]byte, error)
}

type ExecRunner struct{}

func (ExecRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	output, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%s failed: %s: %w", name, strings.TrimSpace(string(output)), err)
	}
	return output, nil
}

type Peer struct {
	Address          string `json:"address"`
	State            string `json:"state"`
	LocalAS          uint32 `json:"local_as"`
	RemoteAS         uint32 `json:"remote_as"`
	PeerUptime       string `json:"peer_uptime"`
	PrefixesReceived int    `json:"prefixes_received"`
}

type NextHop struct {
	IP         string `json:"ip"`
	Accessible bool   `json:"accessible"`
	Used       bool   `json:"used"`
}

type BestPath struct {
	Peer     string    `json:"peer"`
	ASPath   string    `json:"as_path"`
	NextHops []NextHop `json:"next_hops"`
}

type KernelRoute struct {
	Target    string `json:"target"`
	NextHop   string `json:"next_hop,omitempty"`
	Interface string `json:"interface,omitempty"`
	Source    string `json:"source,omitempty"`
	Raw       string `json:"raw"`
}

type Snapshot struct {
	ObservedAt       string          `json:"observed_at"`
	Namespace        string          `json:"namespace"`
	Prefix           string          `json:"prefix"`
	Target           string          `json:"target"`
	Peers            map[string]Peer `json:"peers"`
	BestPath         *BestPath       `json:"best_path"`
	KernelRoute      *KernelRoute    `json:"kernel_route"`
	KernelRouteError string          `json:"kernel_route_error,omitempty"`
}

type Observer struct {
	Runner CommandRunner
}

func ParsePeerSummary(data []byte) (map[string]Peer, error) {
	var response struct {
		IPv4Unicast struct {
			Peers map[string]struct {
				State      string `json:"state"`
				LocalAS    uint32 `json:"localAs"`
				RemoteAS   uint32 `json:"remoteAs"`
				PeerUptime string `json:"peerUptime"`
				Prefixes   int    `json:"pfxRcd"`
			} `json:"peers"`
		} `json:"ipv4Unicast"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("decode BGP peer summary: %w", err)
	}

	peers := make(map[string]Peer)

	for address, details := range response.IPv4Unicast.Peers {
		peers[address] = Peer{
			Address:          address,
			State:            details.State,
			LocalAS:          details.LocalAS,
			RemoteAS:         details.RemoteAS,
			PeerUptime:       details.PeerUptime,
			PrefixesReceived: details.Prefixes,
		}
	}

	return peers, nil
}

func ParseBestPath(data []byte) (*BestPath, error) {
	var response struct {
		Paths []struct {
			ASPath struct {
				String string `json:"string"`
			} `json:"aspath"`
			BestPath struct {
				Overall bool `json:"overall"`
			} `json:"bestpath"`
			Peer struct {
				PeerID string `json:"peerId"`
			} `json:"peer"`
			NextHops []struct {
				IP         string `json:"ip"`
				Accessible bool   `json:"accessible"`
				Used       bool   `json:"used"`
			} `json:"nexthops"`
		} `json:"paths"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("decode BGP prefix: %w", err)
	}

	for _, path := range response.Paths {
		if !path.BestPath.Overall {
			continue
		}

		best := &BestPath{
			Peer:   path.Peer.PeerID,
			ASPath: path.ASPath.String,
		}

		for _, hop := range path.NextHops {
			best.NextHops = append(best.NextHops, NextHop{
				IP:         hop.IP,
				Accessible: hop.Accessible,
				Used:       hop.Used,
			})
		}

		return best, nil
	}

	return nil, nil
}

func ParseKernelRoute(target string, output string) (*KernelRoute, error) {
	fields := strings.Fields(output)

	if len(fields) == 0 {
		return nil, fmt.Errorf("empty kernel route output")
	}

	route := &KernelRoute{
		Target: target,
		Raw:    strings.TrimSpace(output),
	}

	for index := 0; index < len(fields)-1; index++ {
		switch fields[index] {
		case "via":
			route.NextHop = fields[index+1]
		case "dev":
			route.Interface = fields[index+1]
		case "src":
			route.Source = fields[index+1]
		}
	}

	return route, nil
}

func (o Observer) Snapshot(
	ctx context.Context,
	namespace string,
	prefix string,
	target string,
) (Snapshot, error) {
	if o.Runner == nil {
		return Snapshot{}, fmt.Errorf("command runner is required")
	}

	if namespace == "" || prefix == "" || target == "" {
		return Snapshot{}, fmt.Errorf("namespace, prefix and target are required")
	}

	snapshot := Snapshot{
		ObservedAt: time.Now().UTC().Format(time.RFC3339Nano),
		Namespace:  namespace,
		Prefix:     prefix,
		Target:     target,
	}

	peerOutput, err := o.Runner.Run(
		ctx,
		"vtysh",
		"-N", namespace,
		"-c", "show ip bgp summary json",
	)
	if err != nil {
		return Snapshot{}, fmt.Errorf("read BGP neighbors: %w", err)
	}

	snapshot.Peers, err = ParsePeerSummary(peerOutput)
	if err != nil {
		return Snapshot{}, err
	}

	prefixOutput, err := o.Runner.Run(
		ctx,
		"vtysh",
		"-N", namespace,
		"-c", "show ip bgp "+prefix+" json",
	)
	if err != nil {
		return Snapshot{}, fmt.Errorf("read BGP prefix: %w", err)
	}

	snapshot.BestPath, err = ParseBestPath(prefixOutput)
	if err != nil {
		return Snapshot{}, err
	}

	routeOutput, err := o.Runner.Run(
		ctx,
		"ip",
		"-n", namespace,
		"-o", "route", "get", target,
	)

	if err != nil {
		snapshot.KernelRouteError = err.Error()
	} else {
		snapshot.KernelRoute, err = ParseKernelRoute(target, string(routeOutput))
		if err != nil {
			snapshot.KernelRouteError = err.Error()
		}
	}

	return snapshot, nil
}
