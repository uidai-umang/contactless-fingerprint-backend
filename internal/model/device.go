package model

import "time"

type CameraSpec struct {
	CameraSpecID               string    `json:"camera_spec_id"`
	FingerprintHash            string    `json:"fingerprint_hash"`
	CameraID                   string    `json:"camera_id"`
	LensFacing                 string    `json:"lens_facing"`
	HardwareLevel              string    `json:"hardware_level"`
	SensorPhysicalSizeMM       string    `json:"sensor_physical_size_mm"`
	SensorActiveArraySize      string    `json:"sensor_active_array_size"`
	PixelArraySize             string    `json:"pixel_array_size"`
	FocalLengthMM              float64   `json:"focal_length_mm"`
	Aperture                   float64   `json:"aperture"`
	MinFocusDistanceDiopters   float64   `json:"min_focus_distance_diopters"`
	HyperfocalDistanceDiopters float64   `json:"hyperfocal_distance_diopters"`
	HasFlash                   bool      `json:"has_flash"`
	HasOIS                     bool      `json:"has_ois"`
	MaxDigitalZoom             float64   `json:"max_digital_zoom"`
	SensorOrientation          int       `json:"sensor_orientation"`
	SupportsRaw                bool      `json:"supports_raw"`
	AfModes                    []int64   `json:"af_modes"`
	AeModes                    []int64   `json:"ae_modes"`
	AwbModes                   []int64   `json:"awb_modes"`
	CreatedAt                  time.Time `json:"created_at"`
}

type Device struct {
	DeviceID             string     `json:"device_id"`
	OperatorID           string     `json:"operator_id"`
	AndroidID            string     `json:"android_id"`
	DeviceFingerprint    string     `json:"device_fingerprint"`
	DeviceModel          string     `json:"device_model"`
	DeviceManufacturer   string     `json:"device_manufacturer"`
	OSVersion            string     `json:"os_version"`
	PlayIntegrityStatus  string     `json:"play_integrity_status"`
	IsFlagged            bool       `json:"is_flagged"`
	CameraSpecID         string     `json:"camera_spec_id"`
	AndroidSDKVersion    int        `json:"android_sdk_version"`
	AndroidSecurityPatch string     `json:"android_security_patch"`
	SOCModel             string     `json:"soc_model"`
	RAMTotalMB           int        `json:"ram_total_mb"`
	RegisteredAt         time.Time  `json:"registered_at"`
	LastSeenAt           *time.Time `json:"last_seen_at,omitempty"`
}

// DeviceRegistrationRequest is what the Android app sends on first
// launch / device registration — combines device-level and camera-level
// (static, hardware) fields in one call.
type DeviceRegistrationRequest struct {
	OperatorID           string `json:"operator_id"`
	AndroidID            string `json:"android_id"`
	DeviceFingerprint    string `json:"device_fingerprint"`
	DeviceModel          string `json:"device_model"`
	DeviceManufacturer   string `json:"device_manufacturer"`
	OSVersion            string `json:"os_version"`
	PlayIntegrityStatus  string `json:"play_integrity_status"`
	AndroidSDKVersion    int    `json:"android_sdk_version"`
	AndroidSecurityPatch string `json:"android_security_patch"`
	SOCModel             string `json:"soc_model"`
	RAMTotalMB           int    `json:"ram_total_mb"`

	CameraFingerprintHash      string  `json:"camera_fingerprint_hash"`
	CameraID                   string  `json:"camera_id"`
	LensFacing                 string  `json:"lens_facing"`
	HardwareLevel              string  `json:"hardware_level"`
	SensorPhysicalSizeMM       string  `json:"sensor_physical_size_mm"`
	SensorActiveArraySize      string  `json:"sensor_active_array_size"`
	PixelArraySize             string  `json:"pixel_array_size"`
	FocalLengthMM              float64 `json:"focal_length_mm"`
	Aperture                   float64 `json:"aperture"`
	MinFocusDistanceDiopters   float64 `json:"min_focus_distance_diopters"`
	HyperfocalDistanceDiopters float64 `json:"hyperfocal_distance_diopters"`
	HasFlash                   bool    `json:"has_flash"`
	HasOIS                     bool    `json:"has_ois"`
	MaxDigitalZoom             float64 `json:"max_digital_zoom"`
	SensorOrientation          int     `json:"sensor_orientation"`
	SupportsRaw                bool    `json:"supports_raw"`
	AfModes                    []int64 `json:"af_modes"`
	AeModes                    []int64 `json:"ae_modes"`
	AwbModes                   []int64 `json:"awb_modes"`
}
