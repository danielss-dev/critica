package session

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// Session represents an AI session
type Session struct {
	ID        string
	CreatedAt time.Time
	Active    bool
	Context   map[string]interface{}
}

// Service manages AI sessions
type Service interface {
	Create() *Session
	Get(id string) (*Session, bool)
	Close(id string)
	List() []*Session
}

type service struct {
	sessions map[string]*Session
	mu       sync.RWMutex
}

// NewService creates a new session service
func NewService() Service {
	return &service{
		sessions: make(map[string]*Session),
	}
}

// Create creates a new session
func (s *service) Create() *Session {
	s.mu.Lock()
	defer s.mu.Unlock()

	session := &Session{
		ID:        uuid.New().String(),
		CreatedAt: time.Now(),
		Active:    true,
		Context:   make(map[string]interface{}),
	}

	s.sessions[session.ID] = session
	return session
}

// Get retrieves a session by ID
func (s *service) Get(id string) (*Session, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, ok := s.sessions[id]
	return session, ok
}

// Close closes a session
func (s *service) Close(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if session, ok := s.sessions[id]; ok {
		session.Active = false
	}
}

// List returns all sessions
func (s *service) List() []*Session {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sessions := make([]*Session, 0, len(s.sessions))
	for _, session := range s.sessions {
		sessions = append(sessions, session)
	}
	return sessions
}
