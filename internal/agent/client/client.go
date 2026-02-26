// Package client provides the HTTP client for communicating with the Master.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/aven/ngoogle/internal/model"
)

// Client is the Master API client used by agents.
type Client struct {
	baseURL    string
	agentID    string
	token      string
	httpClient *http.Client
}

// New creates a new Client.
func New(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// RegisterResponse is returned by the register endpoint.
type RegisterResponse struct {
	ID    string `json:"id"`
	Token string `json:"token"`
}

// Register registers this agent with the Master.
func (c *Client) Register(ctx context.Context, hostname, ip string, port int, version string) (*RegisterResponse, error) {
	body := map[string]interface{}{
		"hostname": hostname,
		"ip":       ip,
		"port":     port,
		"version":  version,
	}
	var resp RegisterResponse
	if err := c.post(ctx, "/api/v1/agents/register", body, &resp); err != nil {
		return nil, err
	}
	c.agentID = resp.ID
	c.token = resp.Token
	return &resp, nil
}

// Heartbeat sends a heartbeat to the Master.
func (c *Client) Heartbeat(ctx context.Context, rateMbps float64) error {
	body := map[string]interface{}{
		"agent_id":  c.agentID,
		"token":     c.token,
		"rate_mbps": rateMbps,
	}
	return c.post(ctx, "/api/v1/agents/heartbeat", body, nil)
}

// PullTasks fetches tasks assigned to this agent.
func (c *Client) PullTasks(ctx context.Context) ([]*model.Task, error) {
	if c.agentID == "" {
		return nil, fmt.Errorf("not registered")
	}
	url := fmt.Sprintf("/api/v1/agents/%s/tasks/pull", c.agentID)
	var tasks []*model.Task
	if err := c.get(ctx, url, &tasks); err != nil {
		return nil, err
	}
	return tasks, nil
}

// ReportMetrics sends task metrics to the Master.
func (c *Client) ReportMetrics(ctx context.Context, m *model.TaskMetrics) error {
	url := fmt.Sprintf("/api/v1/tasks/%s/metrics", m.TaskID)
	return c.post(ctx, url, m, nil)
}

// MarkRunning marks a task as running.
func (c *Client) MarkRunning(ctx context.Context, taskID string) error {
	return c.post(ctx, fmt.Sprintf("/api/v1/tasks/%s/run", taskID), nil, nil)
}

// AgentID returns the agent's assigned ID.
func (c *Client) AgentID() string { return c.agentID }

// ─── HTTP helpers ─────────────────────────────────────────────────────────────

func (c *Client) post(ctx context.Context, path string, body, resp interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bodyReader)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.do(req, resp)
}

func (c *Client) get(ctx context.Context, path string, resp interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	return c.do(req, resp)
}

func (c *Client) do(req *http.Request, out interface{}) error {
	res, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http %s %s: %w", req.Method, req.URL, err)
	}
	defer res.Body.Close()
	if res.StatusCode >= 400 {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("http %d: %s", res.StatusCode, string(body))
	}
	if out != nil {
		return json.NewDecoder(res.Body).Decode(out)
	}
	return nil
}
