package sqlite

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
		VALUES (?,?,?,?,?,?,?,?,?)`,
		m.TaskID, m.AgentID, m.BytesTotal, m.BytesDelta,
		m.RateMbps5s, m.RateMbps30s, m.RequestCount, m.ErrorCount, m.RecordedAt.UTC(),
	)
	return err
}

func (s *taskMetricsStore) ListByTask(ctx context.Context, taskID string, from, to time.Time) ([]*model.TaskMetrics, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id,task_id,agent_id,bytes_total,bytes_delta,rate_mbps_5s,rate_mbps_30s,request_count,error_count,recorded_at
		FROM task_metrics WHERE task_id=? AND recorded_at BETWEEN ? AND ? ORDER BY recorded_at ASC`,
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
		FROM task_metrics WHERE task_id=? ORDER BY recorded_at DESC LIMIT 1`, taskID)
	m := &model.TaskMetrics{}
	err := row.Scan(&m.ID, &m.TaskID, &m.AgentID, &m.BytesTotal, &m.BytesDelta,
		&m.RateMbps5s, &m.RateMbps30s, &m.RequestCount, &m.ErrorCount, &m.RecordedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return m, err
}

// ─── Bandwidth ────────────────────────────────────────────────────────────────

type bandwidthStore struct{ db *sql.DB }

func (s *bandwidthStore) Insert(ctx context.Context, bs *model.BandwidthSample) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO bandwidth_samples(agent_id,rate_mbps,recorded_at) VALUES(?,?,?)`,
		bs.AgentID, bs.RateMbps, bs.RecordedAt.UTC())
	return err
}

func (s *bandwidthStore) History(ctx context.Context, agentID string, from, to time.Time) ([]*model.BandwidthSample, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id,agent_id,rate_mbps,recorded_at FROM bandwidth_samples
		WHERE agent_id=? AND recorded_at BETWEEN ? AND ? ORDER BY recorded_at ASC`,
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
	// SQLite: bucket by stepSec using integer division of unix timestamp
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT
			datetime((strftime('%%s', recorded_at) / %d) * %d, 'unixepoch') as bucket,
			AVG(rate_mbps),
			MAX(rate_mbps)
		FROM bandwidth_samples
		WHERE recorded_at BETWEEN ? AND ?
		GROUP BY bucket ORDER BY bucket ASC`, stepSec, stepSec),
		from.UTC(), to.UTC())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []store.BandwidthPoint
	for rows.Next() {
		var p store.BandwidthPoint
		var ts string
		if err := rows.Scan(&ts, &p.AvgMbps, &p.MaxMbps); err != nil {
			return nil, err
		}
		p.Ts, _ = time.Parse("2006-01-02 15:04:05", ts)
		result = append(result, p)
	}
	return result, rows.Err()
}

func (s *bandwidthStore) PurgeOlderThan(ctx context.Context, before time.Time) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM bandwidth_samples WHERE recorded_at < ?`, before.UTC())
	return err
}

func (s *bandwidthStore) TotalCurrent(ctx context.Context, since time.Time) (float64, error) {
	// Sum of the latest rate_mbps per agent
	row := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(rate_mbps),0) FROM (
			SELECT agent_id, rate_mbps FROM bandwidth_samples
			WHERE recorded_at >= ?
			GROUP BY agent_id
			HAVING recorded_at=MAX(recorded_at)
		)`, since.UTC())
	var total float64
	return total, row.Scan(&total)
}
