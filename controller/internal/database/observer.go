package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type ReplicationPeer struct {
	ApplicationName string `json:"application_name"`
	ClientAddress   string `json:"client_address"`
	State           string `json:"state"`
	SyncState       string `json:"sync_state"`
	SentLSN         string `json:"sent_lsn"`
	ReplayLSN       string `json:"replay_lsn"`
	LagBytes        int64  `json:"lag_bytes"`
}

type WALReceiver struct {
	Status     string `json:"status"`
	SenderHost string `json:"sender_host"`
	SlotName   string `json:"slot_name"`
}

type Snapshot struct {
	Region          string            `json:"region"`
	ObservedAt      time.Time         `json:"observed_at"`
	Role            string            `json:"role"`
	InRecovery      bool              `json:"in_recovery"`
	CurrentFlushLSN string            `json:"current_flush_lsn,omitempty"`
	ReplayLSN       string            `json:"replay_lsn,omitempty"`
	Receiver        *WALReceiver      `json:"wal_receiver,omitempty"`
	Peers           []ReplicationPeer `json:"replication_peers"`
}

type Observer struct {
	db *sql.DB
}

func Open(ctx context.Context, dsn string) (*Observer, error) {
	if dsn == "" {
		return nil, errors.New("PostgreSQL DSN is required")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open PostgreSQL connection: %w", err)
	}

	db.SetMaxOpenConns(3)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("PostgreSQL ping failed: %w", err)
	}

	return &Observer{db: db}, nil
}

func (o *Observer) Close() error {
	return o.db.Close()
}

func (o *Observer) Snapshot(ctx context.Context, region string) (Snapshot, error) {
	result := Snapshot{
		Region:     region,
		ObservedAt: time.Now().UTC(),
		Peers:      []ReplicationPeer{},
	}

	var flushLSN, replayLSN sql.NullString

	err := o.db.QueryRowContext(ctx, `
		SELECT
			pg_is_in_recovery(),
			CASE WHEN pg_is_in_recovery()
				THEN NULL
				ELSE pg_current_wal_flush_lsn()::text
			END,
			CASE WHEN pg_is_in_recovery()
				THEN pg_last_wal_replay_lsn()::text
				ELSE NULL
			END
	`).Scan(
		&result.InRecovery,
		&flushLSN,
		&replayLSN,
	)

	if err != nil {
		return result, fmt.Errorf("read PostgreSQL role and WAL state: %w", err)
	}
	if flushLSN.Valid {
		result.CurrentFlushLSN = flushLSN.String
	}
	if replayLSN.Valid {
		result.ReplayLSN = replayLSN.String
	}

	if result.InRecovery {
		result.Role = "STANDBY"

		var receiver WALReceiver
		err := o.db.QueryRowContext(ctx, `
			SELECT status, sender_host::text, COALESCE(slot_name, '')
			FROM pg_stat_wal_receiver
		`).Scan(
			&receiver.Status,
			&receiver.SenderHost,
			&receiver.SlotName,
		)

		if err == nil {
			result.Receiver = &receiver
		} else if !errors.Is(err, sql.ErrNoRows) {
			return result, fmt.Errorf("read WAL receiver: %w", err)
		}
	} else {
		result.Role = "PRIMARY"

		rows, err := o.db.QueryContext(ctx, `
			SELECT
				application_name,
				COALESCE(client_addr::text, ''),
				state,
				sync_state,
				COALESCE(sent_lsn::text, ''),
				COALESCE(replay_lsn::text, ''),
				COALESCE(pg_wal_lsn_diff(sent_lsn, replay_lsn), 0)::bigint
			FROM pg_stat_replication
		`)
		if err != nil {
			return result, fmt.Errorf("read replication peers: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var peer ReplicationPeer

			if err := rows.Scan(
				&peer.ApplicationName,
				&peer.ClientAddress,
				&peer.State,
				&peer.SyncState,
				&peer.SentLSN,
				&peer.ReplayLSN,
				&peer.LagBytes,
			); err != nil {
				return result, fmt.Errorf("scan replication peer: %w", err)
			}

			result.Peers = append(result.Peers, peer)
		}

		if err := rows.Err(); err != nil {
			return result, fmt.Errorf("iterate replication peers: %w", err)
		}
	}

	return result, nil
}
