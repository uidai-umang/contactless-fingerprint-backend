package service

import (
	"contactless-fingerprint-backend/internal/model"
	"contactless-fingerprint-backend/internal/repository"
)

type DeviceService struct {
	deviceRepo     *repository.DeviceRepository
	cameraSpecRepo *repository.CameraSpecRepository
}

func NewDeviceService(
	deviceRepo *repository.DeviceRepository,
	cameraSpecRepo *repository.CameraSpecRepository,
) *DeviceService {
	return &DeviceService{deviceRepo: deviceRepo, cameraSpecRepo: cameraSpecRepo}
}

// Register handles device registration end to end:
//  1. Find-or-create the camera_specs row by fingerprint hash (dedup)
//  2. Insert the devices row referencing that camera_spec_id
//
// If the device (by android_id) already exists, returns the existing
// record rather than erroring — registration is idempotent by design,
// since the app may call this on every cold start.
func (s *DeviceService) Register(req model.DeviceRegistrationRequest) (*model.Device, error) {
	existing, err := s.deviceRepo.FindByAndroidID(req.AndroidID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	cameraSpecID, err := s.resolveCameraSpecID(req)
	if err != nil {
		return nil, err
	}

	device := model.Device{
		OperatorID:           req.OperatorID,
		AndroidID:            req.AndroidID,
		DeviceFingerprint:    req.DeviceFingerprint,
		DeviceModel:          req.DeviceModel,
		DeviceManufacturer:   req.DeviceManufacturer,
		OSVersion:            req.OSVersion,
		PlayIntegrityStatus:  req.PlayIntegrityStatus,
		CameraSpecID:         cameraSpecID,
		AndroidSDKVersion:    req.AndroidSDKVersion,
		AndroidSecurityPatch: req.AndroidSecurityPatch,
		SOCModel:             req.SOCModel,
		RAMTotalMB:           req.RAMTotalMB,
	}

	return s.deviceRepo.Insert(device)
}

// resolveCameraSpecID implements the find-or-create dedup logic: reuse
// an existing camera_specs row if this exact hardware fingerprint has
// been seen before, otherwise insert a new one.
func (s *DeviceService) resolveCameraSpecID(req model.DeviceRegistrationRequest) (string, error) {
	existing, err := s.cameraSpecRepo.FindByFingerprintHash(req.CameraFingerprintHash)
	if err != nil {
		return "", err
	}
	if existing != nil {
		return existing.CameraSpecID, nil
	}

	newSpec := model.CameraSpec{
		FingerprintHash:            req.CameraFingerprintHash,
		CameraID:                   req.CameraID,
		LensFacing:                 req.LensFacing,
		HardwareLevel:              req.HardwareLevel,
		SensorPhysicalSizeMM:       req.SensorPhysicalSizeMM,
		SensorActiveArraySize:      req.SensorActiveArraySize,
		PixelArraySize:             req.PixelArraySize,
		FocalLengthMM:              req.FocalLengthMM,
		Aperture:                   req.Aperture,
		MinFocusDistanceDiopters:   req.MinFocusDistanceDiopters,
		HyperfocalDistanceDiopters: req.HyperfocalDistanceDiopters,
		HasFlash:                   req.HasFlash,
		HasOIS:                     req.HasOIS,
		MaxDigitalZoom:             req.MaxDigitalZoom,
		SensorOrientation:          req.SensorOrientation,
		SupportsRaw:                req.SupportsRaw,
		AfModes:                    req.AfModes,
		AeModes:                    req.AeModes,
		AwbModes:                   req.AwbModes,
	}

	created, err := s.cameraSpecRepo.Insert(newSpec)
	if err != nil {
		// Race condition guard: another device with identical hardware
		// may have registered concurrently between our FindByFingerprintHash
		// check and this Insert. Re-fetch rather than fail the whole
		// registration over a benign race.
		if err == repository.ErrDuplicateCameraSpec {
			retry, retryErr := s.cameraSpecRepo.FindByFingerprintHash(req.CameraFingerprintHash)
			if retryErr != nil {
				return "", retryErr
			}
			if retry != nil {
				return retry.CameraSpecID, nil
			}
		}
		return "", err
	}
	return created.CameraSpecID, nil
}
