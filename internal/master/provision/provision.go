// Package provision handles remote agent deployment via SSH.
package provision

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/aven/ngoogle/internal/model"
	"github.com/aven/ngoogle/internal/store"
)

// Service handles SSH-based agent provisioning.
type Service struct {
	store       store.Store
	masterURL   string
	downloadURL string // GitHub release download URL template with {arch} placeholder
}

// NewService creates a new provision Service.
func NewService(st store.Store, masterURL, downloadURL string) *Service {
	if downloadURL == "" {
		downloadURL = "https://github.com/SHIINMASHIRO/New-Google-LF/releases/latest/download/agent-linux-{arch}"
	}
	return &Service{store: st, masterURL: masterURL, downloadURL: downloadURL}
}

// JobRequest is the input for a provisioning job.
type JobRequest struct {
	HostIP        string         `json:"host_ip"`
	SSHPort       int            `json:"ssh_port"`
	SSHUser       string         `json:"ssh_user"`
	AuthType      model.AuthType `json:"auth_type"`
	CredentialRef string         `json:"credential_ref"`
}

// CredentialRequest is the input for creating a credential.
type CredentialRequest struct {
	Name    string         `json:"name"`
	Type    model.AuthType `json:"type"`
	Payload string         `json:"payload"` // private key PEM or password
}

// Start creates a provisioning job and runs it asynchronously.
func (s *Service) Start(ctx context.Context, req *JobRequest) (*model.ProvisionJob, error) {
	if req.HostIP == "" || req.SSHUser == "" || req.CredentialRef == "" {
		return nil, fmt.Errorf("host_ip, ssh_user and credential_ref are required")
	}
	if req.SSHPort <= 0 {
		req.SSHPort = 22
	}
	// Check for duplicate IP in existing agents
	agents, err := s.store.Agents().List(ctx)
	if err != nil {
		return nil, fmt.Errorf("check duplicate IP: %w", err)
	}
	for _, a := range agents {
		if a.IP == req.HostIP {
			return nil, fmt.Errorf("agent with IP %s already exists (id: %s)", req.HostIP, a.ID)
		}
	}
	// Check for in-progress provision jobs with same IP
	jobs, err := s.store.ProvisionJobs().List(ctx)
	if err != nil {
		return nil, fmt.Errorf("check duplicate provision: %w", err)
	}
	for _, j := range jobs {
		if j.HostIP == req.HostIP && (j.Status == model.ProvisionStatusPending || j.Status == model.ProvisionStatusRunning) {
			return nil, fmt.Errorf("a provisioning job for IP %s is already in progress", req.HostIP)
		}
	}
	now := time.Now()
	job := &model.ProvisionJob{
		ID:            newID(),
		HostIP:        req.HostIP,
		SSHPort:       req.SSHPort,
		SSHUser:       req.SSHUser,
		AuthType:      req.AuthType,
		CredentialRef: req.CredentialRef,
		Status:        model.ProvisionStatusPending,
		CurrentStep:   "created",
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := s.store.ProvisionJobs().Create(ctx, job); err != nil {
		return nil, err
	}
	// Run async
	go s.run(job.ID, req)
	return job, nil
}

// run executes the full provisioning workflow.
func (s *Service) run(jobID string, req *JobRequest) {
	ctx := context.Background()
	logLine := func(msg string) {
		slog.Info("provision", "job", jobID, "msg", msg)
		_ = s.store.ProvisionJobs().AppendLog(ctx, jobID, fmt.Sprintf("[%s] %s", time.Now().Format(time.RFC3339), msg))
	}
	fail := func(step, reason string) {
		logLine(fmt.Sprintf("FAILED at %s: %s", step, reason))
		_ = s.store.ProvisionJobs().SetFailed(ctx, jobID, step, reason)
	}

	_ = s.store.ProvisionJobs().UpdateStatus(ctx, jobID, model.ProvisionStatusRunning, "ssh_check")

	// Step 1: Load credential
	logLine("Loading credential...")
	cred, err := s.store.Credentials().Get(ctx, req.CredentialRef)
	if err != nil {
		fail("ssh_check", "credential not found: "+err.Error())
		return
	}

	// Step 2: Build SSH config
	sshCfg, err := buildSSHConfig(req.SSHUser, cred)
	if err != nil {
		fail("ssh_check", "SSH config error: "+err.Error())
		return
	}

	// Step 3: SSH connectivity check
	logLine(fmt.Sprintf("Connecting to %s:%d...", req.HostIP, req.SSHPort))
	addr := fmt.Sprintf("%s:%d", req.HostIP, req.SSHPort)
	client, err := ssh.Dial("tcp", addr, sshCfg)
	if err != nil {
		fail("ssh_check", "SSH connect failed: "+err.Error())
		return
	}
	defer client.Close()
	logLine("SSH connectivity OK")

	_ = s.store.ProvisionJobs().UpdateStatus(ctx, jobID, model.ProvisionStatusRunning, "download_binary")

	// Step 4: Download agent binary from GitHub Releases
	logLine("Detecting target architecture...")
	archOut, err := runSSH(client, "uname -m")
	if err != nil {
		fail("download_binary", "detect arch: "+err.Error())
		return
	}
	goArch := mapArch(strings.TrimSpace(archOut))
	downloadURL := strings.ReplaceAll(s.downloadURL, "{arch}", goArch)
	logLine(fmt.Sprintf("Downloading agent binary (%s) from %s", goArch, downloadURL))

	dlCmd := fmt.Sprintf("wget -q -O /tmp/ngoogle-agent '%s' || curl -fsSL -o /tmp/ngoogle-agent '%s'", downloadURL, downloadURL)
	if out, err := runSSH(client, dlCmd); err != nil {
		fail("download_binary", fmt.Sprintf("download failed: %s; output: %s", err, out))
		return
	}
	logLine("Agent binary downloaded")

	_ = s.store.ProvisionJobs().UpdateStatus(ctx, jobID, model.ProvisionStatusRunning, "install_service")

	// Step 5: Install systemd service
	logLine("Installing systemd service...")
	unitContent := fmt.Sprintf(systemdTemplate, req.HostIP, s.masterURL)
	installCmds := []string{
		"sudo mv /tmp/ngoogle-agent /usr/local/bin/ngoogle-agent && sudo chmod +x /usr/local/bin/ngoogle-agent",
		fmt.Sprintf("sudo tee /etc/systemd/system/ngoogle-agent.service > /dev/null << 'UNIT_EOF'\n%sUNIT_EOF", unitContent),
		"sudo systemctl daemon-reload && sudo systemctl enable ngoogle-agent && sudo systemctl restart ngoogle-agent",
	}
	for _, cmd := range installCmds {
		logLine("  $ " + cmd[:min(80, len(cmd))])
		if out, err := runSSH(client, cmd); err != nil {
			fail("install_service", fmt.Sprintf("cmd error: %s; output: %s", err, out))
			return
		}
	}
	logLine("Service installed and started")

	_ = s.store.ProvisionJobs().UpdateStatus(ctx, jobID, model.ProvisionStatusRunning, "health_check")

	// Step 6: Wait for agent to appear online (max 60s)
	logLine("Waiting for agent to come online (max 60s)...")
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		agents, err := s.store.Agents().List(ctx)
		if err == nil {
			for _, a := range agents {
				if a.IP == req.HostIP && a.Status == model.AgentStatusOnline {
					logLine(fmt.Sprintf("Agent %s is online!", a.ID))
					_ = s.store.ProvisionJobs().SetAgentID(ctx, jobID, a.ID)
					_ = s.store.ProvisionJobs().UpdateStatus(ctx, jobID, model.ProvisionStatusSuccess, "done")
					return
				}
			}
		}
		time.Sleep(5 * time.Second)
	}
	fail("health_check", "agent did not come online within 60s")
}

// Retry resets a failed provision job and re-runs it.
func (s *Service) Retry(ctx context.Context, jobID string) (*model.ProvisionJob, error) {
	job, err := s.store.ProvisionJobs().Get(ctx, jobID)
	if err != nil {
		return nil, err
	}
	if job.Status != model.ProvisionStatusFailed {
		return nil, fmt.Errorf("only failed jobs can be retried (current: %s)", job.Status)
	}
	if err := s.store.ProvisionJobs().ResetForRetry(ctx, jobID); err != nil {
		return nil, err
	}
	req := &JobRequest{
		HostIP:        job.HostIP,
		SSHPort:       job.SSHPort,
		SSHUser:       job.SSHUser,
		AuthType:      job.AuthType,
		CredentialRef: job.CredentialRef,
	}
	go s.run(jobID, req)
	job.Status = model.ProvisionStatusPending
	job.CurrentStep = "created"
	job.Log = ""
	job.FailedStep = ""
	return job, nil
}

// DeleteCredential deletes a credential by ID.
func (s *Service) DeleteCredential(ctx context.Context, id string) error {
	return s.store.Credentials().Delete(ctx, id)
}

// ListJobs returns all provisioning jobs.
func (s *Service) ListJobs(ctx context.Context) ([]*model.ProvisionJob, error) {
	return s.store.ProvisionJobs().List(ctx)
}

// GetJob returns a single provisioning job.
func (s *Service) GetJob(ctx context.Context, id string) (*model.ProvisionJob, error) {
	return s.store.ProvisionJobs().Get(ctx, id)
}

// CreateCredential stores a credential.
func (s *Service) CreateCredential(ctx context.Context, req *CredentialRequest) (*model.Credential, error) {
	c := &model.Credential{
		ID:        newID(),
		Name:      req.Name,
		Type:      req.Type,
		Payload:   req.Payload,
		CreatedAt: time.Now(),
	}
	return c, s.store.Credentials().Create(ctx, c)
}

// ListCredentials returns all credentials.
func (s *Service) ListCredentials(ctx context.Context) ([]*model.Credential, error) {
	return s.store.Credentials().List(ctx)
}

// ─── SSH helpers ──────────────────────────────────────────────────────────────

func buildSSHConfig(user string, cred *model.Credential) (*ssh.ClientConfig, error) {
	cfg := &ssh.ClientConfig{
		User:            user,
		Timeout:         15 * time.Second,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), //nolint:gosec
	}
	switch cred.Type {
	case model.AuthTypeKey:
		signer, err := ssh.ParsePrivateKey([]byte(cred.Payload))
		if err != nil {
			return nil, fmt.Errorf("parse private key: %w", err)
		}
		cfg.Auth = []ssh.AuthMethod{ssh.PublicKeys(signer)}
	case model.AuthTypePassword:
		cfg.Auth = []ssh.AuthMethod{ssh.Password(cred.Payload)}
	default:
		return nil, fmt.Errorf("unknown auth type: %s", cred.Type)
	}
	return cfg, nil
}

func runSSH(client *ssh.Client, cmd string) (string, error) {
	sess, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer sess.Close()
	var buf bytes.Buffer
	sess.Stdout = &buf
	sess.Stderr = &buf
	return buf.String(), sess.Run(cmd)
}

// mapArch converts uname -m output to Go GOARCH names.
func mapArch(uname string) string {
	switch uname {
	case "x86_64":
		return "amd64"
	case "aarch64", "arm64":
		return "arm64"
	default:
		return "amd64"
	}
}

// ─── Systemd template ─────────────────────────────────────────────────────────

const systemdTemplate = `[Unit]
Description=ngoogle Agent
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/ngoogle-agent
Environment=AGENT_HOST_IP=%s
Environment=MASTER_URL=%s
Restart=on-failure
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
`

// ─── Misc helpers ─────────────────────────────────────────────────────────────

func newID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
