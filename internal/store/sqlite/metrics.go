package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/aven/ngoogle/internal/model"
	"github.com/aven/ngoogle/internal/store"
)

type taskMetricsStore struct{ db, ro *sql.DB }

func (s *taskMetricsStore) Insert(ctx context.Context, m *model.TaskMetrics) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO task_metrics (task_id,agent_id,bytes_total,bytes_delta,rate_mbps_5s,rate_mbps_30s,request_count,error_count,recorded_at)
		VALUES (?,?,?,?,?,?,?,?,?)`,
		m.TaskID, m.AgentID, m.BytesTotal, m.BytesDelta,
		m.RateMbps5s, m.RateMbps30s, m.RequestCount, m.ErrorCount, m.RecordedAt.UTC().Format("2006-01-02 15:04:05"),
	)
	return err
}

func (s *taskMetricsStore) ListByTask(ctx context.Context, taskID string, from, to time.Time) ([]*model.TaskMetrics, error) {
	rows, err := s.ro.QueryContext(ctx, `
		SELECT id,task_id,agent_id,bytes_total,bytes_delta,rate_mbps_5s,rate_mbps_30s,request_count,error_count,recorded_at
		FROM task_metrics WHERE task_id=? AND recorded_at BETWEEN ? AND ? ORDER BY recorded_at ASC`,
		taskID, from.UTC().Format("2006-01-02 15:04:05"), to.UTC().Format("2006-01-02 15:04:05"))
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
	row := s.ro.QueryRowContext(ctx, `
		SELECT id,task_id,agent_id,bytes_total,bytes_delta,rate_mbps_5s,rate_mbps_30s,request_count,error_count,recorded_at
		FROM task_metrics WHERE task_id=? ORDER BY recorded_at DESC LIMIT 1`, taskID)
	m := &model.TaskMetrics{}
	err := row.Scan(&m.ID, &m.TaskID, &m.AgentID, &m.BytesTotal, &m.BytesDelta,
		&m.RateMbps5s, &m.RateMbps30s, &m.RequestCount, &m.ErrorCount, &m.RecordedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return m, err
}

func (s *taskMetricsStore) LatestByTaskAgents(ctx context.Context, taskID string) ([]*model.TaskMetrics, error) {
	rows, err := s.ro.QueryContext(ctx, `
		SELECT tm.id,tm.task_id,tm.agent_id,tm.bytes_total,tm.bytes_delta,tm.rate_mbps_5s,tm.rate_mbps_30s,tm.request_count,tm.error_count,tm.recorded_at
		FROM task_metrics tm
		INNER JOIN (
			SELECT agent_id, MAX(recorded_at) AS max_recorded_at
			FROM task_metrics
			WHERE task_id=?
			GROUP BY agent_id
		) latest
			ON latest.agent_id = tm.agent_id
			AND latest.max_recorded_at = tm.recorded_at
		WHERE tm.task_id=?
		ORDER BY tm.agent_id ASC`, taskID, taskID)
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

type bandwidthStore struct{ db, ro *sql.DB }

func (s *bandwidthStore) Insert(ctx context.Context, bs *model.BandwidthSample) error {
	unix := bs.RecordedAt.Unix()
	_, err := s.db.ExecContext(ctx, `INSERT INTO bandwidth_samples(agent_id,rate_mbps,recorded_at,ts) VALUES(?,?,?,?)`,
		bs.AgentID, bs.RateMbps, bs.RecordedAt.UTC().Format("2006-01-02 15:04:05"), unix)
	if err != nil {
		return err
	}
	// Update pre-aggregated 1-minute bucket
	bucket := (unix / 60) * 60
	_, err = s.db.ExecContext(ctx, `INSERT INTO bandwidth_agg(bucket,sum_mbps,max_mbps,cnt) VALUES(?,?,?,1)
		ON CONFLICT(bucket) DO UPDATE SET sum_mbps=sum_mbps+excluded.sum_mbps, max_mbps=MAX(max_mbps,excluded.max_mbps), cnt=cnt+1`,
		bucket, bs.RateMbps, bs.RateMbps)
	return err
}

func (s *bandwidthStore) History(ctx context.Context, agentID string, from, to time.Time) ([]*model.BandwidthSample, error) {
	rows, err := s.ro.QueryContext(ctx, `
		SELECT id,agent_id,rate_mbps,recorded_at FROM bandwidth_samples
		WHERE agent_id=? AND recorded_at BETWEEN ? AND ? ORDER BY recorded_at ASC`,
		agentID, from.UTC().Format("2006-01-02 15:04:05"), to.UTC().Format("2006-01-02 15:04:05"))
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
	// Read from pre-aggregated 1-minute table, re-bucket if stepSec > 60
	rows, err := s.ro.QueryContext(ctx, fmt.Sprintf(`
		SELECT
			(bucket / %d) * %d as b,
			SUM(sum_mbps),
			MAX(max_mbps)
		FROM bandwidth_agg
		WHERE bucket BETWEEN ? AND ?
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
	if _, err := s.db.ExecContext(ctx, `DELETE FROM bandwidth_samples WHERE ts < ?`, unix); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, `DELETE FROM bandwidth_agg WHERE bucket < ?`, unix)
	return err
}

func (s *bandwidthStore) TotalCurrent(ctx context.Context, since time.Time) (float64, error) {
	// Sum of the latest rate_mbps per agent
	row := s.ro.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(rate_mbps),0) FROM (
			SELECT agent_id, rate_mbps FROM bandwidth_samples
			WHERE ts >= ?
			GROUP BY agent_id
			HAVING ts=MAX(ts)
		)`, since.Unix())
	var total float64
	return total, row.Scan(&total)
}
