package model

import "time"

// Centre represents an ASK (Aadhaar Seva Kendra) collection centre
type Centre struct {
	CentreID  string    `json:"centre_id"`
	Name      string    `json:"name"`
	City      string    `json:"city"`
	State     string    `json:"state"`
	Region    string    `json:"region"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

// Operator represents a data collection operator linked to a centre
type Operator struct {
	OperatorID  string     `json:"operator_id"`
	CentreID    string     `json:"centre_id"`
	FaceAuthRef string     `json:"face_auth_ref"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	LastLoginAt *time.Time `json:"last_login_at"`
}

// Device represents an AMAPI managed device registered to an operator
type Device struct {
	DeviceID            string     `json:"device_id"`
	OperatorID          string     `json:"operator_id"`
	AndroidID           string     `json:"android_id"`
	DeviceFingerprint   string     `json:"device_fingerprint"`
	DeviceModel         string     `json:"device_model"`
	DeviceManufacturer  string     `json:"device_manufacturer"`
	OsVersion           string     `json:"os_version"`
	PlayIntegrityStatus string     `json:"play_integrity_status"`
	IsFlagged           bool       `json:"is_flagged"`
	RegisteredAt        time.Time  `json:"registered_at"`
	LastSeenAt          *time.Time `json:"last_seen_at"`
}

// Resident represents a pseudonymised resident — no PII stored
type Resident struct {
	ResidentPseudonymID string    `json:"resident_pseudonym_id"`
	AadhaarHash         string    `json:"-"` // never exposed in API responses
	AgeGroup            string    `json:"age_group"`
	Gender              string    `json:"gender"`
	SkinTone            string    `json:"skin_tone"`
	CreatedAt           time.Time `json:"created_at"`
}

// Session represents one data collection session per resident per operator
type Session struct {
	SessionID           string     `json:"session_id"`
	OperatorID          string     `json:"operator_id"`
	DeviceID            string     `json:"device_id"`
	CentreID            string     `json:"centre_id"`
	ResidentPseudonymID string     `json:"resident_pseudonym_id"`
	Status              string     `json:"status"`
	StartedAt           time.Time  `json:"started_at"`
	ClosedAt            *time.Time `json:"closed_at"`
	CloseReason         string     `json:"close_reason"`
}

// Consent represents resident consent record for a session — append only
type Consent struct {
	ConsentID           string    `json:"consent_id"`
	SessionID           string    `json:"session_id"`
	ResidentPseudonymID string    `json:"resident_pseudonym_id"`
	Consented           bool      `json:"consented"`
	LanguageShown       string    `json:"language_shown"`
	OperatorID          string    `json:"operator_id"`
	CreatedAt           time.Time `json:"created_at"`
}

// Capture represents one fingerprint capture record
// Image is stored in CEPH, only the reference key is stored here
type Capture struct {
	CaptureID           string     `json:"capture_id"`
	SessionID           string     `json:"session_id"`
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

// AuditLog represents an immutable audit event — no updates or deletes ever
type AuditLog struct {
	LogID       string    `json:"log_id"`
	EventType   string    `json:"event_type"`
	OperatorID  string    `json:"operator_id"`
	SessionID   string    `json:"session_id"`
	DeviceID    string    `json:"device_id"`
	PayloadHash string    `json:"payload_hash"`
	IPAddress   string    `json:"ip_address"`
	CreatedAt   time.Time `json:"created_at"`
}

// ── Request / Response structs ──────────────────────────────────────────────

// ResidentLookupRequest is sent by Android when operator enters Aadhaar
type ResidentLookupRequest struct {
	AadhaarHash string `json:"aadhaar_hash" binding:"required"`
	AgeGroup    string `json:"age_group"`
	Gender      string `json:"gender"`
	SkinTone    string `json:"skin_tone"`
}

// ResidentLookupResponse returns resident info and session progress
type ResidentLookupResponse struct {
	ResidentPseudonymID string   `json:"resident_pseudonym_id"`
	CapturedFingers     []string `json:"captured_fingers"` // fingers already done
	PendingUploads      []string `json:"pending_uploads"`  // captures pending upload
	TotalCaptured       int      `json:"total_captured"`
	IsComplete          bool     `json:"is_complete"` // true if 6+ fingers done
}

// CreateSessionRequest is sent when starting a new capture session
type CreateSessionRequest struct {
	OperatorID          string `json:"operator_id" binding:"required"`
	DeviceID            string `json:"device_id" binding:"required"`
	CentreID            string `json:"centre_id" binding:"required"`
	ResidentPseudonymID string `json:"resident_pseudonym_id" binding:"required"`
}

// CaptureRequest is sent after each successful finger capture
type CaptureRequest struct {
	SessionID           string  `json:"session_id" binding:"required"`
	ResidentPseudonymID string  `json:"resident_pseudonym_id" binding:"required"`
	OperatorID          string  `json:"operator_id" binding:"required"`
	FingerType          string  `json:"finger_type" binding:"required"`
	Hand                string  `json:"hand" binding:"required"`
	Nfiq2Score          float64 `json:"nfiq2_score"`
	BlurScore           float64 `json:"blur_score"`
	BrightnessScore     float64 `json:"brightness_score"`
	GlareScore          float64 `json:"glare_score"`
	AttemptCount        int     `json:"attempt_count"`
	DegradedFlag        bool    `json:"degraded_flag"`
	ImageBase64         string  `json:"image_base64" binding:"required"`
	ImageChecksum       string  `json:"image_checksum"`
	CameraModel         string  `json:"camera_model"`
	CameraResolution    string  `json:"camera_resolution"`
	DeviceModel         string  `json:"device_model"`
}

// BatchCaptureRequest wraps multiple captures in one request
type BatchCaptureRequest struct {
	Captures []CaptureRequest `json:"captures" binding:"required"`
}

// CaptureResponse is returned after a successful capture upload
type CaptureResponse struct {
	CaptureID     string `json:"capture_id"`
	FingerType    string `json:"finger_type"`
	UploadStatus  string `json:"upload_status"`
	TotalCaptured int    `json:"total_captured"`
	IsComplete    bool   `json:"is_complete"`
}

// CloseSessionRequest is sent when session ends
type CloseSessionRequest struct {
	SessionID   string `json:"session_id" binding:"required"`
	CloseReason string `json:"close_reason"`
}

// DevResetRequest wipes all data for a resident — dev/test only
type DevResetRequest struct {
	AadhaarHash string `json:"aadhaar_hash" binding:"required"`
}
