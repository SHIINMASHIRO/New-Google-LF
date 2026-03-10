package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/aven/ngoogle/internal/model"
)

type taskGroupStore struct{ db *sql.DB }

const taskGroupCols = `id,name,description,pool_ids_json,agent_id,execution_scope,target_rate_mbps,
start_at,end_at,duration_sec,total_bytes_target,total_requests_target,dispatch_rate_tpm,dispatch_batch_size,
distribution,jitter_pct,ramp_up_sec,ramp_down_sec,traffic_profile_id,concurrent_fragments,retries,
created_at,updated_at`

func (s *taskGroupStore) Create(ctx context.Context, g *model.TaskGroup) error {
	g.Normalize()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO task_groups (id,name,description,pool_ids_json,agent_id,execution_scope,target_rate_mbps,
			start_at,end_at,duration_sec,total_bytes_target,total_requests_target,dispatch_rate_tpm,dispatch_batch_size,
			distribution,jitter_pct,ramp_up_sec,ramp_down_sec,traffic_profile_id,concurrent_fragments,retries,
			created_at,updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		g.ID, g.Name, g.Description, g.PoolIDsJSON, g.AgentID, g.ExecutionScope, g.TargetRateMbps,
		nullTime(g.StartAt), nullTime(g.EndAt), g.DurationSec, g.TotalBytesTarget, g.TotalRequestsTarget,
		g.DispatchRateTpm, g.DispatchBatchSize, g.Distribution, g.JitterPct, g.RampUpSec, g.RampDownSec,
		g.TrafficProfileID, g.ConcurrentFragments, g.Retries, g.CreatedAt.UTC(), g.UpdatedAt.UTC(),
	)
	return err
}

func (s *taskGroupStore) Get(ctx context.Context, id string) (*model.TaskGroup, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+taskGroupCols+` FROM task_groups WHERE id=?`, id)
	return scanTaskGroup(row)
}

func (s *taskGroupStore) List(ctx context.Context) ([]*model.TaskGroup, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+taskGroupCols+` FROM task_groups ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*model.TaskGroup
	for rows.Next() {
		g, err := scanTaskGroup(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, g)
	}
	return list, rows.Err()
}

func (s *taskGroupStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM task_groups WHERE id=?`, id)
	return err
}

func scanTaskGroup(row scanner) (*model.TaskGroup, error) {
	g := &model.TaskGroup{}
	var startAt, endAt sql.NullTime
	err := row.Scan(
		&g.ID, &g.Name, &g.Description, &g.PoolIDsJSON, &g.AgentID, &g.ExecutionScope, &g.TargetRateMbps,
		&startAt, &endAt, &g.DurationSec, &g.TotalBytesTarget, &g.TotalRequestsTarget, &g.DispatchRateTpm, &g.DispatchBatchSize,
		&g.Distribution, &g.JitterPct, &g.RampUpSec, &g.RampDownSec, &g.TrafficProfileID, &g.ConcurrentFragments, &g.Retries,
		&g.CreatedAt, &g.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("task group not found")
	}
	if err != nil {
		return nil, err
	}
	g.StartAt = scanNullTime(startAt)
	g.EndAt = scanNullTime(endAt)
	g.Normalize()
	return g, nil
}
