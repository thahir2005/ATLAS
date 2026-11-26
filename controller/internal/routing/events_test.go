package routing

import "testing"

func TestInitialSnapshotEmitsInitialEvent(t *testing.T) {
	current := Snapshot{
		Namespace: "atlas-mumbai",
		Peers: map[string]Peer{
			"172.20.12.2": {
				Address: "172.20.12.2",
				State:   "Established",
			},
		},
	}

	events := CompareSnapshots(nil, current)

	if len(events) != 1 || events[0].Event != "INITIAL_STATE" {
		t.Fatalf("expected initial state event, got %+v", events)
	}
}

func TestPeerStateChange(t *testing.T) {
	previous := Snapshot{
		Peers: map[string]Peer{
			"172.20.12.2": {State: "Established"},
		},
	}

	current := Snapshot{
		Peers: map[string]Peer{
			"172.20.12.2": {State: "Idle"},
		},
	}

	events := CompareSnapshots(&previous, current)

	if len(events) != 1 || events[0].Event != "PEER_STATE_CHANGE" {
		t.Fatalf("expected peer state change, got %+v", events)
	}
}

func TestBestPathChange(t *testing.T) {
	previous := Snapshot{
		BestPath: &BestPath{
			Peer:   "172.20.12.2",
			ASPath: "65002",
		},
	}

	current := Snapshot{
		BestPath: &BestPath{
			Peer:   "172.20.31.1",
			ASPath: "65003 65002",
		},
	}

	events := CompareSnapshots(&previous, current)

	if len(events) != 1 || events[0].Event != "BGP_BESTPATH_CHANGE" {
		t.Fatalf("expected best-path change, got %+v", events)
	}
}

func TestKernelRouteChange(t *testing.T) {
	previous := Snapshot{
		KernelRoute: &KernelRoute{
			Target:    "10.10.2.1",
			NextHop:   "172.20.12.2",
			Interface: "ms0",
		},
	}

	current := Snapshot{
		KernelRoute: &KernelRoute{
			Target:    "10.10.2.1",
			NextHop:   "172.20.31.1",
			Interface: "fm0",
		},
	}

	events := CompareSnapshots(&previous, current)

	if len(events) != 1 || events[0].Event != "KERNEL_ROUTE_CHANGE" {
		t.Fatalf("expected kernel route change, got %+v", events)
	}
}

func TestUnchangedSnapshotEmitsNoEvents(t *testing.T) {
	previous := Snapshot{
		Peers: map[string]Peer{
			"172.20.12.2": {State: "Established"},
		},
		BestPath: &BestPath{
			Peer:   "172.20.12.2",
			ASPath: "65002",
		},
		KernelRoute: &KernelRoute{
			Target:    "10.10.2.1",
			NextHop:   "172.20.12.2",
			Interface: "ms0",
		},
	}

	current := Snapshot{
		Peers: map[string]Peer{
			"172.20.12.2": {State: "Established"},
		},
		BestPath: &BestPath{
			Peer:   "172.20.12.2",
			ASPath: "65002",
		},
		KernelRoute: &KernelRoute{
			Target:    "10.10.2.1",
			NextHop:   "172.20.12.2",
			Interface: "ms0",
		},
	}

	events := CompareSnapshots(&previous, current)

	if len(events) != 0 {
		t.Fatalf("expected no events, got %+v", events)
	}
}

func TestNewPeerDetection(t *testing.T) {
	previous := Snapshot{
		Peers: map[string]Peer{},
	}

	current := Snapshot{
		Peers: map[string]Peer{
			"172.20.12.2": {State: "Established"},
		},
	}

	events := CompareSnapshots(&previous, current)

	if len(events) != 1 || events[0].Event != "PEER_STATE_CHANGE" {
		t.Fatalf("expected new peer event, got %+v", events)
	}
}

func TestRawRouteOutputChangeDoesNotEmitEvent(t *testing.T) {
	previous := Snapshot{
		KernelRoute: &KernelRoute{
			Target:    "10.10.2.1",
			NextHop:   "172.20.12.2",
			Interface: "ms0",
			Source:    "172.20.12.1",
			Raw:       "10.10.2.1 via 172.20.12.2 dev ms0 uid 0 cache",
		},
	}

	current := Snapshot{
		KernelRoute: &KernelRoute{
			Target:    "10.10.2.1",
			NextHop:   "172.20.12.2",
			Interface: "ms0",
			Source:    "172.20.12.1",
			Raw:       "10.10.2.1 via 172.20.12.2 dev ms0 uid 1000 cache",
		},
	}

	events := CompareSnapshots(&previous, current)

	if len(events) != 0 {
		t.Fatalf("raw output change must not emit route event, got %+v", events)
	}
}

func TestKernelRouteErrorChangeEmitsEvent(t *testing.T) {
	previous := Snapshot{
		KernelRouteError: "route unavailable",
	}

	current := Snapshot{
		KernelRoute: &KernelRoute{
			Target:    "10.10.2.1",
			NextHop:   "172.20.31.1",
			Interface: "fm0",
		},
	}

	events := CompareSnapshots(&previous, current)

	if len(events) != 1 || events[0].Event != "KERNEL_ROUTE_CHANGE" {
		t.Fatalf("expected kernel route recovery event, got %+v", events)
	}
}
