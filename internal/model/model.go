package model

import "time"

// ─── Agent ──────────────────────────────────────────────────────────────────

type AgentStatus string

const (
	AgentStatusOnline  AgentStatus = "online"
	AgentStatusOffline AgentStatus = "offline"
)

type Agent struct {
	ID           string      `json:"id" db:"id"`
	Hostname     string      `json:"hostname" db:"hostname"`
	IP           string      `json:"ip" db:"ip"`
	Port         int         `json:"port" db:"port"`
	Token        string      `json:"token" db:"token"`
	Status       AgentStatus `json:"status" db:"status"`
	Version      string      `json:"version" db:"version"`
	CurrentRateMbps float64  `json:"current_rate_mbps" db:"current_rate_mbps"`
	LastHeartbeat time.Time  `json:"last_heartbeat" db:"last_heartbeat"`
	CreatedAt    time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at" db:"updated_at"`
}

// ─── Task ────────────────────────────────────────────────────────────────────

type TaskType string
type TaskStatus string
type Distribution string

const (
	TaskTypeYoutube TaskType = "youtube"
	TaskTypeStatic  TaskType = "static"

	TaskStatusPending   TaskStatus = "pending"
	TaskStatusDispatched TaskStatus = "dispatched"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusDone      TaskStatus = "done"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusStopped   TaskStatus = "stopped"

	DistributionFlat    Distribution = "flat"
	DistributionRamp    Distribution = "ramp"
	DistributionDiurnal Distribution = "diurnal"
)

type Task struct {
	ID                  string       `json:"id" db:"id"`
	Name                string       `json:"name" db:"name"`
	Type                TaskType     `json:"type" db:"type"`
	TargetURL           string       `json:"target_url" db:"target_url"`
	AgentID             string       `json:"agent_id" db:"agent_id"`
	Status              TaskStatus   `json:"status" db:"status"`
	TargetRateMbps      float64      `json:"target_rate_mbps" db:"target_rate_mbps"`
	StartAt             *time.Time   `json:"start_at,omitempty" db:"start_at"`
	EndAt               *time.Time   `json:"end_at,omitempty" db:"end_at"`
	DurationSec         int          `json:"duration_sec" db:"duration_sec"`
	TotalBytesTarget    int64        `json:"total_bytes_target" db:"total_bytes_target"`
	TotalRequestsTarget int64        `json:"total_requests_target" db:"total_requests_target"`
	DispatchRateTpm     int          `json:"dispatch_rate_tpm" db:"dispatch_rate_tpm"`
	DispatchBatchSize   int          `json:"dispatch_batch_size" db:"dispatch_batch_size"`
	Distribution        Distribution `json:"distribution" db:"distribution"`
	JitterPct           float64      `json:"jitter_pct" db:"jitter_pct"`
	RampUpSec           int          `json:"ramp_up_sec" db:"ramp_up_sec"`
	RampDownSec         int          `json:"ramp_down_sec" db:"ramp_down_sec"`
	TrafficProfileID    string       `json:"traffic_profile_id" db:"traffic_profile_id"`
	ConcurrentFragments int          `json:"concurrent_fragments" db:"concurrent_fragments"`
	Retries             int          `json:"retries" db:"retries"`
	TotalBytesDone      int64        `json:"total_bytes_done" db:"total_bytes_done"`
	ErrorMessage        string       `json:"error_message,omitempty" db:"error_message"`
	DispatchedAt        *time.Time   `json:"dispatched_at,omitempty" db:"dispatched_at"`
	StartedAt           *time.Time   `json:"started_at,omitempty" db:"started_at"`
	FinishedAt          *time.Time   `json:"finished_at,omitempty" db:"finished_at"`
	CreatedAt           time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time    `json:"updated_at" db:"updated_at"`
}

// ─── Task Metrics ─────────────────────────────────────────────────────────────

type TaskMetrics struct {
	ID           int64     `json:"id" db:"id"`
	TaskID       string    `json:"task_id" db:"task_id"`
	AgentID      string    `json:"agent_id" db:"agent_id"`
	BytesTotal   int64     `json:"bytes_total" db:"bytes_total"`
	BytesDelta   int64     `json:"bytes_delta" db:"bytes_delta"`
	RateMbps5s   float64   `json:"rate_mbps_5s" db:"rate_mbps_5s"`
	RateMbps30s  float64   `json:"rate_mbps_30s" db:"rate_mbps_30s"`
	RequestCount int64     `json:"request_count" db:"request_count"`
	ErrorCount   int64     `json:"error_count" db:"error_count"`
	RecordedAt   time.Time `json:"recorded_at" db:"recorded_at"`
}

// ─── Traffic Profile ─────────────────────────────────────────────────────────

type TrafficProfile struct {
	ID          string       `json:"id" db:"id"`
	Name        string       `json:"name" db:"name"`
	Description string       `json:"description" db:"description"`
	Distribution Distribution `json:"distribution" db:"distribution"`
	// Points is a JSON array of {offset_sec, rate_pct} for diurnal curves
	Points    string    `json:"points" db:"points"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// ─── Provision Job ───────────────────────────────────────────────────────────

type ProvisionStatus string

const (
	ProvisionStatusPending  ProvisionStatus = "pending"
	ProvisionStatusRunning  ProvisionStatus = "running"
	ProvisionStatusSuccess  ProvisionStatus = "success"
	ProvisionStatusFailed   ProvisionStatus = "failed"
)

type AuthType string

const (
	AuthTypeKey      AuthType = "key"
	AuthTypePassword AuthType = "password"
)

type ProvisionJob struct {
	ID            string          `json:"id" db:"id"`
	HostIP        string          `json:"host_ip" db:"host_ip"`
	SSHPort       int             `json:"ssh_port" db:"ssh_port"`
	SSHUser       string          `json:"ssh_user" db:"ssh_user"`
	AuthType      AuthType        `json:"auth_type" db:"auth_type"`
	CredentialRef string          `json:"credential_ref" db:"credential_ref"`
	Status        ProvisionStatus `json:"status" db:"status"`
	CurrentStep   string          `json:"current_step" db:"current_step"`
	Log           string          `json:"log" db:"log"`
	AgentID       string          `json:"agent_id,omitempty" db:"agent_id"`
	FailedStep    string          `json:"failed_step,omitempty" db:"failed_step"`
	CreatedAt     time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at" db:"updated_at"`
}

// ─── Bandwidth Sample ─────────────────────────────────────────────────────────

type BandwidthSample struct {
	ID          int64     `json:"id" db:"id"`
	AgentID     string    `json:"agent_id" db:"agent_id"`
	RateMbps    float64   `json:"rate_mbps" db:"rate_mbps"`
	RecordedAt  time.Time `json:"recorded_at" db:"recorded_at"`
}

// ─── Credential ───────────────────────────────────────────────────────────────

type Credential struct {
	ID        string    `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	Type      AuthType  `json:"type" db:"type"`
	Payload   string    `json:"-" db:"payload"` // encrypted at rest
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}
