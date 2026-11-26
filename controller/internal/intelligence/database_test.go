package intelligence

import (
	"errors"
	"testing"

	dbobserver "atlas.local/control-plane/controller/internal/database"
)

func TestAssessHealthyPrimary(t *testing.T) {
	snapshot := dbobserver.Snapshot{
		Region: "singapore",
		Role:   "PRIMARY",
		Peers: []dbobserver.ReplicationPeer{
			{ApplicationName: "atlas-mumbai", State: "streaming", LagBytes: 0},
			{ApplicationName: "atlas-frankfurt", State: "streaming", LagBytes: 128},
		},
	}

	result := AssessDatabase("singapore", snapshot, nil, 2)

	if result.Status != DatabaseHealthy {
		t.Fatalf("expected HEALTHY, got %s: %s", result.Status, result.Reason)
	}
	if result.MaxLagBytes != 128 {
		t.Fatalf("expected max lag 128, got %d", result.MaxLagBytes)
	}
}

func TestAssessPrimaryMissingReplica(t *testing.T) {
	snapshot := dbobserver.Snapshot{
		Role: "PRIMARY",
		Peers: []dbobserver.ReplicationPeer{
			{ApplicationName: "atlas-mumbai", State: "streaming"},
		},
	}

	result := AssessDatabase("singapore", snapshot, nil, 2)

	if result.Status != DatabaseReplicationDegraded {
		t.Fatalf("expected replication degraded, got %s", result.Status)
	}
}

func TestAssessHealthyStandby(t *testing.T) {
	snapshot := dbobserver.Snapshot{
		Region:     "mumbai",
		Role:       "STANDBY",
		InRecovery: true,
		Receiver: &dbobserver.WALReceiver{
			Status:     "streaming",
			SenderHost: "10.10.2.1",
			SlotName:   "atlas_mumbai_slot_v2",
		},
	}

	result := AssessDatabase("mumbai", snapshot, nil, 0)

	if result.Status != DatabaseHealthy {
		t.Fatalf("expected HEALTHY, got %s", result.Status)
	}
}

func TestAssessStandbyWithoutReceiver(t *testing.T) {
	snapshot := dbobserver.Snapshot{
		Role:       "STANDBY",
		InRecovery: true,
	}

	result := AssessDatabase("mumbai", snapshot, nil, 0)

	if result.Status != DatabaseReplicationDegraded {
		t.Fatalf("expected replication degraded, got %s", result.Status)
	}
}

func TestAssessDatabaseUnavailable(t *testing.T) {
	result := AssessDatabase(
		"frankfurt",
		dbobserver.Snapshot{},
		errors.New("connection refused"),
		0,
	)

	if result.Status != DatabaseUnavailable {
		t.Fatalf("expected DATABASE_UNAVAILABLE, got %s", result.Status)
	}
}

func TestAssessUnknownDatabaseRole(t *testing.T) {
	result := AssessDatabase(
		"mumbai",
		dbobserver.Snapshot{Role: "UNKNOWN"},
		nil,
		0,
	)

	if result.Status != DatabaseUnknown {
		t.Fatalf("expected UNKNOWN, got %s", result.Status)
	}
}
