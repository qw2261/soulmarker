package service

import (
	"errors"
	"testing"
	"time"

	"github.com/qw2261/soulmarker/event_go/internal/model"
)

type fakeAdmissionRepository struct {
	credential string
	actor      string
	checkedAt  time.Time
	checkin    *model.Checkin
	duplicate  bool
	err        error
	admission  *model.MyAdmission
	admissions []*model.MyAdmission
	checkins   []*model.Checkin
	total      int
}

func (r *fakeAdmissionRepository) GetAdmissionByUser(int64, int64) (*model.MyAdmission, error) {
	return r.admission, r.err
}

func (r *fakeAdmissionRepository) ListMyAdmissions(int64, int, int) ([]*model.MyAdmission, int, error) {
	return r.admissions, r.total, r.err
}

func (r *fakeAdmissionRepository) CheckIn(_ int64, credentialCode, actor string, checkedInAt time.Time) (*model.Checkin, bool, error) {
	r.credential = credentialCode
	r.actor = actor
	r.checkedAt = checkedInAt
	return r.checkin, r.duplicate, r.err
}

func (r *fakeAdmissionRepository) ListCheckins(int64, int, int) ([]*model.Checkin, int, error) {
	return r.checkins, r.total, r.err
}

func TestAdmissionServiceNormalizesCredentialAndUsesClock(t *testing.T) {
	now := time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC)
	repository := &fakeAdmissionRepository{checkin: &model.Checkin{ID: 1}}
	service := NewAdmissionService(repository, fixedClock{now: now})
	credential := AdmissionCredentialPrefix + "00112233445566778899AABBCCDDEEFF"
	checkin, duplicate, err := service.CheckIn(7, credential, "admin")
	if err != nil || duplicate || checkin.ID != 1 {
		t.Fatalf("unexpected checkin result: checkin=%+v duplicate=%v err=%v", checkin, duplicate, err)
	}
	if repository.credential != "00112233445566778899aabbccddeeff" || repository.actor != "admin" || !repository.checkedAt.Equal(now) {
		t.Fatalf("input was not normalized: %+v", repository)
	}
}

func TestAdmissionServiceRejectsMalformedCredential(t *testing.T) {
	repository := &fakeAdmissionRepository{}
	service := NewAdmissionService(repository, fixedClock{})
	for _, credential := range []string{"", "short", AdmissionCredentialPrefix + "not-hex-not-hex-not-hex-not-hex-"} {
		if _, _, err := service.CheckIn(1, credential, "admin"); !errors.Is(err, model.ErrAdmissionNotFound) {
			t.Fatalf("credential %q: expected not found, got %v", credential, err)
		}
	}
	if repository.credential != "" {
		t.Fatal("malformed credential reached repository")
	}
}
