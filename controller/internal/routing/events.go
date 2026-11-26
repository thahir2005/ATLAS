package routing

import (
	"reflect"
	"sort"
	"time"
)

type Event struct {
	Timestamp string `json:"timestamp"`
	Event     string `json:"event"`
	Details   any    `json:"details"`
}

func CompareSnapshots(previous *Snapshot, current Snapshot) []Event {
	var events []Event

	emit := func(name string, details any) {
		events = append(events, Event{
			Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
			Event:     name,
			Details:   details,
		})
	}

	if previous == nil {
		emit("INITIAL_STATE", current)
		return events
	}

	addresses := make(map[string]struct{})

	for address := range previous.Peers {
		addresses[address] = struct{}{}
	}
	for address := range current.Peers {
		addresses[address] = struct{}{}
	}

	sortedAddresses := make([]string, 0, len(addresses))
	for address := range addresses {
		sortedAddresses = append(sortedAddresses, address)
	}
	sort.Strings(sortedAddresses)

	for _, address := range sortedAddresses {
		oldPeer, oldExists := previous.Peers[address]
		newPeer, newExists := current.Peers[address]

		oldState := "Missing"
		newState := "Missing"

		if oldExists {
			oldState = oldPeer.State
		}
		if newExists {
			newState = newPeer.State
		}

		if oldState != newState {
			emit("PEER_STATE_CHANGE", map[string]any{
				"peer":           address,
				"previous_state": oldState,
				"current_state":  newState,
			})
		}
	}

	if !reflect.DeepEqual(previous.BestPath, current.BestPath) {
		emit("BGP_BESTPATH_CHANGE", map[string]any{
			"previous": previous.BestPath,
			"current":  current.BestPath,
		})
	}

	if !reflect.DeepEqual(previous.KernelRoute, current.KernelRoute) ||
		previous.KernelRouteError != current.KernelRouteError {
		emit("KERNEL_ROUTE_CHANGE", map[string]any{
			"previous":      previous.KernelRoute,
			"current":       current.KernelRoute,
			"current_error": current.KernelRouteError,
		})
	}

	return events
}
