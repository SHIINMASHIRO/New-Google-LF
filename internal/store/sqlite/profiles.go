package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/aven/ngoogle/internal/model"
)

// ─── Traffic Profile ──────────────────────────────────────────────────────────

type trafficProfileStore struct{ db *sql.DB }

func (s *trafficProfileStore) Create(ctx context.Context, p *model.TrafficProfile) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO traffic_profiles(id,name,description,distribution,points,created_at) VALUES(?,?,?,?,?,?)`,
		p.ID, p.Name, p.Description, p.Distribution, p.Points, p.CreatedAt.UTC())
	return err
}

func (s *trafficProfileStore) Get(ctx context.Context, id string) (*model.TrafficProfile, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id,name,description,distribution,points,created_at FROM traffic_profiles WHERE id=?`, id)
	p := &model.TrafficProfile{}
	err := row.Scan(&p.ID, &p.Name, &p.Description, &p.Distribution, &p.Points, &p.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("profile not found")
	}
	return p, err
}

func (s *trafficProfileStore) List(ctx context.Context) ([]*model.TrafficProfile, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,name,description,distribution,points,created_at FROM traffic_profiles ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*model.TrafficProfile
	for rows.Next() {
		p := &model.TrafficProfile{}
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Distribution, &p.Points, &p.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	return list, rows.Err()
}

// ─── Provision Job ────────────────────────────────────────────────────────────

type provisionJobStore struct{ db *sql.DB }

func (s *provisionJobStore) Create(ctx context.Context, j *model.ProvisionJob) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO provision_jobs(id,host_ip,ssh_port,ssh_user,auth_type,credential_ref,status,current_step,log,agent_id,failed_step,created_at,updated_at)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		j.ID, j.HostIP, j.SSHPort, j.SSHUser, j.AuthType, j.CredentialRef,
		j.Status, j.CurrentStep, j.Log, j.AgentID, j.FailedStep,
		j.CreatedAt.UTC(), j.UpdatedAt.UTC())
	return err
}

func (s *provisionJobStore) Get(ctx context.Context, id string) (*model.ProvisionJob, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id,host_ip,ssh_port,ssh_user,auth_type,credential_ref,status,current_step,log,agent_id,failed_step,created_at,updated_at
		 FROM provision_jobs WHERE id=?`, id)
	return scanProvisionJob(row)
}

func (s *provisionJobStore) List(ctx context.Context) ([]*model.ProvisionJob, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id,host_ip,ssh_port,ssh_user,auth_type,credential_ref,status,current_step,log,agent_id,failed_step,created_at,updated_at
		 FROM provision_jobs ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*model.ProvisionJob
	for rows.Next() {
		j, err := scanProvisionJob(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, j)
	}
	return list, rows.Err()
}

func (s *provisionJobStore) UpdateStatus(ctx context.Context, id string, status model.ProvisionStatus, step string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE provision_jobs SET status=?,current_step=?,updated_at=? WHERE id=?`,
		status, step, time.Now().UTC(), id)
	return err
}

func (s *provisionJobStore) AppendLog(ctx context.Context, id string, line string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE provision_jobs SET log=log||?||char(10),updated_at=? WHERE id=?`,
		line, time.Now().UTC(), id)
	return err
}

func (s *provisionJobStore) SetAgentID(ctx context.Context, id string, agentID string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE provision_jobs SET agent_id=?,updated_at=? WHERE id=?`, agentID, time.Now().UTC(), id)
	return err
}

func (s *provisionJobStore) SetFailed(ctx context.Context, id string, step string, reason string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE provision_jobs SET status='failed',failed_step=?,log=log||?||char(10),updated_at=? WHERE id=?`,
		step, "[FAIL] "+reason, time.Now().UTC(), id)
	return err
}

func (s *provisionJobStore) ResetForRetry(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE provision_jobs SET status=?,current_step='created',log='',agent_id='',failed_step='',updated_at=? WHERE id=?`,
		model.ProvisionStatusPending, time.Now().UTC(), id)
	return err
}

func (s *provisionJobStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM provision_jobs WHERE id=?`, id)
	return err
}

func scanProvisionJob(row scanner) (*model.ProvisionJob, error) {
	j := &model.ProvisionJob{}
	err := row.Scan(&j.ID, &j.HostIP, &j.SSHPort, &j.SSHUser, &j.AuthType, &j.CredentialRef,
		&j.Status, &j.CurrentStep, &j.Log, &j.AgentID, &j.FailedStep, &j.CreatedAt, &j.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("provision job not found")
	}
	return j, err
}

// ─── Credentials ─────────────────────────────────────────────────────────────

type credentialStore struct{ db *sql.DB }

func (s *credentialStore) Create(ctx context.Context, c *model.Credential) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO credentials(id,name,type,payload,created_at) VALUES(?,?,?,?,?)`,
		c.ID, c.Name, c.Type, c.Payload, c.CreatedAt.UTC())
	return err
}

func (s *credentialStore) Get(ctx context.Context, id string) (*model.Credential, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id,name,type,payload,created_at FROM credentials WHERE id=?`, id)
	c := &model.Credential{}
	err := row.Scan(&c.ID, &c.Name, &c.Type, &c.Payload, &c.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("credential not found")
	}
	return c, err
}

func (s *credentialStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM credentials WHERE id=?`, id)
	return err
}

func (s *credentialStore) List(ctx context.Context) ([]*model.Credential, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,name,type,payload,created_at FROM credentials ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*model.Credential
	for rows.Next() {
		c := &model.Credential{}
		if err := rows.Scan(&c.ID, &c.Name, &c.Type, &c.Payload, &c.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, rows.Err()
}
