package store

import (
	"context"
	"time"

	"github.com/aven/ngoogle/internal/model"
)

// AgentStore manages agent records.
type AgentStore interface {
	Upsert(ctx context.Context, a *model.Agent) error
	Get(ctx context.Context, id string) (*model.Agent, error)
	List(ctx context.Context) ([]*model.Agent, error)
	UpdateStatus(ctx context.Context, id string, status model.AgentStatus, heartbeat time.Time) error
	UpdateRate(ctx context.Context, id string, rateMbps float64) error
	Delete(ctx context.Context, id string) error
}

// TaskStore manages task records.
type TaskStore interface {
	Create(ctx context.Context, t *model.Task) error
	Get(ctx context.Context, id string) (*model.Task, error)
	List(ctx context.Context) ([]*model.Task, error)
	ListByAgent(ctx context.Context, agentID string, statuses []model.TaskStatus) ([]*model.Task, error)
	UpdateStatus(ctx context.Context, id string, status model.TaskStatus) error
	UpdateStatusWithTime(ctx context.Context, id string, status model.TaskStatus, ts time.Time, field string) error
	UpdateBytes(ctx context.Context, id string, bytesTotal int64) error
	SetError(ctx context.Context, id string, msg string) error
	Delete(ctx context.Context, id string) error
}

// TaskMetricsStore manages task metric samples.
type TaskMetricsStore interface {
	Insert(ctx context.Context, m *model.TaskMetrics) error
	ListByTask(ctx context.Context, taskID string, from, to time.Time) ([]*model.TaskMetrics, error)
	LatestByTask(ctx context.Context, taskID string) (*model.TaskMetrics, error)
}

// TrafficProfileStore manages traffic profile records.
type TrafficProfileStore interface {
	Create(ctx context.Context, p *model.TrafficProfile) error
	Get(ctx context.Context, id string) (*model.TrafficProfile, error)
	List(ctx context.Context) ([]*model.TrafficProfile, error)
}

// ProvisionJobStore manages provisioning job records.
type ProvisionJobStore interface {
	Create(ctx context.Context, j *model.ProvisionJob) error
	Get(ctx context.Context, id string) (*model.ProvisionJob, error)
	List(ctx context.Context) ([]*model.ProvisionJob, error)
	UpdateStatus(ctx context.Context, id string, status model.ProvisionStatus, step string) error
	AppendLog(ctx context.Context, id string, line string) error
	SetAgentID(ctx context.Context, id string, agentID string) error
	SetFailed(ctx context.Context, id string, step string, reason string) error
}

// BandwidthStore manages bandwidth samples.
type BandwidthStore interface {
	Insert(ctx context.Context, s *model.BandwidthSample) error
	History(ctx context.Context, agentID string, from, to time.Time) ([]*model.BandwidthSample, error)
	AggregateHistory(ctx context.Context, from, to time.Time, stepSec int) ([]BandwidthPoint, error)
	PurgeOlderThan(ctx context.Context, before time.Time) error
	TotalCurrent(ctx context.Context, since time.Time) (float64, error)
}

// CredentialStore manages credentials.
type CredentialStore interface {
	Create(ctx context.Context, c *model.Credential) error
	Get(ctx context.Context, id string) (*model.Credential, error)
	List(ctx context.Context) ([]*model.Credential, error)
}

// BandwidthPoint is a time-bucketed bandwidth data point.
type BandwidthPoint struct {
	Ts      time.Time `json:"ts"`
	AvgMbps float64   `json:"avg_mbps"`
	MaxMbps float64   `json:"max_mbps"`
}

// Store bundles all sub-stores.
type Store interface {
	Agents() AgentStore
	Tasks() TaskStore
	TaskMetrics() TaskMetricsStore
	TrafficProfiles() TrafficProfileStore
	ProvisionJobs() ProvisionJobStore
	Bandwidth() BandwidthStore
	Credentials() CredentialStore
	Close() error
}
