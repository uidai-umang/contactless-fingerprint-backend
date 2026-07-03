package service

import (
	"contactless-fingerprint-backend/internal/model"
	"contactless-fingerprint-backend/internal/repository"
)

type CaptureService struct {
	captureRepo *repository.CaptureRepository
	sessionRepo *repository.SessionRepository
}

func NewCaptureService(
	captureRepo *repository.CaptureRepository,
	sessionRepo *repository.SessionRepository,
) *CaptureService {
	return &CaptureService{
		captureRepo: captureRepo,
		sessionRepo: sessionRepo,
	}
}

// Upload handles a single fingerprint capture.
// imageBytes contains the raw image received from multipart upload.
func (s *CaptureService) Upload(req model.CaptureRequest, imageBytes []byte) (*model.CaptureResponse, error) {
	session, err := s.sessionRepo.GetByID(req.SessionID)
	if err != nil {
		return nil, err
	}

	// Generate CEPH object key for image storage path
	cephKey := repository.GenerateCephKey(
		session.CentreID,
		req.ResidentPseudonymID,
		req.SessionID,
		req.FingerType,
	)

	// TODO: Upload imageBytes to CEPH here
	// For now just store the key reference in DB
	_ = imageBytes

	capture, err := s.captureRepo.Insert(req, cephKey)
	if err != nil {
		return nil, err
	}

	allCaptures, err := s.captureRepo.GetByResidentID(req.ResidentPseudonymID)
	if err != nil {
		return nil, err
	}

	uploadedCount := 0
	for _, c := range allCaptures {
		if c.UploadStatus == "UPLOADED" {
			uploadedCount++
		}
	}

	return &model.CaptureResponse{
		CaptureID:     capture.CaptureID,
		FingerType:    capture.FingerType,
		UploadStatus:  capture.UploadStatus,
		TotalCaptured: uploadedCount,
		IsComplete:    uploadedCount >= 10,
	}, nil
}
