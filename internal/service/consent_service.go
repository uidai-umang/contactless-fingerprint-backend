package service

import (
	"contactless-fingerprint-backend/internal/model"
	"contactless-fingerprint-backend/internal/repository"
)

type ConsentService struct {
	consentRepo  *repository.ConsentRepository
	residentRepo *repository.ResidentRepository
}

func NewConsentService(consentRepo *repository.ConsentRepository, residentRepo *repository.ResidentRepository) *ConsentService {
	return &ConsentService{consentRepo: consentRepo, residentRepo: residentRepo}
}

// Record saves the resident's consent decision and returns the resulting state.
// Returns repository.ErrNotFound if the resident does not exist.
func (s *ConsentService) Record(req model.ConsentRequest, operatorID string) (*model.ConsentResponse, error) {
	exists, err := s.residentRepo.ExistsByID(req.ResidentPseudonymID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, repository.ErrNotFound
	}

	if err := s.consentRepo.Record(req.ResidentPseudonymID, operatorID, req.LanguageShown, *req.Consented); err != nil {
		return nil, err
	}

	isConsented, err := s.consentRepo.IsConsented(req.ResidentPseudonymID)
	if err != nil {
		return nil, err
	}

	return &model.ConsentResponse{
		ResidentPseudonymID: req.ResidentPseudonymID,
		IsConsented:         isConsented,
	}, nil
}
