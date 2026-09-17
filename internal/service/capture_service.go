package service

import (
	"context"
	"database/sql"
	"fmt"

	"contactless-fingerprint-backend/internal/crypto"
	"contactless-fingerprint-backend/internal/model"
	"contactless-fingerprint-backend/internal/repository"
	"contactless-fingerprint-backend/internal/storage"
)

type CaptureService struct {
	db           *sql.DB
	captureRepo  *repository.CaptureRepository
	residentRepo *repository.ResidentRepository
	imageStore   storage.ImageStore
	decrypter    *crypto.Decrypter
}

func NewCaptureService(
	db *sql.DB,
	captureRepo *repository.CaptureRepository,
	residentRepo *repository.ResidentRepository,
	imageStore storage.ImageStore,
	decrypter *crypto.Decrypter,
) *CaptureService {
	return &CaptureService{
		db:           db,
		captureRepo:  captureRepo,
		residentRepo: residentRepo,
		imageStore:   imageStore,
		decrypter:    decrypter,
	}
}

// Upload handles one capture. No session concept -- everything keys off
// resident_pseudonym_id + operator_id.
func (s *CaptureService) Upload(req model.CaptureRequest, encryptedImageBytes []byte) (*model.CaptureResponse, error) {
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

	incomingMode := req.CaptureMode

	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Row-lock the resident so two racing captures can't both set capture_mode
	existingMode, err := s.residentRepo.LockCaptureModeTx(tx, req.ResidentPseudonymID)
	if err != nil {
		return nil, err
	}
	if existingMode == nil {
		if err := s.residentRepo.SetCaptureModeTx(tx, req.ResidentPseudonymID, incomingMode); err != nil {
			return nil, err
		}
	} else if *existingMode != incomingMode {
		return nil, repository.ErrCaptureModeMismatch
	}

	exists, err := s.captureRepo.ExistsUploadedTx(tx, req.ResidentPseudonymID, req.FingerType)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, repository.ErrDuplicateCapture
	}

	storageKey := repository.GenerateCephKey(
		req.ResidentPseudonymID,
		req.FingerType,
	)

	if err := s.imageStore.Save(context.Background(), storageKey, plaintextImage); err != nil {
		return nil, fmt.Errorf("failed to save image: %w", err)
	}

	capture, err := s.captureRepo.InsertTx(tx, req, storageKey)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
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

	requiredCount := 10 // SEQUENTIAL
	if incomingMode == model.CaptureModeSlap {
		requiredCount = 4 // LEFT_SLAP, RIGHT_SLAP, LEFT_THUMB, RIGHT_THUMB
	}

	return &model.CaptureResponse{
		CaptureID:     capture.CaptureID,
		FingerType:    capture.FingerType,
		UploadStatus:  capture.UploadStatus,
		TotalCaptured: uploadedCount,
		IsComplete:    uploadedCount >= requiredCount,
	}, nil
}

var ErrDecryptionFailed = fmt.Errorf("failed to decrypt uploaded image")
