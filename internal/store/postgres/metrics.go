package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/aven/ngoogle/internal/model"
	"github.com/aven/ngoogle/internal/store"
)

type taskMetricsStore struct{ db *sql.DB }

func (s *taskMetricsStore) Insert(ctx context.Context, m *model.TaskMetrics) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO task_metrics (task_id,agent_id,bytes_total,bytes_delta,rate_mbps_5s,rate_mbps_30s,request_count,error_count,recorded_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		m.TaskID, m.AgentID, m.BytesTotal, m.BytesDelta,
		m.RateMbps5s, m.RateMbps30s, m.RequestCount, m.ErrorCount, m.RecordedAt.UTC(),
	)
	return err
}

func (s *taskMetricsStore) ListByTask(ctx context.Context, taskID string, from, to time.Time) ([]*model.TaskMetrics, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id,task_id,agent_id,bytes_total,bytes_delta,rate_mbps_5s,rate_mbps_30s,request_count,error_count,recorded_at
		FROM task_metrics WHERE task_id=$1 AND recorded_at BETWEEN $2 AND $3 ORDER BY recorded_at ASC`,
		taskID, from.UTC(), to.UTC())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*model.TaskMetrics
	for rows.Next() {
		m := &model.TaskMetrics{}
		if err := rows.Scan(&m.ID, &m.TaskID, &m.AgentID, &m.BytesTotal, &m.BytesDelta,
			&m.RateMbps5s, &m.RateMbps30s, &m.RequestCount, &m.ErrorCount, &m.RecordedAt); err != nil {
			return nil, err
		}
		list = append(list, m)
	}
	return list, rows.Err()
}

func (s *taskMetricsStore) LatestByTask(ctx context.Context, taskID string) (*model.TaskMetrics, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id,task_id,agent_id,bytes_total,bytes_delta,rate_mbps_5s,rate_mbps_30s,request_count,error_count,recorded_at
		FROM task_metrics WHERE task_id=$1 ORDER BY recorded_at DESC LIMIT 1`, taskID)
	m := &model.TaskMetrics{}
	err := row.Scan(&m.ID, &m.TaskID, &m.AgentID, &m.BytesTotal, &m.BytesDelta,
		&m.RateMbps5s, &m.RateMbps30s, &m.RequestCount, &m.ErrorCount, &m.RecordedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return m, err
}

func (s *taskMetricsStore) LatestByTaskAgents(ctx context.Context, taskID string) ([]*model.TaskMetrics, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT DISTINCT ON (tm.agent_id)
			tm.id,tm.task_id,tm.agent_id,tm.bytes_total,tm.bytes_delta,tm.rate_mbps_5s,tm.rate_mbps_30s,tm.request_count,tm.error_count,tm.recorded_at
		FROM task_metrics tm
		WHERE tm.task_id=$1
		ORDER BY tm.agent_id, tm.recorded_at DESC`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*model.TaskMetrics
	for rows.Next() {
		m := &model.TaskMetrics{}
		if err := rows.Scan(&m.ID, &m.TaskID, &m.AgentID, &m.BytesTotal, &m.BytesDelta,
			&m.RateMbps5s, &m.RateMbps30s, &m.RequestCount, &m.ErrorCount, &m.RecordedAt); err != nil {
			return nil, err
		}
		list = append(list, m)
	}
	return list, rows.Err()
}

// ─── Bandwidth ────────────────────────────────────────────────────────────────

type bandwidthStore struct{ db *sql.DB }

func (s *bandwidthStore) Insert(ctx context.Context, bs *model.BandwidthSample) error {
	unix := bs.RecordedAt.Unix()
	_, err := s.db.ExecContext(ctx, `INSERT INTO bandwidth_samples(agent_id,rate_mbps,recorded_at,ts) VALUES($1,$2,$3,$4)`,
		bs.AgentID, bs.RateMbps, bs.RecordedAt.UTC(), unix)
	if err != nil {
		return err
	}
	// Update pre-aggregated 1-minute bucket
	bucket := (unix / 60) * 60
	_, err = s.db.ExecContext(ctx, `INSERT INTO bandwidth_agg(bucket,sum_mbps,max_mbps,cnt) VALUES($1,$2,$3,1)
		ON CONFLICT(bucket) DO UPDATE SET sum_mbps=bandwidth_agg.sum_mbps+excluded.sum_mbps, max_mbps=GREATEST(bandwidth_agg.max_mbps,excluded.max_mbps), cnt=bandwidth_agg.cnt+1`,
		bucket, bs.RateMbps, bs.RateMbps)
	return err
}

func (s *bandwidthStore) History(ctx context.Context, agentID string, from, to time.Time) ([]*model.BandwidthSample, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id,agent_id,rate_mbps,recorded_at FROM bandwidth_samples
		WHERE agent_id=$1 AND recorded_at BETWEEN $2 AND $3 ORDER BY recorded_at ASC`,
		agentID, from.UTC(), to.UTC())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*model.BandwidthSample
	for rows.Next() {
		b := &model.BandwidthSample{}
		if err := rows.Scan(&b.ID, &b.AgentID, &b.RateMbps, &b.RecordedAt); err != nil {
			return nil, err
		}
		list = append(list, b)
	}
	return list, rows.Err()
}

func (s *bandwidthStore) AggregateHistory(ctx context.Context, from, to time.Time, stepSec int) ([]store.BandwidthPoint, error) {
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT
			(bucket / %d) * %d as b,
			SUM(sum_mbps),
			MAX(max_mbps)
		FROM bandwidth_agg
		WHERE bucket BETWEEN $1 AND $2
		GROUP BY b ORDER BY b ASC`, stepSec, stepSec),
		from.Unix(), to.Unix())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []store.BandwidthPoint
	for rows.Next() {
		var p store.BandwidthPoint
		var bucketUnix int64
		if err := rows.Scan(&bucketUnix, &p.AvgMbps, &p.MaxMbps); err != nil {
			return nil, err
		}
		p.Ts = time.Unix(bucketUnix, 0).UTC()
		result = append(result, p)
	}
	return result, rows.Err()
}

func (s *bandwidthStore) PurgeOlderThan(ctx context.Context, before time.Time) error {
	unix := before.Unix()
	if _, err := s.db.ExecContext(ctx, `DELETE FROM bandwidth_samples WHERE ts < $1`, unix); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, `DELETE FROM bandwidth_agg WHERE bucket < $1`, unix)
	return err
}

func (s *bandwidthStore) TotalCurrent(ctx context.Context, since time.Time) (float64, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(rate_mbps),0) FROM (
			SELECT DISTINCT ON (agent_id) rate_mbps
			FROM bandwidth_samples
			WHERE ts >= $1
			ORDER BY agent_id, ts DESC
		) sub`, since.Unix())
	var total float64
	return total, row.Scan(&total)
}
