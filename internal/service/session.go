package service

import (
	"contactless-fingerprint-backend/internal/model"
	"contactless-fingerprint-backend/internal/repository"
)

type SessionService struct {
	sessionRepo *repository.SessionRepository
}

func NewSessionService(sessionRepo *repository.SessionRepository) *SessionService {
	return &SessionService{sessionRepo: sessionRepo}
}

// Create starts a new capture session for a resident
func (s *SessionService) Create(req model.CreateSessionRequest) (*model.Session, error) {
	return s.sessionRepo.Create(req)
}

// Close marks a session as completed
func (s *SessionService) Close(sessionID string, closeReason string) error {
	return s.sessionRepo.Close(sessionID, closeReason)
}

// GetByID fetches a session by ID
func (s *SessionService) GetByID(sessionID string) (*model.Session, error) {
	return s.sessionRepo.GetByID(sessionID)
}
