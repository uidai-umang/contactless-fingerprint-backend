package service

import (
	"errors"

	"contactless-fingerprint-backend/internal/model"
	"contactless-fingerprint-backend/internal/repository"
)

// ErrOperatorNotAllowed means the operator is not on the allowed list, or is
// marked fraud.
var ErrOperatorNotAllowed = errors.New("operator not allowed")

type OperatorService struct {
	operatorRepo *repository.OperatorRepository
}

func NewOperatorService(operatorRepo *repository.OperatorRepository) *OperatorService {
	return &OperatorService{operatorRepo: operatorRepo}
}

// Authorize resolves operator_ref_id to an operator:
//  1. already in operators -> returned, unless marked fraud (ErrOperatorNotAllowed)
//  2. else in allowed_operators -> created in operators and returned
//  3. else ErrOperatorNotAllowed
func (s *OperatorService) Authorize(operatorRefID string) (*model.Operator, error) {
	op, err := s.operatorRepo.FindByRefID(operatorRefID)
	if err == nil {
		if op.IsFraud {
			return nil, ErrOperatorNotAllowed
		}
		return op, nil
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}

	allowed, err := s.operatorRepo.IsAllowed(operatorRefID)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, ErrOperatorNotAllowed
	}

	return s.operatorRepo.Create(operatorRefID)
}

// AddAllowed adds operator_ref_id to the allowed list. created is false if it
// was already there.
func (s *OperatorService) AddAllowed(operatorRefID string) (bool, error) {
	return s.operatorRepo.AddAllowed(operatorRefID)
}

// SetFraud marks or clears the fraud flag. Returns repository.ErrNotFound if
// the operator does not exist.
func (s *OperatorService) SetFraud(operatorRefID string, isFraud bool) error {
	return s.operatorRepo.SetFraud(operatorRefID, isFraud)
}
