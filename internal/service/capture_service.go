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
	sessionRepo  *repository.SessionRepository
	residentRepo *repository.ResidentRepository
	imageStore   storage.ImageStore
	decrypter    *crypto.Decrypter
}

func NewCaptureService(
	db *sql.DB,
	captureRepo *repository.CaptureRepository,
	sessionRepo *repository.SessionRepository,
	residentRepo *repository.ResidentRepository,
	imageStore storage.ImageStore,
	decrypter *crypto.Decrypter,
) *CaptureService {
	return &CaptureService{
		db:           db,
		captureRepo:  captureRepo,
		sessionRepo:  sessionRepo,
		residentRepo: residentRepo,
		imageStore:   imageStore,
		decrypter:    decrypter,
	}
}

// Upload handles a single fingerprint capture (sequential or slap — the
// finger_type value decides which).
// Returns repository.ErrNotFound if the session does not exist,
// repository.ErrDuplicateCapture if this finger was already captured for the session,
// repository.ErrCaptureModeMismatch if the resident is already locked into the other mode.
func (s *CaptureService) Upload(req model.CaptureRequest, encryptedImageBytes []byte) (*model.CaptureResponse, error) {
	session, err := s.sessionRepo.GetByID(req.SessionID)
	if err != nil {
		return nil, err
	}

	// Decrypt before anything touches storage or the DB. imageStore.Save
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

	// capture_mode is declared by the client on every request (validated by
	// the handler already), not inferred from finger_type — LEFT_THUMB/
	// RIGHT_THUMB are legal in both modes, so finger_type alone can't tell
	// them apart.
	incomingMode := req.CaptureMode

	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // no-op once tx.Commit() below succeeds

	// Row-locks the resident so a second capture racing in at the same time
	// (e.g. two images from the same slap batch upload) can't also see
	// capture_mode as unset and both try to set/validate it.
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

	exists, err := s.captureRepo.ExistsUploadedTx(tx, req.SessionID, req.ResidentPseudonymID, req.FingerType)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, repository.ErrDuplicateCapture
	}

	storageKey := repository.GenerateCephKey(
		session.CentreID,
		req.ResidentPseudonymID,
		req.SessionID,
		req.FingerType,
	)

	// NOTE: this was already true before capture_mode existed and is unchanged
	// here — if imageStore.Save succeeds but the transaction below fails/rolls
	// back, the file is orphaned in storage with no DB row pointing to it.
	// Worth a cleanup job if that's not already handled elsewhere.
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
