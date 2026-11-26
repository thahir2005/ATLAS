package intelligence

import (
	"fmt"

	dbobserver "atlas.local/control-plane/controller/internal/database"
)

type DatabaseStatus string

const (
	DatabaseHealthy             DatabaseStatus = "HEALTHY"
	DatabaseReplicationDegraded DatabaseStatus = "REPLICATION_DEGRADED"
	DatabaseUnavailable         DatabaseStatus = "DATABASE_UNAVAILABLE"
	DatabaseUnknown             DatabaseStatus = "UNKNOWN"
)

type DatabaseAssessment struct {
	Region           string                       `json:"region"`
	Role             string                       `json:"role"`
	Status           DatabaseStatus               `json:"status"`
	InRecovery       bool                         `json:"in_recovery"`
	Receiver         *dbobserver.WALReceiver      `json:"wal_receiver,omitempty"`
	ReplicationPeers []dbobserver.ReplicationPeer `json:"replication_peers"`
	MaxLagBytes      int64                        `json:"max_lag_bytes"`
	Reason           string                       `json:"reason"`
}

func AssessDatabase(
	region string,
	snapshot dbobserver.Snapshot,
	observationErr error,
	expectedReplicas int,
) DatabaseAssessment {
	result := DatabaseAssessment{
		Region:           region,
		Role:             snapshot.Role,
		InRecovery:       snapshot.InRecovery,
		Receiver:         snapshot.Receiver,
		ReplicationPeers: snapshot.Peers,
		Status:           DatabaseUnknown,
	}

	if observationErr != nil {
		result.Status = DatabaseUnavailable
		result.Reason = fmt.Sprintf("Database observation failed: %v", observationErr)
		return result
	}

	switch snapshot.Role {
	case "PRIMARY":
		if snapshot.InRecovery {
			result.Status = DatabaseUnknown
			result.Reason = "Primary role conflicts with recovery state"
			return result
		}

		if len(snapshot.Peers) != expectedReplicas {
			result.Status = DatabaseReplicationDegraded
			result.Reason = fmt.Sprintf(
				"Expected %d replication peers, observed %d",
				expectedReplicas,
				len(snapshot.Peers),
			)
			return result
		}

		for _, peer := range snapshot.Peers {
			if peer.LagBytes > result.MaxLagBytes {
				result.MaxLagBytes = peer.LagBytes
			}

			if peer.State != "streaming" {
				result.Status = DatabaseReplicationDegraded
				result.Reason = fmt.Sprintf(
					"Replication peer %s is %s",
					peer.ApplicationName,
					peer.State,
				)
				return result
			}
		}

		result.Status = DatabaseHealthy
		result.Reason = "Primary is writable and all expected replicas are streaming"

	case "STANDBY":
		if !snapshot.InRecovery {
			result.Status = DatabaseUnknown
			result.Reason = "Standby role conflicts with recovery state"
			return result
		}

		if snapshot.Receiver == nil {
			result.Status = DatabaseReplicationDegraded
			result.Reason = "Standby has no active WAL receiver"
			return result
		}

		if snapshot.Receiver.Status != "streaming" {
			result.Status = DatabaseReplicationDegraded
			result.Reason = fmt.Sprintf(
				"WAL receiver status is %s",
				snapshot.Receiver.Status,
			)
			return result
		}

		result.Status = DatabaseHealthy
		result.Reason = "Standby is receiving WAL from its upstream primary"

	default:
		result.Status = DatabaseUnknown
		result.Reason = "Database role is unknown"
	}

	return result
}
