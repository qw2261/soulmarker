package service

import (
	"errors"
	"strings"
	"time"

	"github.com/qw2261/soulmarker/event_go/internal/clock"
	"github.com/qw2261/soulmarker/event_go/internal/model"
)

var (
	ErrContentReportCategoryInvalid  = errors.New("无效的举报分类")
	ErrContentReportDetailRequired   = errors.New("其他分类必须填写举报说明")
	ErrContentReportDetailTooLong    = errors.New("举报说明不能超过 1000 个字符")
	ErrContentReportOwnContent       = errors.New("不能举报自己发布的内容")
	ErrContentReportStatusInvalid    = errors.New("无效的举报状态")
	ErrContentTargetTypeInvalid      = errors.New("无效的内容类型")
	ErrContentResolutionInvalid      = errors.New("无效的处理结论")
	ErrContentModerationNoteRequired = errors.New("处理说明不能为空")
	ErrContentModerationNoteTooLong  = errors.New("处理说明不能超过 1000 个字符")
)

type ContentModerationRepository interface {
	GetEvent(id int64) (*model.Event, error)
	GetPostForModeration(postID int64) (*model.Post, error)
	GetReplyForModeration(replyID int64) (*model.Reply, error)
	IsRegisteredByUserID(eventID, userID int64) (bool, error)
	CreateContentReport(report *model.ContentReport) (bool, error)
	ListContentReports(params model.ListContentReportsParams) ([]*model.ContentReport, int, error)
	ResolveContentReport(id int64, resolution, note, actor string, now time.Time) (*model.ContentReport, error)
	ModerateContent(eventID, postID int64, targetType string, targetID int64, remove bool, actor, reason string, now time.Time) error
	ListContentModerationActions(offset, limit int) ([]*model.ContentModerationAction, int, error)
}

type ContentModerationService struct {
	repository ContentModerationRepository
	clock      clock.Clock
}

func NewContentModerationService(repository ContentModerationRepository, businessClock clock.Clock) *ContentModerationService {
	return &ContentModerationService{repository: repository, clock: businessClock}
}

func validContentReportCategory(category string) bool {
	return map[string]bool{
		model.ContentReportCategorySpam: true, model.ContentReportCategoryAbuse: true,
		model.ContentReportCategoryIllegal: true, model.ContentReportCategoryPrivacy: true,
		model.ContentReportCategoryOther: true,
	}[category]
}

func normalizeReportInput(category, detail string) (string, string, error) {
	category = strings.TrimSpace(category)
	detail = strings.TrimSpace(detail)
	if !validContentReportCategory(category) {
		return "", "", ErrContentReportCategoryInvalid
	}
	if len([]rune(detail)) > 1000 {
		return "", "", ErrContentReportDetailTooLong
	}
	if category == model.ContentReportCategoryOther && detail == "" {
		return "", "", ErrContentReportDetailRequired
	}
	return category, detail, nil
}

func normalizeModerationNote(note string) (string, error) {
	note = strings.TrimSpace(note)
	if note == "" {
		return "", ErrContentModerationNoteRequired
	}
	if len([]rune(note)) > 1000 {
		return "", ErrContentModerationNoteTooLong
	}
	return note, nil
}

func (s *ContentModerationService) requireParticipant(eventID, userID int64) error {
	registered, err := s.repository.IsRegisteredByUserID(eventID, userID)
	if err != nil {
		return err
	}
	if !registered {
		return model.ErrNotRegistered
	}
	return nil
}

func (s *ContentModerationService) ReportPost(eventID, postID int64, user *model.User, category, detail string) (*model.ContentReport, bool, error) {
	category, detail, err := normalizeReportInput(category, detail)
	if err != nil {
		return nil, false, err
	}
	post, err := s.repository.GetPostForModeration(postID)
	if err != nil {
		return nil, false, err
	}
	if post == nil || post.EventID != eventID || post.ModerationStatus != model.ModerationStatusVisible {
		return nil, false, model.ErrContentTargetNotFound
	}
	if post.UserID != nil && *post.UserID == user.ID {
		return nil, false, ErrContentReportOwnContent
	}
	if err := s.requireParticipant(eventID, user.ID); err != nil {
		return nil, false, err
	}
	report := &model.ContentReport{
		EventID: eventID, PostID: postID, TargetType: model.ContentTargetPost, TargetID: postID,
		ReporterUserID: user.ID, ReporterName: user.Name, Category: category, Detail: detail,
		Status: model.ContentReportStatusOpen, CreatedAt: s.clock.Now(),
	}
	created, err := s.repository.CreateContentReport(report)
	return report, created, err
}

func (s *ContentModerationService) ReportReply(eventID, postID, replyID int64, user *model.User, category, detail string) (*model.ContentReport, bool, error) {
	category, detail, err := normalizeReportInput(category, detail)
	if err != nil {
		return nil, false, err
	}
	post, err := s.repository.GetPostForModeration(postID)
	if err != nil {
		return nil, false, err
	}
	reply, err := s.repository.GetReplyForModeration(replyID)
	if err != nil {
		return nil, false, err
	}
	if post == nil || post.EventID != eventID || reply == nil || reply.PostID != postID ||
		post.ModerationStatus != model.ModerationStatusVisible || reply.ModerationStatus != model.ModerationStatusVisible {
		return nil, false, model.ErrContentTargetNotFound
	}
	if reply.UserID != nil && *reply.UserID == user.ID {
		return nil, false, ErrContentReportOwnContent
	}
	if err := s.requireParticipant(eventID, user.ID); err != nil {
		return nil, false, err
	}
	report := &model.ContentReport{
		EventID: eventID, PostID: postID, TargetType: model.ContentTargetReply, TargetID: replyID,
		ReporterUserID: user.ID, ReporterName: user.Name, Category: category, Detail: detail,
		Status: model.ContentReportStatusOpen, CreatedAt: s.clock.Now(),
	}
	created, err := s.repository.CreateContentReport(report)
	return report, created, err
}

func (s *ContentModerationService) ListReports(params model.ListContentReportsParams) ([]*model.ContentReport, int, error) {
	if params.Status != "" && params.Status != model.ContentReportStatusOpen &&
		params.Status != model.ContentReportStatusResolved && params.Status != model.ContentReportStatusDismissed {
		return nil, 0, ErrContentReportStatusInvalid
	}
	if params.TargetType != "" && params.TargetType != model.ContentTargetPost && params.TargetType != model.ContentTargetReply {
		return nil, 0, ErrContentTargetTypeInvalid
	}
	return s.repository.ListContentReports(params)
}

func (s *ContentModerationService) ResolveReport(id int64, resolution, note, actor string) (*model.ContentReport, error) {
	if resolution != model.ContentResolutionRemove && resolution != model.ContentResolutionDismiss {
		return nil, ErrContentResolutionInvalid
	}
	note, err := normalizeModerationNote(note)
	if err != nil {
		return nil, err
	}
	return s.repository.ResolveContentReport(id, resolution, note, actor, s.clock.Now())
}

func (s *ContentModerationService) moderate(eventID, postID int64, targetType string, targetID int64, remove bool, actor, reason string) error {
	reason, err := normalizeModerationNote(reason)
	if err != nil {
		return err
	}
	post, err := s.repository.GetPostForModeration(postID)
	if err != nil {
		return err
	}
	if post == nil || post.EventID != eventID {
		return model.ErrContentTargetNotFound
	}
	if targetType == model.ContentTargetReply {
		reply, err := s.repository.GetReplyForModeration(targetID)
		if err != nil {
			return err
		}
		if reply == nil || reply.PostID != postID {
			return model.ErrContentTargetNotFound
		}
	} else if targetType != model.ContentTargetPost || targetID != postID {
		return model.ErrContentTargetNotFound
	}
	return s.repository.ModerateContent(eventID, postID, targetType, targetID, remove, actor, reason, s.clock.Now())
}

func (s *ContentModerationService) RemovePost(eventID, postID int64, actor, reason string) error {
	return s.moderate(eventID, postID, model.ContentTargetPost, postID, true, actor, reason)
}

func (s *ContentModerationService) RestorePost(eventID, postID int64, actor, reason string) error {
	return s.moderate(eventID, postID, model.ContentTargetPost, postID, false, actor, reason)
}

func (s *ContentModerationService) RemoveReply(eventID, postID, replyID int64, actor, reason string) error {
	return s.moderate(eventID, postID, model.ContentTargetReply, replyID, true, actor, reason)
}

func (s *ContentModerationService) RestoreReply(eventID, postID, replyID int64, actor, reason string) error {
	return s.moderate(eventID, postID, model.ContentTargetReply, replyID, false, actor, reason)
}

func (s *ContentModerationService) ListActions(offset, limit int) ([]*model.ContentModerationAction, int, error) {
	return s.repository.ListContentModerationActions(offset, limit)
}
