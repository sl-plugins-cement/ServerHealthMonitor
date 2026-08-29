package servers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

// Server represents a monitored server.
type Server struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Host      string    `json:"host"`
	Port      int       `json:"port"`       // SSH port
	User      string    `json:"user"`       // SSH user
	KeyFile   string    `json:"key_file"`   // Path to SSH private key
	AgentPort int       `json:"agent_port"` // Monitor agent HTTP port
	Status    string    `json:"status"`     // online / offline / unknown
	LastCheck time.Time `json:"last_check"`
	CreatedAt time.Time `json:"created_at"`
}

// DeploymentLog records a deployment attempt.
type DeploymentLog struct {
	ID        string    `json:"id"`
	ServerID  string    `json:"server_id"`
	User      string    `json:"user"`
	Status    string    `json:"status"` // success / failed / running
	Output    string    `json:"output"`
	StartedAt time.Time `json:"started_at"`
	EndedAt   time.Time `json:"ended_at"`
}

// Manager handles server inventory and deployments.
type Manager struct {
	mu       sync.RWMutex
	servers  map[string]*Server
	deploys  []*DeploymentLog
	dataPath string
}

// NewManager creates a server manager.
func NewManager(dataPath string) *Manager {
	m := &Manager{
		servers:  make(map[string]*Server),
		dataPath: dataPath,
	}
	m.load()
	return m
}

// --- CRUD ---

// AddServer adds a new server.
func (m *Manager) AddServer(s Server) (*Server, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if s.Name == "" || s.Host == "" {
		return nil, errors.New("服务器名称和地址不能为空")
	}
	if s.Port == 0 {
		s.Port = 22
	}
	if s.AgentPort == 0 {
		s.AgentPort = 8080
	}

	s.ID = generateID()
	s.Status = "unknown"
	s.CreatedAt = time.Now()

	m.servers[s.ID] = &s
	m.save()
	return &s, nil
}

// UpdateServer updates server info.
func (m *Manager) UpdateServer(id string, updates Server) (*Server, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	s, exists := m.servers[id]
	if !exists {
		return nil, errors.New("服务器不存在")
	}

	if updates.Name != "" {
		s.Name = updates.Name
	}
	if updates.Host != "" {
		s.Host = updates.Host
	}
	if updates.Port > 0 {
		s.Port = updates.Port
	}
	if updates.User != "" {
		s.User = updates.User
	}
	if updates.KeyFile != "" {
		s.KeyFile = updates.KeyFile
	}
	if updates.AgentPort > 0 {
		s.AgentPort = updates.AgentPort
	}

	m.save()
	copy := *s
	return &copy, nil
}

// DeleteServer removes a server.
func (m *Manager) DeleteServer(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.servers[id]; !exists {
		return errors.New("服务器不存在")
	}
	delete(m.servers, id)
	m.save()
	return nil
}

// GetServer returns a server by ID.
func (m *Manager) GetServer(id string) (*Server, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.servers[id]
	if !ok {
		return nil, false
	}
	copy := *s
	return &copy, true
}

// ListServers returns all servers.
func (m *Manager) ListServers() []Server {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]Server, 0, len(m.servers))
	for _, s := range m.servers {
		result = append(result, *s)
	}
	return result
}

// --- Deployment ---

// DeployResult is the outcome of a deployment.
type DeployResult struct {
	Success bool
	Output  string
	LogID   string
}

// DeployToServer runs the deployment script against a server.
// It executes deploy/deploy.py via Python and captures real output.
// Requires Python 3 with paramiko and scp installed on the console host.
func (m *Manager) DeployToServer(serverID, username, binaryPath string) (*DeployResult, error) {
	srv, ok := m.GetServer(serverID)
	if !ok {
		return nil, errors.New("服务器不存在")
	}

	m.mu.Lock()
	log := &DeploymentLog{
		ID:        generateID(),
		ServerID:  serverID,
		User:      username,
		Status:    "running",
		StartedAt: time.Now(),
	}
	m.deploys = append(m.deploys, log)
	m.save()
	m.mu.Unlock()

	result := &DeployResult{LogID: log.ID}

	// Locate deploy script relative to working directory.
	deployScript := filepath.Join("deploy", "deploy.py")
	if _, err := os.Stat(deployScript); os.IsNotExist(err) {
		// Try absolute path based on executable location.
		if exe, err := os.Executable(); err == nil {
			alt := filepath.Join(filepath.Dir(exe), "deploy", "deploy.py")
			if _, err := os.Stat(alt); err == nil {
				deployScript = alt
			}
		}
	}
	if _, err := os.Stat(deployScript); os.IsNotExist(err) {
		result.Success = false
		result.Output = "部署脚本不存在: " + deployScript + "\n请确保 deploy/deploy.py 存在于工作目录。"
		m.updateDeployLog(log.ID, result)
		return result, errors.New(result.Output)
	}

	// Default binary path.
	if binaryPath == "" {
		binaryPath = filepath.Join("build", "server-health-monitor-agent")
	}
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		result.Success = false
		result.Output = "二进制文件不存在: " + binaryPath + "\n请先编译: go build -o build/server-health-monitor-agent ./cmd/agent"
		m.updateDeployLog(log.ID, result)
		return result, errors.New(result.Output)
	}

	// Build command arguments.
	args := []string{
		deployScript,
		"--host", srv.Host,
		"--port", fmt.Sprintf("%d", srv.Port),
		"--user", srv.User,
		"--binary", binaryPath,
	}
	if srv.KeyFile != "" {
		args = append(args, "--key", srv.KeyFile)
	}

	output := fmt.Sprintf("开始部署到 %s (%s)...\n", srv.Name, srv.Host)
	output += fmt.Sprintf("SSH 用户: %s, 端口: %d\n", srv.User, srv.Port)
	output += fmt.Sprintf("二进制文件: %s\n\n", binaryPath)
	output += "正在执行部署脚本...\n"

	// Execute deploy.py with a 5-minute timeout.
	cmd := exec.Command("python", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run with timeout.
	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	select {
	case err := <-done:
		output += stdout.String()
		if stderr.Len() > 0 {
			output += "\n--- stderr ---\n" + stderr.String()
		}
		if err != nil {
			result.Success = false
			output += fmt.Sprintf("\n部署失败: %v", err)
		} else {
			result.Success = true
			output += "\n部署完成！"
		}
	case <-time.After(5 * time.Minute):
		_ = cmd.Process.Kill()
		result.Success = false
		output += "\n部署超时（5分钟），已终止。"
	}

	result.Output = output
	m.updateDeployLog(log.ID, result)
	return result, nil
}

func (m *Manager) updateDeployLog(logID string, result *DeployResult) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, d := range m.deploys {
		if d.ID == logID {
			if result.Success {
				d.Status = "success"
			} else {
				d.Status = "failed"
			}
			d.Output = result.Output
			d.EndedAt = time.Now()
			break
		}
	}
	m.save()
}

// ListDeployments returns recent deployment logs.
func (m *Manager) ListDeployments(limit int) []DeploymentLog {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if limit <= 0 || limit > len(m.deploys) {
		limit = len(m.deploys)
	}

	// Return most recent first
	result := make([]DeploymentLog, limit)
	for i := 0; i < limit; i++ {
		result[i] = *m.deploys[len(m.deploys)-1-i]
	}
	return result
}

// --- Persistence ---

type persistenceData struct {
	Servers map[string]*Server `json:"servers"`
	Deploys []*DeploymentLog   `json:"deploys"`
}

func (m *Manager) load() {
	dataFile := filepath.Join(m.dataPath, "servers.json")
	f, err := os.ReadFile(dataFile)
	if err != nil {
		return
	}
	var data persistenceData
	if err := json.Unmarshal(f, &data); err != nil {
		return
	}
	if data.Servers != nil {
		m.servers = data.Servers
	}
	if data.Deploys != nil {
		m.deploys = data.Deploys
	}
}

func (m *Manager) save() {
	dataFile := filepath.Join(m.dataPath, "servers.json")
	os.MkdirAll(m.dataPath, 0750)

	data := persistenceData{
		Servers: m.servers,
		Deploys: m.deploys,
	}

	f, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return
	}
	os.WriteFile(dataFile, f, 0600)
}

func generateID() string {
	return fmt.Sprintf("srv-%d", time.Now().UnixNano())
}
