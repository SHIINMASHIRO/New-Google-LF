package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/aven/ngoogle/internal/model"
)

type agentStore struct{ db *sql.DB }

func (s *agentStore) Upsert(ctx context.Context, a *model.Agent) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO agents (id, hostname, ip, port, token, status, version, current_rate_mbps, last_heartbeat, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		ON CONFLICT(id) DO UPDATE SET
			hostname=excluded.hostname, ip=excluded.ip, port=excluded.port,
			token=excluded.token, status=excluded.status, version=excluded.version,
			current_rate_mbps=excluded.current_rate_mbps,
			last_heartbeat=excluded.last_heartbeat, updated_at=excluded.updated_at`,
		a.ID, a.Hostname, a.IP, a.Port, a.Token, a.Status, a.Version,
		a.CurrentRateMbps, a.LastHeartbeat.UTC(), a.CreatedAt.UTC(), a.UpdatedAt.UTC(),
	)
	return err
}

func (s *agentStore) Get(ctx context.Context, id string) (*model.Agent, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id,hostname,ip,port,token,status,version,current_rate_mbps,last_heartbeat,created_at,updated_at FROM agents WHERE id=$1`, id)
	return scanAgent(row)
}

func (s *agentStore) List(ctx context.Context) ([]*model.Agent, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id,hostname,ip,port,token,status,version,current_rate_mbps,last_heartbeat,created_at,updated_at FROM agents ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*model.Agent
	for rows.Next() {
		a, err := scanAgent(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, a)
	}
	return list, rows.Err()
}

func (s *agentStore) UpdateStatus(ctx context.Context, id string, status model.AgentStatus, heartbeat time.Time) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE agents SET status=$1, last_heartbeat=$2, updated_at=$3 WHERE id=$4`,
		status, heartbeat.UTC(), time.Now().UTC(), id)
	return err
}

func (s *agentStore) UpdateRate(ctx context.Context, id string, rateMbps float64) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE agents SET current_rate_mbps=$1, updated_at=$2 WHERE id=$3`,
		rateMbps, time.Now().UTC(), id)
	return err
}

func (s *agentStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM agents WHERE id=$1`, id)
	return err
}

func scanAgent(row scanner) (*model.Agent, error) {
	a := &model.Agent{}
	err := row.Scan(&a.ID, &a.Hostname, &a.IP, &a.Port, &a.Token,
		&a.Status, &a.Version, &a.CurrentRateMbps,
		&a.LastHeartbeat, &a.CreatedAt, &a.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("agent not found")
	}
	return a, err
}
