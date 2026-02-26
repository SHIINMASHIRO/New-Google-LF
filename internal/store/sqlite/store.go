package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"

	"github.com/aven/ngoogle/internal/store"
)

// sqliteStore implements store.Store.
type sqliteStore struct {
	db      *sql.DB
	agents  *agentStore
	tasks   *taskStore
	metrics *taskMetricsStore
	profiles *trafficProfileStore
	jobs    *provisionJobStore
	bw      *bandwidthStore
	creds   *credentialStore
}

// New opens (or creates) a SQLite database and runs migrations.
func New(dsn string) (store.Store, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("sqlite open: %w", err)
	}
	db.SetMaxOpenConns(1) // SQLite single-writer
	if err := migrate(db); err != nil {
		return nil, fmt.Errorf("sqlite migrate: %w", err)
	}
	s := &sqliteStore{
		db:       db,
		agents:   &agentStore{db},
		tasks:    &taskStore{db},
		metrics:  &taskMetricsStore{db},
		profiles: &trafficProfileStore{db},
		jobs:     &provisionJobStore{db},
		bw:       &bandwidthStore{db},
		creds:    &credentialStore{db},
	}
	return s, nil
}

func (s *sqliteStore) Agents() store.AgentStore            { return s.agents }
func (s *sqliteStore) Tasks() store.TaskStore              { return s.tasks }
func (s *sqliteStore) TaskMetrics() store.TaskMetricsStore { return s.metrics }
func (s *sqliteStore) TrafficProfiles() store.TrafficProfileStore { return s.profiles }
func (s *sqliteStore) ProvisionJobs() store.ProvisionJobStore     { return s.jobs }
func (s *sqliteStore) Bandwidth() store.BandwidthStore            { return s.bw }
func (s *sqliteStore) Credentials() store.CredentialStore         { return s.creds }
func (s *sqliteStore) Close() error                               { return s.db.Close() }

// ─── Migrations ───────────────────────────────────────────────────────────────

func migrate(db *sql.DB) error {
	stmts := []string{
		`PRAGMA journal_mode=WAL;`,
		`PRAGMA foreign_keys=ON;`,
		`CREATE TABLE IF NOT EXISTS agents (
			id TEXT PRIMARY KEY,
			hostname TEXT NOT NULL DEFAULT '',
			ip TEXT NOT NULL DEFAULT '',
			port INTEGER NOT NULL DEFAULT 0,
			token TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'offline',
			version TEXT NOT NULL DEFAULT '',
			current_rate_mbps REAL NOT NULL DEFAULT 0,
			last_heartbeat DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS tasks (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL DEFAULT '',
			type TEXT NOT NULL DEFAULT 'static',
			target_url TEXT NOT NULL DEFAULT '',
			agent_id TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'pending',
			target_rate_mbps REAL NOT NULL DEFAULT 0,
			start_at DATETIME,
			end_at DATETIME,
			duration_sec INTEGER NOT NULL DEFAULT 0,
			total_bytes_target INTEGER NOT NULL DEFAULT 0,
			total_requests_target INTEGER NOT NULL DEFAULT 0,
			dispatch_rate_tpm INTEGER NOT NULL DEFAULT 0,
			dispatch_batch_size INTEGER NOT NULL DEFAULT 1,
			distribution TEXT NOT NULL DEFAULT 'flat',
			jitter_pct REAL NOT NULL DEFAULT 0,
			ramp_up_sec INTEGER NOT NULL DEFAULT 0,
			ramp_down_sec INTEGER NOT NULL DEFAULT 0,
			traffic_profile_id TEXT NOT NULL DEFAULT '',
			concurrent_fragments INTEGER NOT NULL DEFAULT 1,
			retries INTEGER NOT NULL DEFAULT 3,
			total_bytes_done INTEGER NOT NULL DEFAULT 0,
			error_message TEXT NOT NULL DEFAULT '',
			dispatched_at DATETIME,
			started_at DATETIME,
			finished_at DATETIME,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS task_metrics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			task_id TEXT NOT NULL,
			agent_id TEXT NOT NULL,
			bytes_total INTEGER NOT NULL DEFAULT 0,
			bytes_delta INTEGER NOT NULL DEFAULT 0,
			rate_mbps_5s REAL NOT NULL DEFAULT 0,
			rate_mbps_30s REAL NOT NULL DEFAULT 0,
			request_count INTEGER NOT NULL DEFAULT 0,
			error_count INTEGER NOT NULL DEFAULT 0,
			recorded_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE INDEX IF NOT EXISTS idx_task_metrics_task_id ON task_metrics(task_id, recorded_at);`,
		`CREATE TABLE IF NOT EXISTS traffic_profiles (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL DEFAULT '',
			description TEXT NOT NULL DEFAULT '',
			distribution TEXT NOT NULL DEFAULT 'flat',
			points TEXT NOT NULL DEFAULT '[]',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS provision_jobs (
			id TEXT PRIMARY KEY,
			host_ip TEXT NOT NULL DEFAULT '',
			ssh_port INTEGER NOT NULL DEFAULT 22,
			ssh_user TEXT NOT NULL DEFAULT '',
			auth_type TEXT NOT NULL DEFAULT 'key',
			credential_ref TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'pending',
			current_step TEXT NOT NULL DEFAULT '',
			log TEXT NOT NULL DEFAULT '',
			agent_id TEXT NOT NULL DEFAULT '',
			failed_step TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS bandwidth_samples (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			agent_id TEXT NOT NULL,
			rate_mbps REAL NOT NULL DEFAULT 0,
			recorded_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE INDEX IF NOT EXISTS idx_bandwidth_agent_time ON bandwidth_samples(agent_id, recorded_at);`,
		`CREATE TABLE IF NOT EXISTS credentials (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL DEFAULT '',
			type TEXT NOT NULL DEFAULT 'key',
			payload TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
	}
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("exec %q: %w", stmt[:min(40, len(stmt))], err)
		}
	}
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func nullTime(t *time.Time) interface{} {
	if t == nil {
		return nil
	}
	return t.UTC()
}

func scanNullTime(ns sql.NullTime) *time.Time {
	if !ns.Valid {
		return nil
	}
	t := ns.Time
	return &t
}

// Ensure context is not done before executing.
func checkCtx(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}
