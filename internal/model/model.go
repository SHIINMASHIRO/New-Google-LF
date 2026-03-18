package model

import (
	"encoding/json"
	"strings"
	"time"
)

// ─── Agent ──────────────────────────────────────────────────────────────────

type AgentStatus string

const (
	AgentStatusOnline  AgentStatus = "online"
	AgentStatusOffline AgentStatus = "offline"
)

type Agent struct {
	ID              string      `json:"id" db:"id"`
	Hostname        string      `json:"hostname" db:"hostname"`
	IP              string      `json:"ip" db:"ip"`
	Port            int         `json:"port" db:"port"`
	Token           string      `json:"token" db:"token"`
	Status          AgentStatus `json:"status" db:"status"`
	Version         string      `json:"version" db:"version"`
	CurrentRateMbps float64     `json:"current_rate_mbps" db:"current_rate_mbps"`
	LastHeartbeat   time.Time   `json:"last_heartbeat" db:"last_heartbeat"`
	CreatedAt       time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time   `json:"updated_at" db:"updated_at"`
}

// ─── Task ────────────────────────────────────────────────────────────────────

type TaskType string
type TaskStatus string
type Distribution string
type TaskExecutionScope string
type URLPoolType string

const (
	TaskTypeYoutube TaskType = "youtube"
	TaskTypeStatic  TaskType = "static"
	TaskTypeMixed   TaskType = "mixed"

	TaskStatusPending    TaskStatus = "pending"
	TaskStatusDispatched TaskStatus = "dispatched"
	TaskStatusRunning    TaskStatus = "running"
	TaskStatusDone       TaskStatus = "done"
	TaskStatusFailed     TaskStatus = "failed"
	TaskStatusStopped    TaskStatus = "stopped"

	DistributionFlat    Distribution = "flat"
	DistributionRamp    Distribution = "ramp"
	DistributionDiurnal Distribution = "diurnal"

	TaskExecutionScopeSingleAgent TaskExecutionScope = "single_agent"
	TaskExecutionScopeGlobal      TaskExecutionScope = "global"

	URLPoolTypeYoutube URLPoolType = "youtube"
	URLPoolTypeStatic  URLPoolType = "static"
)

type Task struct {
	ID                  string             `json:"id" db:"id"`
	GroupID             string             `json:"group_id,omitempty" db:"group_id"`
	Name                string             `json:"name" db:"name"`
	Type                TaskType           `json:"type" db:"type"`
	URLPoolID           string             `json:"url_pool_id" db:"url_pool_id"`
	TargetURL           string             `json:"target_url" db:"target_url"`
	TargetURLsJSON      string             `json:"-" db:"target_urls_json"`
	TargetURLs          []string           `json:"target_urls,omitempty" db:"-"`
	URLPool             *URLPool           `json:"url_pool,omitempty" db:"-"`
	AgentID             string             `json:"agent_id" db:"agent_id"`
	ExecutionScope      TaskExecutionScope `json:"execution_scope" db:"execution_scope"`
	Status              TaskStatus         `json:"status" db:"status"`
	TargetRateMbps      float64            `json:"target_rate_mbps" db:"target_rate_mbps"`
	StartAt             *time.Time         `json:"start_at,omitempty" db:"start_at"`
	EndAt               *time.Time         `json:"end_at,omitempty" db:"end_at"`
	DurationSec         int                `json:"duration_sec" db:"duration_sec"`
	TotalBytesTarget    int64              `json:"total_bytes_target" db:"total_bytes_target"`
	TotalRequestsTarget int64              `json:"total_requests_target" db:"total_requests_target"`
	DispatchRateTpm     int                `json:"dispatch_rate_tpm" db:"dispatch_rate_tpm"`
	DispatchBatchSize   int                `json:"dispatch_batch_size" db:"dispatch_batch_size"`
	Distribution        Distribution       `json:"distribution" db:"distribution"`
	JitterPct           float64            `json:"jitter_pct" db:"jitter_pct"`
	RampUpSec           int                `json:"ramp_up_sec" db:"ramp_up_sec"`
	RampDownSec         int                `json:"ramp_down_sec" db:"ramp_down_sec"`
	TrafficProfileID    string             `json:"traffic_profile_id" db:"traffic_profile_id"`
	ConcurrentFragments int                `json:"concurrent_fragments" db:"concurrent_fragments"`
	Retries             int                `json:"retries" db:"retries"`
	TotalBytesDone      int64              `json:"total_bytes_done" db:"total_bytes_done"`
	ErrorMessage        string             `json:"error_message,omitempty" db:"error_message"`
	DispatchedAt        *time.Time         `json:"dispatched_at,omitempty" db:"dispatched_at"`
	StartedAt           *time.Time         `json:"started_at,omitempty" db:"started_at"`
	FinishedAt          *time.Time         `json:"finished_at,omitempty" db:"finished_at"`
	CreatedAt           time.Time          `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time          `json:"updated_at" db:"updated_at"`
}

type TaskGroup struct {
	ID                  string             `json:"id" db:"id"`
	Name                string             `json:"name" db:"name"`
	Description         string             `json:"description" db:"description"`
	PoolIDsJSON         string             `json:"-" db:"pool_ids_json"`
	PoolIDs             []string           `json:"pool_ids" db:"-"`
	AgentID             string             `json:"agent_id" db:"agent_id"`
	ExecutionScope      TaskExecutionScope `json:"execution_scope" db:"execution_scope"`
	TargetRateMbps      float64            `json:"target_rate_mbps" db:"target_rate_mbps"`
	StartAt             *time.Time         `json:"start_at,omitempty" db:"start_at"`
	EndAt               *time.Time         `json:"end_at,omitempty" db:"end_at"`
	DurationSec         int                `json:"duration_sec" db:"duration_sec"`
	TotalBytesTarget    int64              `json:"total_bytes_target" db:"total_bytes_target"`
	TotalRequestsTarget int64              `json:"total_requests_target" db:"total_requests_target"`
	DispatchRateTpm     int                `json:"dispatch_rate_tpm" db:"dispatch_rate_tpm"`
	DispatchBatchSize   int                `json:"dispatch_batch_size" db:"dispatch_batch_size"`
	Distribution        Distribution       `json:"distribution" db:"distribution"`
	JitterPct           float64            `json:"jitter_pct" db:"jitter_pct"`
	RampUpSec           int                `json:"ramp_up_sec" db:"ramp_up_sec"`
	RampDownSec         int                `json:"ramp_down_sec" db:"ramp_down_sec"`
	TrafficProfileID    string             `json:"traffic_profile_id" db:"traffic_profile_id"`
	ConcurrentFragments int                `json:"concurrent_fragments" db:"concurrent_fragments"`
	Retries             int                `json:"retries" db:"retries"`
	CreatedAt           time.Time          `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time          `json:"updated_at" db:"updated_at"`

	Status         TaskStatus `json:"status" db:"-"`
	Type           TaskType   `json:"type" db:"-"`
	TotalBytesDone int64      `json:"total_bytes_done" db:"-"`
	PoolCount      int        `json:"pool_count,omitempty" db:"-"`
	ChildCount     int        `json:"child_count,omitempty" db:"-"`
	Children       []*Task    `json:"children,omitempty" db:"-"`
	Pools          []*URLPool `json:"pools,omitempty" db:"-"`
}

type URLPool struct {
	ID          string      `json:"id" db:"id"`
	Name        string      `json:"name" db:"name"`
	Type        URLPoolType `json:"type" db:"type"`
	Description string      `json:"description" db:"description"`
	URLsJSON    string      `json:"-" db:"urls_json"`
	URLs        []string    `json:"urls" db:"-"`
	CreatedAt   time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at" db:"updated_at"`
}

func (t *Task) Normalize() {
	if t.ExecutionScope == "" {
		t.ExecutionScope = TaskExecutionScopeSingleAgent
	}
	if t.URLPool != nil {
		t.URLPool.Normalize()
		if len(t.TargetURLs) == 0 {
			t.TargetURLs = append([]string(nil), t.URLPool.URLs...)
		}
		if t.Type == "" {
			t.Type = t.URLPool.TaskType()
		}
	}
	if len(t.TargetURLs) == 0 && t.TargetURLsJSON != "" {
		var urls []string
		if err := json.Unmarshal([]byte(t.TargetURLsJSON), &urls); err == nil {
			t.TargetURLs = sanitizeURLs(urls)
		}
	}
	if len(t.TargetURLs) == 0 && t.TargetURL != "" {
		t.TargetURLs = []string{t.TargetURL}
	}
	if t.TargetURL == "" && len(t.TargetURLs) > 0 {
		t.TargetURL = t.TargetURLs[0]
	}
	if t.TargetURLsJSON == "" {
		t.syncTargetURLsJSON()
	}
}

func (t *Task) SetTargetURLs(urls []string) {
	t.TargetURLs = sanitizeURLs(urls)
	if len(t.TargetURLs) > 0 {
		t.TargetURL = t.TargetURLs[0]
	}
	t.syncTargetURLsJSON()
}

func (t *Task) URLs() []string {
	t.Normalize()
	if len(t.TargetURLs) == 0 {
		return nil
	}
	out := make([]string, len(t.TargetURLs))
	copy(out, t.TargetURLs)
	return out
}

func (t *Task) Clone() *Task {
	if t == nil {
		return nil
	}
	cp := *t
	if len(t.TargetURLs) > 0 {
		cp.TargetURLs = append([]string(nil), t.TargetURLs...)
	}
	if t.URLPool != nil {
		cp.URLPool = t.URLPool.Clone()
	}
	return &cp
}

func (g *TaskGroup) Normalize() {
	if g.ExecutionScope == "" {
		g.ExecutionScope = TaskExecutionScopeSingleAgent
	}
	if len(g.PoolIDs) == 0 && g.PoolIDsJSON != "" {
		var ids []string
		if err := json.Unmarshal([]byte(g.PoolIDsJSON), &ids); err == nil {
			g.PoolIDs = sanitizeURLs(ids)
		}
	}
	if g.PoolIDsJSON == "" {
		g.syncPoolIDsJSON()
	}
}

func (g *TaskGroup) SetPoolIDs(ids []string) {
	g.PoolIDs = sanitizeURLs(ids)
	g.syncPoolIDsJSON()
}

func (g *TaskGroup) Clone() *TaskGroup {
	if g == nil {
		return nil
	}
	cp := *g
	if len(g.PoolIDs) > 0 {
		cp.PoolIDs = append([]string(nil), g.PoolIDs...)
	}
	if len(g.Children) > 0 {
		cp.Children = make([]*Task, 0, len(g.Children))
		for _, child := range g.Children {
			cp.Children = append(cp.Children, child.Clone())
		}
	}
	if len(g.Pools) > 0 {
		cp.Pools = make([]*URLPool, 0, len(g.Pools))
		for _, pool := range g.Pools {
			cp.Pools = append(cp.Pools, pool.Clone())
		}
	}
	return &cp
}

func (g *TaskGroup) syncPoolIDsJSON() {
	raw, err := json.Marshal(g.PoolIDs)
	if err != nil {
		g.PoolIDsJSON = "[]"
		return
	}
	g.PoolIDsJSON = string(raw)
}

func (t *Task) syncTargetURLsJSON() {
	raw, err := json.Marshal(t.TargetURLs)
	if err != nil {
		t.TargetURLsJSON = "[]"
		return
	}
	t.TargetURLsJSON = string(raw)
}

func sanitizeURLs(urls []string) []string {
	if len(urls) == 0 {
		return nil
	}
	out := make([]string, 0, len(urls))
	seen := make(map[string]struct{}, len(urls))
	for _, u := range urls {
		u = strings.TrimSpace(u)
		if u == "" {
			continue
		}
		if _, ok := seen[u]; ok {
			continue
		}
		seen[u] = struct{}{}
		out = append(out, u)
	}
	return out
}

func (p *URLPool) Normalize() {
	if p == nil {
		return
	}
	if len(p.URLs) == 0 && p.URLsJSON != "" {
		var urls []string
		if err := json.Unmarshal([]byte(p.URLsJSON), &urls); err == nil {
			p.URLs = sanitizeURLs(urls)
		}
	}
	if p.URLsJSON == "" {
		p.syncURLsJSON()
	}
}

func (p *URLPool) SetURLs(urls []string) {
	p.URLs = sanitizeURLs(urls)
	p.syncURLsJSON()
}

func (p *URLPool) Clone() *URLPool {
	if p == nil {
		return nil
	}
	cp := *p
	if len(p.URLs) > 0 {
		cp.URLs = append([]string(nil), p.URLs...)
	}
	return &cp
}

func (p *URLPool) syncURLsJSON() {
	raw, err := json.Marshal(p.URLs)
	if err != nil {
		p.URLsJSON = "[]"
		return
	}
	p.URLsJSON = string(raw)
}

func (p *URLPool) TaskType() TaskType {
	switch p.Type {
	case URLPoolTypeYoutube:
		return TaskTypeYoutube
	default:
		return TaskTypeStatic
	}
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
	ID           string       `json:"id" db:"id"`
	Name         string       `json:"name" db:"name"`
	Description  string       `json:"description" db:"description"`
	Distribution Distribution `json:"distribution" db:"distribution"`
	// Points is a JSON array of {offset_sec, rate_pct} for diurnal curves
	Points    string    `json:"points" db:"points"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// ─── Provision Job ───────────────────────────────────────────────────────────

type ProvisionStatus string

const (
	ProvisionStatusPending ProvisionStatus = "pending"
	ProvisionStatusRunning ProvisionStatus = "running"
	ProvisionStatusSuccess ProvisionStatus = "success"
	ProvisionStatusFailed  ProvisionStatus = "failed"
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
	ID         int64     `json:"id" db:"id"`
	AgentID    string    `json:"agent_id" db:"agent_id"`
	RateMbps   float64   `json:"rate_mbps" db:"rate_mbps"`
	RecordedAt time.Time `json:"recorded_at" db:"recorded_at"`
}

// ─── Credential ───────────────────────────────────────────────────────────────

type Credential struct {
	ID        string    `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	Type      AuthType  `json:"type" db:"type"`
	Payload   string    `json:"-" db:"payload"` // encrypted at rest
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}
