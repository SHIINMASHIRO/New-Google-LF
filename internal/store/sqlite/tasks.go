package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/aven/ngoogle/internal/model"
)

type taskStore struct{ db *sql.DB }

const taskCols = `id,name,type,target_url,agent_id,status,target_rate_mbps,
start_at,end_at,duration_sec,total_bytes_target,total_requests_target,
dispatch_rate_tpm,dispatch_batch_size,distribution,jitter_pct,ramp_up_sec,ramp_down_sec,
traffic_profile_id,concurrent_fragments,retries,total_bytes_done,error_message,
dispatched_at,started_at,finished_at,created_at,updated_at`

func (s *taskStore) Create(ctx context.Context, t *model.Task) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO tasks (id,name,type,target_url,agent_id,status,target_rate_mbps,
			start_at,end_at,duration_sec,total_bytes_target,total_requests_target,
			dispatch_rate_tpm,dispatch_batch_size,distribution,jitter_pct,ramp_up_sec,ramp_down_sec,
			traffic_profile_id,concurrent_fragments,retries,total_bytes_done,error_message,
			dispatched_at,started_at,finished_at,created_at,updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		t.ID, t.Name, t.Type, t.TargetURL, t.AgentID, t.Status, t.TargetRateMbps,
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
	row := s.db.QueryRowContext(ctx, `SELECT `+taskCols+` FROM tasks WHERE id=?`, id)
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

func (s *taskStore) ListByAgent(ctx context.Context, agentID string, statuses []model.TaskStatus) ([]*model.Task, error) {
	placeholders := make([]string, len(statuses))
	args := []interface{}{agentID}
	for i, st := range statuses {
		placeholders[i] = "?"
		args = append(args, st)
	}
	q := fmt.Sprintf(`SELECT %s FROM tasks WHERE agent_id=? AND status IN (%s) ORDER BY created_at ASC`,
		taskCols, strings.Join(placeholders, ","))
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTasks(rows)
}

func (s *taskStore) UpdateStatus(ctx context.Context, id string, status model.TaskStatus) error {
	_, err := s.db.ExecContext(ctx, `UPDATE tasks SET status=?,updated_at=? WHERE id=?`, status, time.Now().UTC(), id)
	return err
}

func (s *taskStore) UpdateStatusWithTime(ctx context.Context, id string, status model.TaskStatus, ts time.Time, field string) error {
	q := fmt.Sprintf(`UPDATE tasks SET status=?,%s=?,updated_at=? WHERE id=?`, field)
	_, err := s.db.ExecContext(ctx, q, status, ts.UTC(), time.Now().UTC(), id)
	return err
}

func (s *taskStore) UpdateBytes(ctx context.Context, id string, bytesTotal int64) error {
	_, err := s.db.ExecContext(ctx, `UPDATE tasks SET total_bytes_done=?,updated_at=? WHERE id=?`, bytesTotal, time.Now().UTC(), id)
	return err
}

func (s *taskStore) SetError(ctx context.Context, id string, msg string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE tasks SET error_message=?,updated_at=? WHERE id=?`, msg, time.Now().UTC(), id)
	return err
}

func (s *taskStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM tasks WHERE id=?`, id)
	return err
}

func scanTask(row scanner) (*model.Task, error) {
	t := &model.Task{}
	var startAt, endAt, dispatchedAt, startedAt, finishedAt sql.NullTime
	err := row.Scan(
		&t.ID, &t.Name, &t.Type, &t.TargetURL, &t.AgentID, &t.Status, &t.TargetRateMbps,
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
