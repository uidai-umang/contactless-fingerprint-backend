package service

import (
	"strconv"

	"contactless-fingerprint-backend/internal/model"
	"contactless-fingerprint-backend/internal/repository"
)

const nearFullThresholdPct = 0.95

var genderKeys = []string{"MALE", "FEMALE", "OTHER"}
var ageGroupKeys = []string{"5-17", "18-40", "41-60", "60+"}

var genderLabels = map[string]string{
	"MALE":   "Male residents",
	"FEMALE": "Female residents",
	"OTHER":  "Other gender residents",
}

var ageGroupLabels = map[string]string{
	"5-17":  "Residents aged 5-17 yrs",
	"18-40": "Residents aged 18-40 yrs",
	"41-60": "Residents aged 41-60 yrs",
	"60+":   "Residents aged 60+ yrs",
}

type QuotaService struct {
	quotaRepo *repository.QuotaRepository
}

func NewQuotaService(quotaRepo *repository.QuotaRepository) *QuotaService {
	return &QuotaService{quotaRepo: quotaRepo}
}

// classify determines a bracket's quota status and remaining slots given its
// captured count and national target.
func classify(captured, target int) (model.QuotaStatus, int) {
	if target <= 0 {
		return model.QuotaStatusOpen, 0
	}

	slotsOpen := target - captured
	if slotsOpen <= 0 {
		return model.QuotaStatusFull, 0
	}

	if float64(captured) >= float64(target)*nearFullThresholdPct {
		return model.QuotaStatusWarn, slotsOpen
	}

	return model.QuotaStatusOpen, slotsOpen
}

func targetMapByKey(targets []model.QuotaTarget) map[string]int {
	m := make(map[string]int, len(targets))
	for _, t := range targets {
		m[t.Dimension+"|"+t.Key] = t.TargetCount
	}
	return m
}

// BuildDiversityLines returns quota status lines for every gender and
// age_group bracket, including how many overrides have been logged against
// each (total, and by the given operator).
func (s *QuotaService) BuildDiversityLines(operatorID string) (*model.DiversityResponse, error) {
	targets, err := s.quotaRepo.GetTargets()
	if err != nil {
		return nil, err
	}
	targetMap := targetMapByKey(targets)

	genderCaptured, err := s.quotaRepo.GetNationalCapturedByGender()
	if err != nil {
		return nil, err
	}
	ageCaptured, err := s.quotaRepo.GetNationalCapturedByAgeGroup()
	if err != nil {
		return nil, err
	}

	genderLines, err := s.buildQuotaLines("GENDER", genderKeys, targetMap, genderCaptured, operatorID)
	if err != nil {
		return nil, err
	}
	ageLines, err := s.buildQuotaLines("AGE_GROUP", ageGroupKeys, targetMap, ageCaptured, operatorID)
	if err != nil {
		return nil, err
	}

	return &model.DiversityResponse{Gender: genderLines, AgeGroup: ageLines}, nil
}

func (s *QuotaService) buildQuotaLines(dimension string, keys []string, targetMap map[string]int, captured map[string]int, operatorID string) ([]model.QuotaLine, error) {
	lines := make([]model.QuotaLine, 0, len(keys))
	for _, key := range keys {
		target := targetMap[dimension+"|"+key]
		capturedCount := captured[key]
		status, slotsOpen := classify(capturedCount, target)

		total, byOperator, err := s.quotaRepo.CountOverrides(dimension, key, operatorID)
		if err != nil {
			return nil, err
		}

		lines = append(lines, model.QuotaLine{
			Key:                 key,
			TargetCount:         target,
			CapturedCount:       capturedCount,
			SlotsOpen:           slotsOpen,
			Status:              status,
			OverridesTotal:      total,
			OverridesByOperator: byOperator,
		})
	}
	return lines, nil
}

// BuildAlerts returns one alert per demographic bracket that is currently
// WARN or FULL. OPEN brackets are omitted.
func (s *QuotaService) BuildAlerts() (*model.AlertsResponse, error) {
	targets, err := s.quotaRepo.GetTargets()
	if err != nil {
		return nil, err
	}
	targetMap := targetMapByKey(targets)

	genderCaptured, err := s.quotaRepo.GetNationalCapturedByGender()
	if err != nil {
		return nil, err
	}
	ageCaptured, err := s.quotaRepo.GetNationalCapturedByAgeGroup()
	if err != nil {
		return nil, err
	}

	alerts := []model.QuotaAlert{}
	alerts = append(alerts, buildAlerts("GENDER", genderKeys, targetMap, genderCaptured, genderLabels)...)
	alerts = append(alerts, buildAlerts("AGE_GROUP", ageGroupKeys, targetMap, ageCaptured, ageGroupLabels)...)

	return &model.AlertsResponse{Alerts: alerts}, nil
}

func buildAlerts(dimension string, keys []string, targetMap map[string]int, captured map[string]int, labels map[string]string) []model.QuotaAlert {
	var alerts []model.QuotaAlert
	for _, key := range keys {
		target := targetMap[dimension+"|"+key]
		capturedCount := captured[key]
		status, slotsOpen := classify(capturedCount, target)
		if status == model.QuotaStatusOpen {
			continue
		}

		label := labels[key]
		message := "NEAR FULL: " + label + " — " + strconv.Itoa(slotsOpen) + " slots left"
		if status == model.QuotaStatusFull {
			message = "QUOTA FULL: " + label
		}

		alerts = append(alerts, model.QuotaAlert{
			Dimension: dimension,
			Key:       key,
			Status:    status,
			SlotsOpen: slotsOpen,
			Message:   message,
		})
	}
	return alerts
}

// CheckQuota reports the quota status for a proposed gender+age_group pair,
// plus every OPEN bracket across both dimensions as an alternative.
func (s *QuotaService) CheckQuota(gender, ageGroup string) (*model.QuotaCheckResponse, error) {
	targets, err := s.quotaRepo.GetTargets()
	if err != nil {
		return nil, err
	}
	targetMap := targetMapByKey(targets)

	genderCaptured, err := s.quotaRepo.GetNationalCapturedByGender()
	if err != nil {
		return nil, err
	}
	ageCaptured, err := s.quotaRepo.GetNationalCapturedByAgeGroup()
	if err != nil {
		return nil, err
	}

	genderStatus, genderSlots := classify(genderCaptured[gender], targetMap["GENDER|"+gender])
	ageStatus, ageSlots := classify(ageCaptured[ageGroup], targetMap["AGE_GROUP|"+ageGroup])

	alternatives := []model.Alternative{}
	for _, key := range genderKeys {
		status, slots := classify(genderCaptured[key], targetMap["GENDER|"+key])
		if status == model.QuotaStatusOpen {
			alternatives = append(alternatives, model.Alternative{
				Dimension: "GENDER", Key: key, Label: genderLabels[key], SlotsOpen: slots,
			})
		}
	}
	for _, key := range ageGroupKeys {
		status, slots := classify(ageCaptured[key], targetMap["AGE_GROUP|"+key])
		if status == model.QuotaStatusOpen {
			alternatives = append(alternatives, model.Alternative{
				Dimension: "AGE_GROUP", Key: key, Label: ageGroupLabels[key], SlotsOpen: slots,
			})
		}
	}

	return &model.QuotaCheckResponse{
		Gender:       model.QuotaCheckLine{Key: gender, Status: genderStatus, SlotsOpen: genderSlots},
		AgeGroup:     model.QuotaCheckLine{Key: ageGroup, Status: ageStatus, SlotsOpen: ageSlots},
		Alternatives: alternatives,
	}, nil
}

// LogOverride records that an operator proceeded with a capture despite the
// resident's demographic bracket being full/near-full.
func (s *QuotaService) LogOverride(req model.LogOverrideRequest) (*model.QuotaOverride, error) {
	return s.quotaRepo.InsertOverride(req)
}
