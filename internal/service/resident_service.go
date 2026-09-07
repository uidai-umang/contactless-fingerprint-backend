package service

import (
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

// LookupOrCreate finds or creates a resident by aadhaar_hash,
// then fetches their capture progress and returns a summary
// including which fingers are done and whether the session is complete.
func (s *ResidentService) FindOrCreateResident(req model.ResidentLookupRequest) (*model.ResidentLookupResponse, error) {
	resident, err := s.residentRepo.FindOrCreateByAadhaarHash(req)
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

	// Required count depends on which mode this resident is locked into.
	// Unset (no captures yet) defaults to the SEQUENTIAL count — harmless,
	// since len(capturedFingers) is 0 either way at that point.
	requiredCount := 10
	if resident.CaptureMode == model.CaptureModeSlap {
		requiredCount = 4
	}
	isComplete := len(capturedFingers) >= requiredCount

	return &model.ResidentLookupResponse{
		ResidentPseudonymID: resident.ResidentPseudonymID,
		CaptureMode:         resident.CaptureMode,
		CapturedFingers:     capturedFingers,
		PendingUploads:      pendingUploads,
		TotalCaptured:       len(capturedFingers),
		IsComplete:          isComplete,
	}, nil
}

// Reset wipes all resident data for testing purposes — dev only
func (s *ResidentService) Reset(aadhaarHash string) error {
	return s.residentRepo.DeleteByAadhaarHash(aadhaarHash)
}
