package service

import (
	"time"

	"contactless-fingerprint-backend/internal/model"
	"contactless-fingerprint-backend/internal/repository"
)

type ResidentService struct {
	residentRepo *repository.ResidentRepository
	captureRepo  *repository.CaptureRepository
}

func NewResidentService(residentRepo *repository.ResidentRepository, captureRepo *repository.CaptureRepository) *ResidentService {
	return &ResidentService{
		residentRepo: residentRepo,
		captureRepo:  captureRepo,
	}
}

// FindOrCreateResident finds or creates a resident by Mitra ref id, then
// fetches their capture progress and returns a summary including which
// fingers are done and whether the resident is complete.
func (s *ResidentService) FindOrCreateResident(refID string, dob time.Time, gender, ageGroup string) (*model.ResidentLookupResponse, error) {
	resident, err := s.residentRepo.FindOrCreateByRefID(refID, dob, gender, ageGroup)
	if err != nil {
		return nil, err
	}

	// Fetch capture progress for the resident
	captures, err := s.captureRepo.GetByResidentID(resident.ResidentPseudonymID)
	if err != nil {
		return nil, err
	}

	// Build list of captured fingers and pending uploads
	capturedFingers := []string{}
	pendingUploads := []string{}

	for _, c := range captures {
		if c.UploadStatus == "UPLOADED" {
			capturedFingers = append(capturedFingers, c.FingerType)
		}

		if c.UploadStatus == "PENDING" {
			pendingUploads = append(pendingUploads, c.FingerType)
		}
	}

	// A resident's capture mode is whatever mode their captures were made in.
	// Empty when there are no captures yet.
	captureMode := ""
	if len(captures) > 0 {
		captureMode = captures[0].CaptureMode
	}

	// Required count depends on the resident's capture mode. Unset defaults to
	// the SEQUENTIAL count -- harmless, since len(capturedFingers) is 0 then.
	requiredCount := 10
	if captureMode == model.CaptureModeSlap {
		requiredCount = 4
	}
	isComplete := len(capturedFingers) >= requiredCount

	return &model.ResidentLookupResponse{
		ResidentPseudonymID: resident.ResidentPseudonymID,
		Gender:              resident.Gender,
		AgeGroup:            resident.AgeGroup,
		CaptureMode:         captureMode,
		CapturedFingers:     capturedFingers,
		PendingUploads:      pendingUploads,
		TotalCaptured:       len(capturedFingers),
		IsComplete:          isComplete,
	}, nil
}

// Reset wipes a resident for testing purposes -- dev only
func (s *ResidentService) Reset(refID string) error {
	return s.residentRepo.DeleteByRefID(refID)
}
