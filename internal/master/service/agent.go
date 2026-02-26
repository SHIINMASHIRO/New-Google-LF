package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"time"

	"github.com/aven/ngoogle/internal/model"
	"github.com/aven/ngoogle/internal/store"
)

// AgentService handles agent lifecycle.
type AgentService struct {
	store   store.Store
	timeout time.Duration // heartbeat timeout for offline detection
}

// NewAgentService creates a new AgentService.
func NewAgentService(st store.Store) *AgentService {
	return &AgentService{store: st, timeout: 30 * time.Second}
}

// Register registers a new agent or updates an existing one.
func (s *AgentService) Register(ctx context.Context, hostname, ip string, port int, version string) (*model.Agent, error) {
	// Check if agent with same hostname+ip exists
	agents, err := s.store.Agents().List(ctx)
	if err != nil {
		return nil, err
	}
	for _, a := range agents {
		if a.Hostname == hostname && a.IP == ip {
			// Re-register: update token + status
			a.Token = generateToken()
			a.Status = model.AgentStatusOnline
			a.LastHeartbeat = time.Now()
			a.Version = version
			a.UpdatedAt = time.Now()
			if err := s.store.Agents().Upsert(ctx, a); err != nil {
				return nil, err
			}
			return a, nil
		}
	}
	// New agent
	a := &model.Agent{
		ID:            generateID(),
		Hostname:      hostname,
		IP:            ip,
		Port:          port,
		Token:         generateToken(),
		Status:        model.AgentStatusOnline,
		Version:       version,
		LastHeartbeat: time.Now(),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	if err := s.store.Agents().Upsert(ctx, a); err != nil {
		return nil, err
	}
	return a, nil
}

// Heartbeat updates agent last-seen and status.
func (s *AgentService) Heartbeat(ctx context.Context, agentID string, rateMbps float64) error {
	now := time.Now()
	if err := s.store.Agents().UpdateStatus(ctx, agentID, model.AgentStatusOnline, now); err != nil {
		return err
	}
	if err := s.store.Agents().UpdateRate(ctx, agentID, rateMbps); err != nil {
		return err
	}
	// Record bandwidth sample
	return s.store.Bandwidth().Insert(ctx, &model.BandwidthSample{
		AgentID:    agentID,
		RateMbps:   rateMbps,
		RecordedAt: now,
	})
}

// RunOfflineDetection periodically marks agents that haven't sent heartbeats as offline.
func (s *AgentService) RunOfflineDetection(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.detectOffline(ctx)
		}
	}
}

func (s *AgentService) detectOffline(ctx context.Context) {
	agents, err := s.store.Agents().List(ctx)
	if err != nil {
		slog.Error("offline detection list", "err", err)
		return
	}
	threshold := time.Now().Add(-s.timeout)
	for _, a := range agents {
		if a.Status == model.AgentStatusOnline && a.LastHeartbeat.Before(threshold) {
			if err := s.store.Agents().UpdateStatus(ctx, a.ID, model.AgentStatusOffline, a.LastHeartbeat); err != nil {
				slog.Error("mark offline", "agent", a.ID, "err", err)
			}
		}
	}
}

// List returns all agents.
func (s *AgentService) List(ctx context.Context) ([]*model.Agent, error) {
	return s.store.Agents().List(ctx)
}

// Get returns a single agent.
func (s *AgentService) Get(ctx context.Context, id string) (*model.Agent, error) {
	return s.store.Agents().Get(ctx, id)
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func generateID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func generateToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// ValidateToken checks if the provided token matches the agent's token.
func (s *AgentService) ValidateToken(ctx context.Context, agentID, token string) error {
	a, err := s.store.Agents().Get(ctx, agentID)
	if err != nil {
		return err
	}
	if a.Token != token {
		return fmt.Errorf("invalid token")
	}
	return nil
}
