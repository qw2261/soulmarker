package service

import (
	"errors"

	"github.com/qw2261/soulmarker/event_go/internal/model"
)

var (
	ErrPostNotFound         = errors.New("帖子不存在")
	ErrPostTitleRequired    = errors.New("帖子标题不能为空")
	ErrPostContentRequired  = errors.New("帖子内容不能为空")
	ErrReplyContentRequired = errors.New("回复内容不能为空")
)

// DiscussionRepository 只描述讨论写入用例需要的数据能力。
type DiscussionRepository interface {
	GetEvent(id int64) (*model.Event, error)
	GetPost(postID int64) (*model.Post, error)
	IsRegisteredByUserID(eventID, userID int64) (bool, error)
	CreatePost(post *model.Post) error
	CreateReply(reply *model.Reply) error
}

type DiscussionService struct {
	repository DiscussionRepository
}

func NewDiscussionService(repository DiscussionRepository) *DiscussionService {
	return &DiscussionService{repository: repository}
}

func (s *DiscussionService) GetEvent(eventID int64) (*model.Event, error) {
	event, err := s.repository.GetEvent(eventID)
	if err != nil {
		return nil, err
	}
	if event == nil {
		return nil, model.ErrNotFound
	}
	return event, nil
}

func (s *DiscussionService) GetPost(eventID, postID int64) (*model.Post, error) {
	post, err := s.repository.GetPost(postID)
	if err != nil {
		return nil, err
	}
	if post == nil || post.EventID != eventID {
		return nil, ErrPostNotFound
	}
	return post, nil
}

func (s *DiscussionService) CreatePost(eventID int64, user *model.User, title, content string) (*model.Post, error) {
	if err := s.requireRegistration(eventID, user.ID); err != nil {
		return nil, err
	}
	if title == "" {
		return nil, ErrPostTitleRequired
	}
	if content == "" {
		return nil, ErrPostContentRequired
	}

	userID := user.ID
	post := &model.Post{
		EventID:       eventID,
		UserID:        &userID,
		AuthorName:    user.Name,
		AuthorContact: user.Contact,
		Title:         title,
		Content:       content,
	}
	if err := s.repository.CreatePost(post); err != nil {
		return nil, err
	}
	return post, nil
}

func (s *DiscussionService) CreateReply(post *model.Post, user *model.User, content string) (*model.Reply, error) {
	if err := s.requireRegistration(post.EventID, user.ID); err != nil {
		return nil, err
	}
	if content == "" {
		return nil, ErrReplyContentRequired
	}

	userID := user.ID
	reply := &model.Reply{
		PostID:        post.ID,
		UserID:        &userID,
		AuthorName:    user.Name,
		AuthorContact: user.Contact,
		Content:       content,
	}
	if err := s.repository.CreateReply(reply); err != nil {
		return nil, err
	}
	return reply, nil
}

func (s *DiscussionService) requireRegistration(eventID, userID int64) error {
	registered, err := s.repository.IsRegisteredByUserID(eventID, userID)
	if err != nil {
		return err
	}
	if !registered {
		return model.ErrNotRegistered
	}
	return nil
}
