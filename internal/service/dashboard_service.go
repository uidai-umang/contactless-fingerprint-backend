package service

import (
	"strconv"

	"contactless-fingerprint-backend/internal/model"
	"contactless-fingerprint-backend/internal/repository"
)

type DashboardService struct {
	dashboardRepo *repository.DashboardRepository
}

func NewDashboardService(dashboardRepo *repository.DashboardRepository) *DashboardService {
	return &DashboardService{dashboardRepo: dashboardRepo}
}

// GetOverview summarizes an operator's overall capture progress
func (s *DashboardService) GetOverview(operatorID string) (*model.OverviewResponse, error) {
	total, err := s.dashboardRepo.GetOperatorTotalCaptured(operatorID)
	if err != nil {
		return nil, err
	}
	today, err := s.dashboardRepo.GetOperatorCapturedToday(operatorID)
	if err != nil {
		return nil, err
	}
	byGender, err := s.dashboardRepo.GetOperatorByGender(operatorID)
	if err != nil {
		return nil, err
	}
	byAgeGroup, err := s.dashboardRepo.GetOperatorByAgeGroup(operatorID)
	if err != nil {
		return nil, err
	}

	return &model.OverviewResponse{
		TotalCaptured: total,
		CapturedToday: today,
		ByGender:      byGender,
		ByAgeGroup:    byAgeGroup,
	}, nil
}

// GetFingerStats summarizes an operator's finger-level capture stats,
// normalizing resident-by-finger-count buckets to string keys '1'..'10'.
func (s *DashboardService) GetFingerStats(operatorID string) (*model.FingersResponse, error) {
	totalFingers, err := s.dashboardRepo.GetOperatorTotalFingers(operatorID)
	if err != nil {
		return nil, err
	}
	byFingerType, err := s.dashboardRepo.GetOperatorByFingerType(operatorID)
	if err != nil {
		return nil, err
	}
	residentsByFingerCount, err := s.dashboardRepo.GetOperatorResidentsByFingerCount(operatorID)
	if err != nil {
		return nil, err
	}

	residentCount := 0
	normalized := make(map[string]int, 10)
	for i := 1; i <= 10; i++ {
		count := residentsByFingerCount[i]
		normalized[strconv.Itoa(i)] = count
		residentCount += count
	}

	return &model.FingersResponse{
		TotalFingers:           totalFingers,
		ResidentCount:          residentCount,
		ByFingerType:           byFingerType,
		ResidentsByFingerCount: normalized,
	}, nil
}
