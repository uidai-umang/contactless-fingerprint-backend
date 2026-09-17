package model

import "time"

// Operator represents a data collection operator
type Operator struct {
	OperatorID  string     `json:"operator_id"`
	FaceAuthRef string     `json:"face_auth_ref"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	LastLoginAt *time.Time `json:"last_login_at"`
}

// Resident — no PII stored
type Resident struct {
	ResidentPseudonymID string    `json:"resident_pseudonym_id"`
	AadhaarHash         string    `json:"-"`
	AgeGroup            string    `json:"age_group"`
	Gender              string    `json:"gender"`
	SkinTone            string    `json:"skin_tone"`
	CaptureMode         string    `json:"capture_mode"` // "" until first capture sets it
	CreatedAt           time.Time `json:"created_at"`
}

// Client declares capture_mode per request; can't be inferred from finger_type
// since LEFT_THUMB/RIGHT_THUMB are valid in both SEQUENTIAL and SLAP.
const (
	CaptureModeSequential = "SEQUENTIAL"
	CaptureModeSlap       = "SLAP"
)

// Consent — no session concept, tied to resident + operator directly
type Consent struct {
	ConsentID           string    `json:"consent_id"`
	ResidentPseudonymID string    `json:"resident_pseudonym_id"`
	Consented           bool      `json:"consented"`
	LanguageShown       string    `json:"language_shown"`
	OperatorID          string    `json:"operator_id"`
	CreatedAt           time.Time `json:"created_at"`
}

// Capture — image lives in CEPH, only the key is stored here
type Capture struct {
	CaptureID           string     `json:"capture_id"`
	ResidentPseudonymID string     `json:"resident_pseudonym_id"`
	OperatorID          string     `json:"operator_id"`
	FingerType          string     `json:"finger_type"`
	Hand                string     `json:"hand"`
	Nfiq2Score          float64    `json:"nfiq2_score"`
	BlurScore           float64    `json:"blur_score"`
	BrightnessScore     float64    `json:"brightness_score"`
	GlareScore          float64    `json:"glare_score"`
	AttemptCount        int        `json:"attempt_count"`
	DegradedFlag        bool       `json:"degraded_flag"`
	CephObjectKey       string     `json:"ceph_object_key"`
	ImageChecksum       string     `json:"image_checksum"`
	WrappedDekRef       string     `json:"wrapped_dek_ref"`
	CameraModel         string     `json:"camera_model"`
	CameraResolution    string     `json:"camera_resolution"`
	DeviceModel         string     `json:"device_model"`
	UploadStatus        string     `json:"upload_status"`
	UploadAttempts      int        `json:"upload_attempts"`
	CreatedAt           time.Time  `json:"created_at"`
	UploadedAt          *time.Time `json:"uploaded_at"`
}

type AuditLog struct {
	LogID       string    `json:"log_id"`
	EventType   string    `json:"event_type"`
	OperatorID  string    `json:"operator_id"`
	DeviceID    string    `json:"device_id"`
	PayloadHash string    `json:"payload_hash"`
	IPAddress   string    `json:"ip_address"`
	CreatedAt   time.Time `json:"created_at"`
}

// ── Request / Response structs ──────────────────────────────────

type ResidentLookupRequest struct {
	AadhaarHash string `json:"aadhaar_hash" binding:"required"`
	AgeGroup    string `json:"age_group"`
	Gender      string `json:"gender"`
	SkinTone    string `json:"skin_tone"`
}

type ResidentLookupResponse struct {
	ResidentPseudonymID string   `json:"resident_pseudonym_id"`
	CaptureMode         string   `json:"capture_mode"`
	CapturedFingers     []string `json:"captured_fingers"`
	PendingUploads      []string `json:"pending_uploads"`
	TotalCaptured       int      `json:"total_captured"`
	IsComplete          bool     `json:"is_complete"`
}

type CaptureRequest struct {
	ResidentPseudonymID string  `json:"resident_pseudonym_id" binding:"required"`
	OperatorID          string  `json:"operator_id" binding:"required"`
	CaptureMode         string  `json:"capture_mode" binding:"required"` // SEQUENTIAL | SLAP
	FingerType          string  `json:"finger_type" binding:"required"`
	Hand                string  `json:"hand" binding:"required"`
	Nfiq2Score          float64 `json:"nfiq2_score"`
	BlurScore           float64 `json:"blur_score"`
	BrightnessScore     float64 `json:"brightness_score"`
	GlareScore          float64 `json:"glare_score"`
	AttemptCount        int     `json:"attempt_count"`
	DegradedFlag        bool    `json:"degraded_flag"`
	ImageChecksum       string  `json:"image_checksum"`
	CameraModel         string  `json:"camera_model"`
	CameraResolution    string  `json:"camera_resolution"`
	DeviceModel         string  `json:"device_model"`
	EncryptedSessionKey string  `json:"encrypted_session_key" binding:"required"`
	IV                  string  `json:"iv" binding:"required"`
	Hmac                string  `json:"hmac" binding:"required"`
	Thumbprint          string  `json:"thumbprint" binding:"required"`
}

type BatchCaptureRequest struct {
	Captures []CaptureRequest `json:"captures" binding:"required"`
}

type CaptureResponse struct {
	CaptureID     string `json:"capture_id"`
	FingerType    string `json:"finger_type"`
	UploadStatus  string `json:"upload_status"`
	TotalCaptured int    `json:"total_captured"`
	IsComplete    bool   `json:"is_complete"`
}

// DevResetRequest wipes all data for a resident — dev/test only
type DevResetRequest struct {
	AadhaarHash string `json:"aadhaar_hash" binding:"required"`
}
