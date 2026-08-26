package model

import "time"

// OverviewResponse summarizes an operator's overall capture progress
type OverviewResponse struct {
	TotalCaptured int            `json:"total_captured"`
	CapturedToday int            `json:"captured_today"`
	ByGender      map[string]int `json:"by_gender"`
	ByAgeGroup    map[string]int `json:"by_age_group"`
}

// QuotaStatus represents how close a demographic bracket is to its national target
type QuotaStatus string

const (
	QuotaStatusOpen QuotaStatus = "OPEN"
	QuotaStatusWarn QuotaStatus = "WARN"
	QuotaStatusFull QuotaStatus = "FULL"
)

// QuotaLine reports the quota status of a single demographic bracket
type QuotaLine struct {
	Key                 string      `json:"key"`
	TargetCount         int         `json:"target_count"`
	CapturedCount       int         `json:"captured_count"`
	SlotsOpen           int         `json:"slots_open"`
	Status              QuotaStatus `json:"status"`
	OverridesTotal      int         `json:"overrides_total"`
	OverridesByOperator int         `json:"overrides_by_operator"`
}

// DiversityResponse reports quota status across both demographic dimensions
type DiversityResponse struct {
	Gender   []QuotaLine `json:"gender"`
	AgeGroup []QuotaLine `json:"age_group"`
}

// FingersResponse summarizes an operator's finger-level capture stats
type FingersResponse struct {
	TotalFingers           int            `json:"total_fingers"`
	ResidentCount          int            `json:"resident_count"`
	ByFingerType           map[string]int `json:"by_finger_type"`
	ResidentsByFingerCount map[string]int `json:"residents_by_finger_count"`
}

// AlertsResponse lists all demographic brackets currently near or at capacity
type AlertsResponse struct {
	Alerts []QuotaAlert `json:"alerts"`
}

// QuotaAlert flags a single demographic bracket that is near/at capacity
type QuotaAlert struct {
	Dimension string      `json:"dimension"`
	Key       string      `json:"key"`
	Status    QuotaStatus `json:"status"`
	SlotsOpen int         `json:"slots_open"`
	Message   string      `json:"message"`
}

// QuotaCheckResponse reports quota status for a proposed gender+age_group pair,
// plus alternative brackets that still have open slots
type QuotaCheckResponse struct {
	Gender       QuotaCheckLine `json:"gender"`
	AgeGroup     QuotaCheckLine `json:"age_group"`
	Alternatives []Alternative  `json:"alternatives"`
}

// QuotaCheckLine reports the quota status of a single checked bracket
type QuotaCheckLine struct {
	Key       string      `json:"key"`
	Status    QuotaStatus `json:"status"`
	SlotsOpen int         `json:"slots_open"`
}

// Alternative suggests an open demographic bracket
type Alternative struct {
	Dimension string `json:"dimension"`
	Key       string `json:"key"`
	Label     string `json:"label"`
	SlotsOpen int    `json:"slots_open"`
}

// LogOverrideRequest is sent when an operator proceeds with a capture
// despite the resident's demographic bracket being full/near-full
type LogOverrideRequest struct {
	SessionID           string `json:"session_id" binding:"required"`
	ResidentPseudonymID string `json:"resident_pseudonym_id" binding:"required"`
	OperatorID          string `json:"operator_id" binding:"required"`
	Dimension           string `json:"dimension" binding:"required"`
	Key                 string `json:"key" binding:"required"`
}

// QuotaOverride represents a logged quota override record
type QuotaOverride struct {
	OverrideID          string    `json:"override_id"`
	SessionID           string    `json:"session_id"`
	ResidentPseudonymID string    `json:"resident_pseudonym_id"`
	OperatorID          string    `json:"operator_id"`
	Dimension           string    `json:"dimension"`
	Key                 string    `json:"key"`
	CreatedAt           time.Time `json:"created_at"`
}

// QuotaTarget represents a national capture target for a demographic bracket
type QuotaTarget struct {
	Dimension   string `json:"dimension"`
	Key         string `json:"key"`
	TargetCount int    `json:"target_count"`
}
