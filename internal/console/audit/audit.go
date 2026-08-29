package audit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Action type constants
const (
	ActionLogin          = "login"
	ActionLoginFailed    = "login_failed"
	ActionLogout         = "logout"
	ActionUserCreate     = "user_create"
	ActionUserDelete     = "user_delete"
	ActionUserPassword   = "user_password_change"
	ActionUserKeyRotate  = "user_key_rotate"
	ActionUserUnlock     = "user_unlock"
	ActionUserPermission = "user_permission_change"
	ActionServerAdd      = "server_add"
	ActionServerUpdate   = "server_update"
	ActionServerDelete   = "server_delete"
	ActionDeploy         = "deploy"
	ActionConfigChange   = "config_change"
	ActionDiagnostics    = "diagnostics"
	ActionRemediation    = "remediation"
	ActionTicketCreate   = "ticket_create"
	ActionTicketUpdate   = "ticket_update"
	ActionViewDashboard  = "view_dashboard"
)

// Entry represents one audit log entry.
type Entry struct {
	ID      int64  `json:"id"`
	Time    string `json:"time"`
	User    string `json:"user"`
	Action  string `json:"action"`
	Target  string `json:"target,omitempty"`
	Detail  string `json:"detail,omitempty"`
	IP      string `json:"ip,omitempty"`
	Success bool   `json:"success"`
}

// Logger records audit events.
type Logger struct {
	mu       sync.Mutex
	entries  []Entry
	dataPath string
	nextID   int64
}

// NewLogger creates an audit logger.
func NewLogger(dataPath string) *Logger {
	l := &Logger{
		dataPath: dataPath,
	}
	l.load()
	return l
}

// Log records an audit event.
func (l *Logger) Log(user, action, target, detail, ip string, success bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.nextID++
	entry := Entry{
		ID:      l.nextID,
		Time:    time.Now().Format("2006-01-02 15:04:05"),
		User:    user,
		Action:  action,
		Target:  target,
		Detail:  detail,
		IP:      ip,
		Success: success,
	}
	l.entries = append(l.entries, entry)

	// Keep only last 1000 entries
	if len(l.entries) > 1000 {
		l.entries = l.entries[len(l.entries)-1000:]
	}

	l.save()
}

// List returns audit entries, most recent first.
func (l *Logger) List(limit int, userFilter, actionFilter string) []Entry {
	l.mu.Lock()
	defer l.mu.Unlock()

	if limit <= 0 || limit > len(l.entries) {
		limit = len(l.entries)
	}

	result := make([]Entry, 0, limit)
	count := 0

	for i := len(l.entries) - 1; i >= 0 && count < limit; i-- {
		e := l.entries[i]
		if userFilter != "" && e.User != userFilter {
			continue
		}
		if actionFilter != "" && e.Action != actionFilter {
			continue
		}
		result = append(result, e)
		count++
	}

	return result
}

// --- Persistence ---

func (l *Logger) load() {
	dataFile := filepath.Join(l.dataPath, "audit.json")
	f, err := os.ReadFile(dataFile)
	if err != nil {
		return
	}
	var entries []Entry
	if err := json.Unmarshal(f, &entries); err != nil {
		return
	}
	l.entries = entries
	if len(entries) > 0 {
		l.nextID = entries[len(entries)-1].ID
	}
}

func (l *Logger) save() {
	dataFile := filepath.Join(l.dataPath, "audit.json")
	os.MkdirAll(l.dataPath, 0750)

	f, err := json.MarshalIndent(l.entries, "", "  ")
	if err != nil {
		return
	}
	os.WriteFile(dataFile, f, 0600)
}
