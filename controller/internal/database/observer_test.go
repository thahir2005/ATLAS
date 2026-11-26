package database

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestSnapshotPrimaryWithTwoReplicas(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mock.ExpectQuery("SELECT\\s+pg_is_in_recovery").
		WillReturnRows(sqlmock.NewRows([]string{
			"in_recovery", "flush_lsn", "replay_lsn",
		}).AddRow(false, "0/70015C8", nil))

	mock.ExpectQuery("SELECT\\s+application_name").
		WillReturnRows(sqlmock.NewRows([]string{
			"application_name", "client_addr", "state",
			"sync_state", "sent_lsn", "replay_lsn", "lag_bytes",
		}).
			AddRow("atlas-mumbai", "172.20.12.1/32", "streaming",
				"async", "0/70015C8", "0/70015C8", int64(0)).
			AddRow("atlas-frankfurt", "172.20.23.2/32", "streaming",
				"async", "0/70015C8", "0/70015C8", int64(0)))

	observer := &Observer{db: db}
	snapshot, err := observer.Snapshot(context.Background(), "singapore")
	if err != nil {
		t.Fatal(err)
	}

	if snapshot.Role != "PRIMARY" || snapshot.InRecovery {
		t.Fatalf("unexpected primary state: %+v", snapshot)
	}
	if snapshot.CurrentFlushLSN != "0/70015C8" {
		t.Fatalf("unexpected flush LSN: %s", snapshot.CurrentFlushLSN)
	}
	if len(snapshot.Peers) != 2 {
		t.Fatalf("expected 2 replicas, got %d", len(snapshot.Peers))
	}
	for _, peer := range snapshot.Peers {
		if peer.State != "streaming" || peer.LagBytes != 0 {
			t.Errorf("unexpected replica state: %+v", peer)
		}
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSnapshotStandbyWithWALReceiver(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mock.ExpectQuery("SELECT\\s+pg_is_in_recovery").
		WillReturnRows(sqlmock.NewRows([]string{
			"in_recovery", "flush_lsn", "replay_lsn",
		}).AddRow(true, nil, "0/70015C8"))

	mock.ExpectQuery(regexp.QuoteMeta(
		"SELECT status, sender_host::text, COALESCE(slot_name, '')",
	)).WillReturnRows(sqlmock.NewRows([]string{
		"status", "sender_host", "slot_name",
	}).AddRow("streaming", "10.10.2.1", "atlas_mumbai_slot_v2"))

	observer := &Observer{db: db}
	snapshot, err := observer.Snapshot(context.Background(), "mumbai")
	if err != nil {
		t.Fatal(err)
	}

	if snapshot.Role != "STANDBY" || !snapshot.InRecovery {
		t.Fatalf("unexpected standby state: %+v", snapshot)
	}
	if snapshot.Receiver == nil {
		t.Fatal("expected WAL receiver")
	}
	if snapshot.Receiver.SenderHost != "10.10.2.1" {
		t.Fatalf("unexpected sender: %s", snapshot.Receiver.SenderHost)
	}
	if snapshot.Receiver.Status != "streaming" {
		t.Fatalf("unexpected receiver status: %s", snapshot.Receiver.Status)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSnapshotStandbyWithoutWALReceiver(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mock.ExpectQuery("SELECT\\s+pg_is_in_recovery").
		WillReturnRows(sqlmock.NewRows([]string{
			"in_recovery", "flush_lsn", "replay_lsn",
		}).AddRow(true, nil, "0/70015C8"))

	mock.ExpectQuery(regexp.QuoteMeta(
		"SELECT status, sender_host::text, COALESCE(slot_name, '')",
	)).WillReturnRows(sqlmock.NewRows([]string{
		"status", "sender_host", "slot_name",
	}))

	observer := &Observer{db: db}
	snapshot, err := observer.Snapshot(context.Background(), "mumbai")
	if err != nil {
		t.Fatal(err)
	}

	if snapshot.Role != "STANDBY" {
		t.Fatalf("expected STANDBY, got %s", snapshot.Role)
	}
	if snapshot.Receiver != nil {
		t.Fatalf("expected no receiver, got %+v", snapshot.Receiver)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
