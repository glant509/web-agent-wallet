package session

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
	"web3-service-agent/internal/llm"
)

type Snapshot struct {
	ID        string        `json:"id"`
	Messages  []llm.Message `json:"messages"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}

type Manager struct {
	mu       sync.RWMutex
	sessions map[string]*Snapshot
}

func NewManager() *Manager {
	return &Manager{
		sessions: make(map[string]*Snapshot),
	}
}

func (m *Manager) Create(id string) Snapshot {
	m.mu.Lock()
	defer m.mu.Unlock()

	if id == "" {
		id = newID()
	}

	now := time.Now().UTC()
	snapshot := &Snapshot{
		ID:        id,
		Messages:  []llm.Message{},
		CreatedAt: now,
		UpdatedAt: now,
	}
	m.sessions[id] = snapshot
	return cloneSnapshot(snapshot)
}

func (m *Manager) Ensure(id string) Snapshot {
	m.mu.Lock()
	defer m.mu.Unlock()

	if id == "" {
		id = newID()
	}

	if existing, ok := m.sessions[id]; ok {
		return cloneSnapshot(existing)
	}

	now := time.Now().UTC()
	snapshot := &Snapshot{
		ID:        id,
		Messages:  []llm.Message{},
		CreatedAt: now,
		UpdatedAt: now,
	}
	m.sessions[id] = snapshot
	return cloneSnapshot(snapshot)
}

func (m *Manager) Append(sessionID string, messages ...llm.Message) Snapshot {
	m.mu.Lock()
	defer m.mu.Unlock()

	current, ok := m.sessions[sessionID]
	if !ok {
		now := time.Now().UTC()
		current = &Snapshot{
			ID:        sessionID,
			CreatedAt: now,
			UpdatedAt: now,
		}
		m.sessions[sessionID] = current
	}

	current.Messages = append(current.Messages, messages...)
	current.UpdatedAt = time.Now().UTC()
	return cloneSnapshot(current)
}

func (m *Manager) Get(sessionID string) (Snapshot, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	snapshot, ok := m.sessions[sessionID]
	if !ok {
		return Snapshot{}, false
	}
	return cloneSnapshot(snapshot), true
}

func (m *Manager) GetOrEmpty(sessionID string) Snapshot {
	snapshot, ok := m.Get(sessionID)
	if !ok {
		return Snapshot{ID: sessionID, Messages: []llm.Message{}}
	}
	return snapshot
}

func cloneSnapshot(snapshot *Snapshot) Snapshot {
	messages := make([]llm.Message, len(snapshot.Messages))
	copy(messages, snapshot.Messages)
	return Snapshot{
		ID:        snapshot.ID,
		Messages:  messages,
		CreatedAt: snapshot.CreatedAt,
		UpdatedAt: snapshot.UpdatedAt,
	}
}

func newID() string {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return time.Now().UTC().Format("20060102150405")
	}
	return hex.EncodeToString(buf[:])
}
