package permission

import (
	"errors"
	"fmt"
	"sync"
)

var ErrPermissionDenied = errors.New("permission denied")

// PermissionRequest represents a request for permission
type PermissionRequest struct {
	ID          string `json:"id"`
	SessionID   string `json:"session_id"`
	ToolName    string `json:"tool_name"`
	Action      string `json:"action"`
	Description string `json:"description"`
	Params      any    `json:"params"`
	Path        string `json:"path"`
}

// Service manages permissions for AI operations
type Service interface {
	Request(req PermissionRequest) bool
	Grant(permission PermissionRequest)
	Deny(permission PermissionRequest)
	AutoApproveSession(sessionID string)
	SetCIMode(enabled bool)
}

type service struct {
	sessionPermissions  []PermissionRequest
	pendingRequests     sync.Map
	autoApproveSessions []string
	ciMode              bool
	mu                  sync.RWMutex
}

// NewService creates a new permission service
func NewService() Service {
	return &service{
		sessionPermissions: make([]PermissionRequest, 0),
	}
}

// Request checks if a permission request should be granted
func (s *service) Request(req PermissionRequest) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Check if session is auto-approved
	for _, sessionID := range s.autoApproveSessions {
		if sessionID == req.SessionID {
			return true
		}
	}

	// Check existing permissions
	for _, p := range s.sessionPermissions {
		if p.ToolName == req.ToolName && p.Action == req.Action &&
			p.SessionID == req.SessionID && p.Path == req.Path {
			return true
		}
	}

	// Auto-approve read-only operations
	if req.Action == "read" || req.Action == "analyze" {
		return true
	}

	// In CI mode, deny write operations unless explicitly granted
	if s.ciMode && (req.Action == "write" || req.Action == "edit") {
		return false
	}

	// For interactive mode, we would prompt the user here
	// For now, auto-approve for development
	fmt.Printf("Permission requested: %s - %s\n", req.ToolName, req.Description)

	s.mu.RUnlock()
	s.mu.Lock()
	s.sessionPermissions = append(s.sessionPermissions, req)
	s.mu.Unlock()
	s.mu.RLock()

	return true
}

// Grant explicitly grants a permission
func (s *service) Grant(permission PermissionRequest) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessionPermissions = append(s.sessionPermissions, permission)
}

// Deny denies a permission request
func (s *service) Deny(permission PermissionRequest) {
	// Currently a no-op, but could be extended to track denials
}

// AutoApproveSession adds a session to the auto-approve list
func (s *service) AutoApproveSession(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.autoApproveSessions = append(s.autoApproveSessions, sessionID)
}

// SetCIMode enables or disables CI mode
func (s *service) SetCIMode(enabled bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ciMode = enabled
}
