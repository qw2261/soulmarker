package service

import (
	"encoding/hex"
	"strings"
	"time"

	"github.com/qw2261/soulmarker/event_go/internal/clock"
	"github.com/qw2261/soulmarker/event_go/internal/model"
)

const AdmissionCredentialPrefix = model.AdmissionCredentialPrefix

type AdmissionRepository interface {
	GetAdmissionByUser(eventID, userID int64) (*model.MyAdmission, error)
	ListMyAdmissions(userID int64, offset, limit int) ([]*model.MyAdmission, int, error)
	CheckIn(eventID int64, credentialCode, actor string, checkedInAt time.Time) (*model.Checkin, bool, error)
	ListCheckins(eventID int64, offset, limit int) ([]*model.Checkin, int, error)
}

type AdmissionService struct {
	repository AdmissionRepository
	clock      clock.Clock
}

func NewAdmissionService(repository AdmissionRepository, businessClock clock.Clock) *AdmissionService {
	return &AdmissionService{repository: repository, clock: businessClock}
}

func (s *AdmissionService) GetForUser(eventID, userID int64) (*model.MyAdmission, error) {
	return s.repository.GetAdmissionByUser(eventID, userID)
}

func (s *AdmissionService) ListForUser(userID int64, offset, limit int) ([]*model.MyAdmission, int, error) {
	return s.repository.ListMyAdmissions(userID, offset, limit)
}

func normalizeCredential(value string) (string, bool) {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, AdmissionCredentialPrefix)
	if len(value) != 32 {
		return "", false
	}
	if _, err := hex.DecodeString(value); err != nil {
		return "", false
	}
	return strings.ToLower(value), true
}

func (s *AdmissionService) CheckIn(eventID int64, credential, actor string) (*model.Checkin, bool, error) {
	code, ok := normalizeCredential(credential)
	if !ok {
		return nil, false, model.ErrAdmissionNotFound
	}
	return s.repository.CheckIn(eventID, code, actor, s.clock.Now())
}

func (s *AdmissionService) ListCheckins(eventID int64, offset, limit int) ([]*model.Checkin, int, error) {
	return s.repository.ListCheckins(eventID, offset, limit)
}
