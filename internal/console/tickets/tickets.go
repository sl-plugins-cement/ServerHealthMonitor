package tickets

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Status string

const (
	StatusOpen       Status = "open"
	StatusInProgress Status = "in_progress"
	StatusResolved   Status = "resolved"
	StatusClosed     Status = "closed"
)

type Ticket struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Priority    string    `json:"priority"`
	Status      Status    `json:"status"`
	Reporter    string    `json:"reporter"`
	Assignee    string    `json:"assignee,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Store struct {
	mu       sync.RWMutex
	tickets  map[string]*Ticket
	dataPath string
}

func NewStore(dataPath string) *Store {
	s := &Store{tickets: make(map[string]*Ticket), dataPath: dataPath}
	s.load()
	return s
}

func (s *Store) Create(title, description, priority, reporter string) (*Ticket, error) {
	title = strings.TrimSpace(title)
	description = strings.TrimSpace(description)
	if title == "" || description == "" || reporter == "" {
		return nil, errors.New("工单标题、描述和提交人不能为空")
	}
	if len([]rune(title)) > 120 || len([]rune(description)) > 10000 {
		return nil, errors.New("工单内容超出长度限制")
	}
	if priority != "low" && priority != "normal" && priority != "high" {
		priority = "normal"
	}
	now := time.Now().UTC()
	ticket := &Ticket{ID: "tkt-" + now.Format("20060102150405.000000000"), Title: title, Description: description, Priority: priority, Status: StatusOpen, Reporter: reporter, CreatedAt: now, UpdatedAt: now}
	s.mu.Lock()
	s.tickets[ticket.ID] = ticket
	s.save()
	s.mu.Unlock()
	copy := *ticket
	return &copy, nil
}

func (s *Store) List(requester string, manage bool) []Ticket {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Ticket, 0, len(s.tickets))
	for _, ticket := range s.tickets {
		if !manage && ticket.Reporter != requester {
			continue
		}
		result = append(result, *ticket)
	}
	return result
}

func (s *Store) Update(id string, status Status, assignee string) (*Ticket, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ticket, ok := s.tickets[id]
	if !ok {
		return nil, errors.New("工单不存在")
	}
	switch status {
	case StatusOpen, StatusInProgress, StatusResolved, StatusClosed:
		ticket.Status = status
	default:
		return nil, errors.New("无效工单状态")
	}
	ticket.Assignee = strings.TrimSpace(assignee)
	ticket.UpdatedAt = time.Now().UTC()
	s.save()
	copy := *ticket
	return &copy, nil
}

type data struct {
	Tickets map[string]*Ticket `json:"tickets"`
}

func (s *Store) load() {
	content, err := os.ReadFile(filepath.Join(s.dataPath, "tickets.json"))
	if err != nil {
		return
	}
	var saved data
	if json.Unmarshal(content, &saved) == nil && saved.Tickets != nil {
		s.tickets = saved.Tickets
	}
}

func (s *Store) save() {
	_ = os.MkdirAll(s.dataPath, 0750)
	content, err := json.MarshalIndent(data{Tickets: s.tickets}, "", "  ")
	if err == nil {
		_ = os.WriteFile(filepath.Join(s.dataPath, "tickets.json"), content, 0600)
	}
}
