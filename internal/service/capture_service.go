package service

import (
	"context"
	"fmt"

	"contactless-fingerprint-backend/internal/crypto"
	"contactless-fingerprint-backend/internal/model"
	"contactless-fingerprint-backend/internal/repository"
	"contactless-fingerprint-backend/internal/storage"
)

type CaptureService struct {
	captureRepo *repository.CaptureRepository
	sessionRepo *repository.SessionRepository
	imageStore  storage.ImageStore
	decrypter   *crypto.Decrypter
}

func NewCaptureService(
	captureRepo *repository.CaptureRepository,
	sessionRepo *repository.SessionRepository,
	imageStore storage.ImageStore,
	decrypter *crypto.Decrypter,
) *CaptureService {
	return &CaptureService{
		captureRepo: captureRepo,
		sessionRepo: sessionRepo,
		imageStore:  imageStore,
		decrypter:   decrypter,
	}
}

// Upload handles a single fingerprint capture.
// Returns repository.ErrNotFound if the session does not exist,
// repository.ErrDuplicateCapture if this finger was already captured for the session.
func (s *CaptureService) Upload(req model.CaptureRequest, encryptedImageBytes []byte) (*model.CaptureResponse, error) {
	session, err := s.sessionRepo.GetByID(req.SessionID)
	if err != nil {
		return nil, err
	}

	exists, err := s.captureRepo.ExistsUploaded(req.SessionID, req.ResidentPseudonymID, req.FingerType)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, repository.ErrDuplicateCapture
	}

	// NEW -- decrypt before anything touches storage. imageStore.Save
	// (LocalStore today, CephStore later -- same interface, zero change
	// needed there) always receives plaintext, so the storage layer stays
	// completely unaware encryption exists at all.
	plaintextImage, err := s.decrypter.Decrypt(crypto.EncryptedPayload{
		EncryptedData:       encryptedImageBytes,
		EncryptedSessionKey: req.EncryptedSessionKey,
		IV:                  req.IV,
		Hmac:                req.Hmac,
		Thumbprint:          req.Thumbprint,
	})

	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
	}

	storageKey := repository.GenerateCephKey(
		session.CentreID,
		req.ResidentPseudonymID,
		req.SessionID,
		req.FingerType,
	)

	if err := s.imageStore.Save(context.Background(), storageKey, plaintextImage); err != nil {
		return nil, fmt.Errorf("failed to save image: %w", err)
	}

	capture, err := s.captureRepo.Insert(req, storageKey)
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

var ErrDecryptionFailed = fmt.Errorf("failed to decrypt uploaded image")
