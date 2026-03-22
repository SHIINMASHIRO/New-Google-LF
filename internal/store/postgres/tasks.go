package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/aven/ngoogle/internal/model"
)

type taskStore struct{ db *sql.DB }

const taskCols = `id,group_id,name,type,url_pool_id,target_url,target_urls_json,agent_id,execution_scope,status,target_rate_mbps,
start_at,end_at,duration_sec,total_bytes_target,total_requests_target,
dispatch_rate_tpm,dispatch_batch_size,distribution,jitter_pct,ramp_up_sec,ramp_down_sec,
traffic_profile_id,concurrent_fragments,retries,total_bytes_done,error_message,
dispatched_at,started_at,finished_at,created_at,updated_at`

func (s *taskStore) Create(ctx context.Context, t *model.Task) error {
	t.Normalize()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO tasks (id,group_id,name,type,url_pool_id,target_url,target_urls_json,agent_id,execution_scope,status,target_rate_mbps,
			start_at,end_at,duration_sec,total_bytes_target,total_requests_target,
			dispatch_rate_tpm,dispatch_batch_size,distribution,jitter_pct,ramp_up_sec,ramp_down_sec,
			traffic_profile_id,concurrent_fragments,retries,total_bytes_done,error_message,
			dispatched_at,started_at,finished_at,created_at,updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,$31,$32)`,
		t.ID, t.GroupID, t.Name, t.Type, t.URLPoolID, t.TargetURL, t.TargetURLsJSON, t.AgentID, t.ExecutionScope, t.Status, t.TargetRateMbps,
		nullTime(t.StartAt), nullTime(t.EndAt), t.DurationSec,
		t.TotalBytesTarget, t.TotalRequestsTarget,
		t.DispatchRateTpm, t.DispatchBatchSize, t.Distribution,
		t.JitterPct, t.RampUpSec, t.RampDownSec,
		t.TrafficProfileID, t.ConcurrentFragments, t.Retries,
		t.TotalBytesDone, t.ErrorMessage,
		nullTime(t.DispatchedAt), nullTime(t.StartedAt), nullTime(t.FinishedAt),
		t.CreatedAt.UTC(), t.UpdatedAt.UTC(),
	)
	return err
}

func (s *taskStore) Get(ctx context.Context, id string) (*model.Task, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+taskCols+` FROM tasks WHERE id=$1`, id)
	return scanTask(row)
}

func (s *taskStore) List(ctx context.Context) ([]*model.Task, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+taskCols+` FROM tasks ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTasks(rows)
}

func (s *taskStore) ListByGroup(ctx context.Context, groupID string) ([]*model.Task, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+taskCols+` FROM tasks WHERE group_id=$1 ORDER BY created_at ASC`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTasks(rows)
}

func (s *taskStore) ListByAgent(ctx context.Context, agentID string, statuses []model.TaskStatus) ([]*model.Task, error) {
	placeholders := make([]string, len(statuses))
	args := []interface{}{agentID}
	for i, st := range statuses {
		placeholders[i] = fmt.Sprintf("$%d", i+2)
		args = append(args, st)
	}
	q := fmt.Sprintf(`SELECT %s FROM tasks WHERE agent_id=$1 AND status IN (%s) ORDER BY created_at ASC`,
		taskCols, strings.Join(placeholders, ","))
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTasks(rows)
}

func (s *taskStore) UpdateStatus(ctx context.Context, id string, status model.TaskStatus) error {
	_, err := s.db.ExecContext(ctx, `UPDATE tasks SET status=$1,updated_at=$2 WHERE id=$3`, status, time.Now().UTC(), id)
	return err
}

func (s *taskStore) UpdateStatusWithTime(ctx context.Context, id string, status model.TaskStatus, ts time.Time, field string) error {
	q := fmt.Sprintf(`UPDATE tasks SET status=$1,%s=$2,updated_at=$3 WHERE id=$4`, field)
	_, err := s.db.ExecContext(ctx, q, status, ts.UTC(), time.Now().UTC(), id)
	return err
}

func (s *taskStore) UpdateBytes(ctx context.Context, id string, bytesTotal int64) error {
	_, err := s.db.ExecContext(ctx, `UPDATE tasks SET total_bytes_done=$1,updated_at=$2 WHERE id=$3`, bytesTotal, time.Now().UTC(), id)
	return err
}

func (s *taskStore) SetError(ctx context.Context, id string, msg string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE tasks SET error_message=$1,updated_at=$2 WHERE id=$3`, msg, time.Now().UTC(), id)
	return err
}

func (s *taskStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM tasks WHERE id=$1`, id)
	return err
}

func scanTask(row scanner) (*model.Task, error) {
	t := &model.Task{}
	var startAt, endAt, dispatchedAt, startedAt, finishedAt sql.NullTime
	err := row.Scan(
		&t.ID, &t.GroupID, &t.Name, &t.Type, &t.URLPoolID, &t.TargetURL, &t.TargetURLsJSON, &t.AgentID, &t.ExecutionScope, &t.Status, &t.TargetRateMbps,
		&startAt, &endAt, &t.DurationSec,
		&t.TotalBytesTarget, &t.TotalRequestsTarget,
		&t.DispatchRateTpm, &t.DispatchBatchSize, &t.Distribution,
		&t.JitterPct, &t.RampUpSec, &t.RampDownSec,
		&t.TrafficProfileID, &t.ConcurrentFragments, &t.Retries,
		&t.TotalBytesDone, &t.ErrorMessage,
		&dispatchedAt, &startedAt, &finishedAt,
		&t.CreatedAt, &t.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("task not found")
	}
	if err != nil {
		return nil, err
	}
	t.StartAt = scanNullTime(startAt)
	t.EndAt = scanNullTime(endAt)
	t.DispatchedAt = scanNullTime(dispatchedAt)
	t.StartedAt = scanNullTime(startedAt)
	t.FinishedAt = scanNullTime(finishedAt)
	t.Normalize()
	return t, nil
}

func scanTasks(rows *sql.Rows) ([]*model.Task, error) {
	var list []*model.Task
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, t)
	}
	return list, rows.Err()
}
