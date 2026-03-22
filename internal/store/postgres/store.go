package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/aven/ngoogle/internal/store"
)

type pgStore struct {
	db       *sql.DB
	agents   *agentStore
	tasks    *taskStore
	metrics  *taskMetricsStore
	profiles *trafficProfileStore
	pools    *urlPoolStore
	groups   *taskGroupStore
	jobs     *provisionJobStore
	bw       *bandwidthStore
	creds    *credentialStore
}

// New opens a PostgreSQL database and runs migrations.
func New(dsn string) (store.Store, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("postgres open: %w", err)
	}
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := migrate(db); err != nil {
		return nil, fmt.Errorf("postgres migrate: %w", err)
	}
	s := &pgStore{
		db:       db,
		agents:   &agentStore{db},
		tasks:    &taskStore{db},
		metrics:  &taskMetricsStore{db},
		profiles: &trafficProfileStore{db},
		pools:    &urlPoolStore{db},
		groups:   &taskGroupStore{db},
		jobs:     &provisionJobStore{db},
		bw:       &bandwidthStore{db},
		creds:    &credentialStore{db},
	}
	return s, nil
}

func (s *pgStore) Agents() store.AgentStore                   { return s.agents }
func (s *pgStore) Tasks() store.TaskStore                     { return s.tasks }
func (s *pgStore) TaskMetrics() store.TaskMetricsStore        { return s.metrics }
func (s *pgStore) TrafficProfiles() store.TrafficProfileStore { return s.profiles }
func (s *pgStore) URLPools() store.URLPoolStore               { return s.pools }
func (s *pgStore) TaskGroups() store.TaskGroupStore           { return s.groups }
func (s *pgStore) ProvisionJobs() store.ProvisionJobStore     { return s.jobs }
func (s *pgStore) Bandwidth() store.BandwidthStore            { return s.bw }
func (s *pgStore) Credentials() store.CredentialStore         { return s.creds }
func (s *pgStore) Close() error                               { return s.db.Close() }

// ─── Migrations ───────────────────────────────────────────────────────────────

func migrate(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS agents (
			id TEXT PRIMARY KEY,
			hostname TEXT NOT NULL DEFAULT '',
			ip TEXT NOT NULL DEFAULT '',
			port INTEGER NOT NULL DEFAULT 0,
			token TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'offline',
			version TEXT NOT NULL DEFAULT '',
			current_rate_mbps DOUBLE PRECISION NOT NULL DEFAULT 0,
			last_heartbeat TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS tasks (
			id TEXT PRIMARY KEY,
			group_id TEXT NOT NULL DEFAULT '',
			name TEXT NOT NULL DEFAULT '',
			type TEXT NOT NULL DEFAULT 'static',
			url_pool_id TEXT NOT NULL DEFAULT '',
			target_url TEXT NOT NULL DEFAULT '',
			target_urls_json TEXT NOT NULL DEFAULT '[]',
			agent_id TEXT NOT NULL DEFAULT '',
			execution_scope TEXT NOT NULL DEFAULT 'single_agent',
			status TEXT NOT NULL DEFAULT 'pending',
			target_rate_mbps DOUBLE PRECISION NOT NULL DEFAULT 0,
			start_at TIMESTAMPTZ,
			end_at TIMESTAMPTZ,
			duration_sec INTEGER NOT NULL DEFAULT 0,
			total_bytes_target BIGINT NOT NULL DEFAULT 0,
			total_requests_target BIGINT NOT NULL DEFAULT 0,
			dispatch_rate_tpm INTEGER NOT NULL DEFAULT 0,
			dispatch_batch_size INTEGER NOT NULL DEFAULT 1,
			distribution TEXT NOT NULL DEFAULT 'flat',
			jitter_pct DOUBLE PRECISION NOT NULL DEFAULT 0,
			ramp_up_sec INTEGER NOT NULL DEFAULT 0,
			ramp_down_sec INTEGER NOT NULL DEFAULT 0,
			traffic_profile_id TEXT NOT NULL DEFAULT '',
			concurrent_fragments INTEGER NOT NULL DEFAULT 1,
			retries INTEGER NOT NULL DEFAULT 3,
			total_bytes_done BIGINT NOT NULL DEFAULT 0,
			error_message TEXT NOT NULL DEFAULT '',
			dispatched_at TIMESTAMPTZ,
			started_at TIMESTAMPTZ,
			finished_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS task_metrics (
			id BIGSERIAL PRIMARY KEY,
			task_id TEXT NOT NULL,
			agent_id TEXT NOT NULL,
			bytes_total BIGINT NOT NULL DEFAULT 0,
			bytes_delta BIGINT NOT NULL DEFAULT 0,
			rate_mbps_5s DOUBLE PRECISION NOT NULL DEFAULT 0,
			rate_mbps_30s DOUBLE PRECISION NOT NULL DEFAULT 0,
			request_count BIGINT NOT NULL DEFAULT 0,
			error_count BIGINT NOT NULL DEFAULT 0,
			recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_task_metrics_task_id ON task_metrics(task_id, recorded_at)`,
		`CREATE TABLE IF NOT EXISTS traffic_profiles (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL DEFAULT '',
			description TEXT NOT NULL DEFAULT '',
			distribution TEXT NOT NULL DEFAULT 'flat',
			points TEXT NOT NULL DEFAULT '[]',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS url_pools (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL DEFAULT '',
			type TEXT NOT NULL DEFAULT 'static',
			description TEXT NOT NULL DEFAULT '',
			urls_json TEXT NOT NULL DEFAULT '[]',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS task_groups (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL DEFAULT '',
			description TEXT NOT NULL DEFAULT '',
			pool_ids_json TEXT NOT NULL DEFAULT '[]',
			agent_id TEXT NOT NULL DEFAULT '',
			execution_scope TEXT NOT NULL DEFAULT 'single_agent',
			target_rate_mbps DOUBLE PRECISION NOT NULL DEFAULT 0,
			start_at TIMESTAMPTZ,
			end_at TIMESTAMPTZ,
			duration_sec INTEGER NOT NULL DEFAULT 0,
			total_bytes_target BIGINT NOT NULL DEFAULT 0,
			total_requests_target BIGINT NOT NULL DEFAULT 0,
			dispatch_rate_tpm INTEGER NOT NULL DEFAULT 0,
			dispatch_batch_size INTEGER NOT NULL DEFAULT 1,
			distribution TEXT NOT NULL DEFAULT 'flat',
			jitter_pct DOUBLE PRECISION NOT NULL DEFAULT 0,
			ramp_up_sec INTEGER NOT NULL DEFAULT 0,
			ramp_down_sec INTEGER NOT NULL DEFAULT 0,
			traffic_profile_id TEXT NOT NULL DEFAULT '',
			concurrent_fragments INTEGER NOT NULL DEFAULT 1,
			retries INTEGER NOT NULL DEFAULT 3,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
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
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS bandwidth_samples (
			id BIGSERIAL PRIMARY KEY,
			agent_id TEXT NOT NULL,
			rate_mbps DOUBLE PRECISION NOT NULL DEFAULT 0,
			recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			ts BIGINT NOT NULL DEFAULT 0
		)`,
		`CREATE INDEX IF NOT EXISTS idx_bandwidth_agent_time ON bandwidth_samples(agent_id, recorded_at)`,
		`CREATE INDEX IF NOT EXISTS idx_bandwidth_ts ON bandwidth_samples(ts)`,
		`CREATE TABLE IF NOT EXISTS bandwidth_agg (
			bucket BIGINT PRIMARY KEY,
			sum_mbps DOUBLE PRECISION NOT NULL DEFAULT 0,
			max_mbps DOUBLE PRECISION NOT NULL DEFAULT 0,
			cnt INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS credentials (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL DEFAULT '',
			type TEXT NOT NULL DEFAULT 'key',
			payload TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
	}
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			maxLen := len(stmt)
			if maxLen > 40 {
				maxLen = 40
			}
			return fmt.Errorf("exec %q: %w", stmt[:maxLen], err)
		}
	}
	// Ensure columns added in later migrations
	ensureColumn(db, "tasks", "target_urls_json", "TEXT NOT NULL DEFAULT '[]'")
	ensureColumn(db, "tasks", "group_id", "TEXT NOT NULL DEFAULT ''")
	ensureColumn(db, "tasks", "url_pool_id", "TEXT NOT NULL DEFAULT ''")
	ensureColumn(db, "tasks", "execution_scope", "TEXT NOT NULL DEFAULT 'single_agent'")
	ensureColumn(db, "bandwidth_samples", "ts", "BIGINT NOT NULL DEFAULT 0")

	// Backfill ts from recorded_at for existing rows
	if _, err := db.Exec(`UPDATE bandwidth_samples SET ts = EXTRACT(EPOCH FROM recorded_at)::BIGINT WHERE ts = 0`); err != nil {
		return fmt.Errorf("backfill bandwidth ts: %w", err)
	}
	// Backfill bandwidth_agg if empty
	var aggCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM bandwidth_agg`).Scan(&aggCount); err != nil {
		return fmt.Errorf("count bandwidth_agg: %w", err)
	}
	if aggCount == 0 {
		if _, err := db.Exec(`INSERT INTO bandwidth_agg(bucket, sum_mbps, max_mbps, cnt)
			SELECT (ts/60)*60, SUM(rate_mbps), MAX(rate_mbps), COUNT(*)
			FROM bandwidth_samples WHERE ts > 0 GROUP BY (ts/60)*60`); err != nil {
			return fmt.Errorf("backfill bandwidth_agg: %w", err)
		}
	}
	return nil
}

func ensureColumn(db *sql.DB, table, column, spec string) {
	var exists bool
	err := db.QueryRow(`SELECT EXISTS(
		SELECT 1 FROM information_schema.columns
		WHERE table_name=$1 AND column_name=$2
	)`, table, column).Scan(&exists)
	if err != nil || exists {
		return
	}
	db.Exec(fmt.Sprintf(`ALTER TABLE %s ADD COLUMN %s %s`, table, column, spec))
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

func checkCtx(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

type scanner interface {
	Scan(dest ...any) error
}
