package service

import (
	"errors"
	"testing"

	"github.com/qw2261/soulmarker/event_go/internal/model"
)

type fakeDiscussionRepository struct {
	event           *model.Event
	post            *model.Post
	getEventErr     error
	getPostErr      error
	registered      bool
	registrationErr error
	createPostErr   error
	createReplyErr  error
	createdPost     *model.Post
	createdReply    *model.Reply
}

func (r *fakeDiscussionRepository) GetEvent(int64) (*model.Event, error) {
	return r.event, r.getEventErr
}

func (r *fakeDiscussionRepository) GetPost(int64) (*model.Post, error) {
	return r.post, r.getPostErr
}

func (r *fakeDiscussionRepository) IsRegisteredByUserID(int64, int64) (bool, error) {
	return r.registered, r.registrationErr
}

func (r *fakeDiscussionRepository) CreatePost(post *model.Post) error {
	r.createdPost = post
	return r.createPostErr
}

func (r *fakeDiscussionRepository) CreateReply(reply *model.Reply) error {
	r.createdReply = reply
	return r.createReplyErr
}

func TestDiscussionServiceLoadsScopedResources(t *testing.T) {
	repository := &fakeDiscussionRepository{}
	service := NewDiscussionService(repository)
	if _, err := service.GetEvent(1); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("expected event not found, got %v", err)
	}

	repository.event = &model.Event{ID: 1}
	if event, err := service.GetEvent(1); err != nil || event.ID != 1 {
		t.Fatalf("unexpected event result: event=%+v err=%v", event, err)
	}

	repository.post = &model.Post{ID: 2, EventID: 10}
	if _, err := service.GetPost(11, 2); !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("cross-event post must be hidden, got %v", err)
	}
	if post, err := service.GetPost(10, 2); err != nil || post.ID != 2 {
		t.Fatalf("unexpected post result: post=%+v err=%v", post, err)
	}
}

func TestDiscussionServicePropagatesReadFailures(t *testing.T) {
	repository := &fakeDiscussionRepository{getEventErr: errors.New("event lookup failed")}
	service := NewDiscussionService(repository)
	if _, err := service.GetEvent(1); err == nil || errors.Is(err, model.ErrNotFound) {
		t.Fatalf("expected event repository failure, got %v", err)
	}

	repository.getEventErr = nil
	repository.getPostErr = errors.New("post lookup failed")
	if _, err := service.GetPost(1, 2); err == nil || errors.Is(err, ErrPostNotFound) {
		t.Fatalf("expected post repository failure, got %v", err)
	}
}

func TestDiscussionServiceCreatePostCopiesTrustedIdentity(t *testing.T) {
	repository := &fakeDiscussionRepository{registered: true}
	service := NewDiscussionService(repository)
	user := &model.User{ID: 7, Name: "可信作者", Contact: "author@example.com"}

	post, err := service.CreatePost(8, user, "标题", "内容")
	if err != nil {
		t.Fatal(err)
	}
	if repository.createdPost != post || post.UserID == nil || *post.UserID != user.ID {
		t.Fatalf("post was not persisted with trusted user id: %+v", post)
	}
	if post.AuthorName != user.Name || post.AuthorContact != user.Contact || post.EventID != 8 {
		t.Fatalf("post identity mismatch: %+v", post)
	}
}

func TestDiscussionServiceCreatePostRejectsInOrder(t *testing.T) {
	repository := &fakeDiscussionRepository{}
	service := NewDiscussionService(repository)
	user := &model.User{ID: 1}

	if _, err := service.CreatePost(1, user, "", ""); !errors.Is(err, model.ErrNotRegistered) {
		t.Fatalf("registration must be checked first, got %v", err)
	}
	repository.registered = true
	if _, err := service.CreatePost(1, user, "", "内容"); !errors.Is(err, ErrPostTitleRequired) {
		t.Fatalf("expected title error, got %v", err)
	}
	if _, err := service.CreatePost(1, user, "标题", ""); !errors.Is(err, ErrPostContentRequired) {
		t.Fatalf("expected content error, got %v", err)
	}
	if repository.createdPost != nil {
		t.Fatal("invalid post must not be persisted")
	}
}

func TestDiscussionServiceCreateReplyCopiesTrustedIdentity(t *testing.T) {
	repository := &fakeDiscussionRepository{registered: true}
	service := NewDiscussionService(repository)
	post := &model.Post{ID: 3, EventID: 4}
	user := &model.User{ID: 5, Name: "回复者", Contact: "reply@example.com"}

	reply, err := service.CreateReply(post, user, "回复内容")
	if err != nil {
		t.Fatal(err)
	}
	if repository.createdReply != reply || reply.UserID == nil || *reply.UserID != user.ID {
		t.Fatalf("reply was not persisted with trusted user id: %+v", reply)
	}
	if reply.PostID != post.ID || reply.AuthorName != user.Name || reply.AuthorContact != user.Contact {
		t.Fatalf("reply identity mismatch: %+v", reply)
	}
}

func TestDiscussionServiceCreateReplyRejectsInOrder(t *testing.T) {
	repository := &fakeDiscussionRepository{}
	service := NewDiscussionService(repository)
	post := &model.Post{ID: 2, EventID: 3}
	user := &model.User{ID: 1}

	if _, err := service.CreateReply(post, user, ""); !errors.Is(err, model.ErrNotRegistered) {
		t.Fatalf("registration must be checked first, got %v", err)
	}
	repository.registered = true
	if _, err := service.CreateReply(post, user, ""); !errors.Is(err, ErrReplyContentRequired) {
		t.Fatalf("expected reply content error, got %v", err)
	}
	if repository.createdReply != nil {
		t.Fatal("invalid reply must not be persisted")
	}
}

func TestDiscussionServicePropagatesEligibilityAndWriteFailures(t *testing.T) {
	repository := &fakeDiscussionRepository{registrationErr: errors.New("registration lookup failed")}
	service := NewDiscussionService(repository)
	user := &model.User{ID: 1}
	post := &model.Post{ID: 2, EventID: 3}

	if _, err := service.CreatePost(3, user, "标题", "内容"); err == nil {
		t.Fatal("expected eligibility lookup failure")
	}
	repository.registrationErr = nil
	repository.registered = true
	repository.createPostErr = errors.New("post write failed")
	if _, err := service.CreatePost(3, user, "标题", "内容"); err == nil {
		t.Fatal("expected post write failure")
	}

	repository.createReplyErr = errors.New("reply write failed")
	if _, err := service.CreateReply(post, user, "内容"); err == nil {
		t.Fatal("expected reply write failure")
	}
}
