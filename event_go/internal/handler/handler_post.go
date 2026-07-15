package handler

import (
	"errors"
	"net/http"

	"github.com/qw2261/soulmarker/event_go/internal/model"
	"github.com/qw2261/soulmarker/event_go/internal/service"
)

func (h *Handler) getDiscussionEventOr404(w http.ResponseWriter, eventID int64) (*model.Event, bool) {
	event, err := h.discussions.GetEvent(eventID)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, model.APIResp{Code: 404, Message: model.ErrNotFound.Error()})
		} else {
			writeInternalError(w, "get_discussion_event", err)
		}
		return nil, false
	}
	return event, true
}

func (h *Handler) getDiscussionPostOr404(w http.ResponseWriter, eventID, postID int64) (*model.Post, bool) {
	post, err := h.discussions.GetPost(eventID, postID)
	if err != nil {
		if errors.Is(err, service.ErrPostNotFound) {
			writeJSON(w, http.StatusNotFound, model.APIResp{Code: 404, Message: service.ErrPostNotFound.Error()})
		} else {
			writeInternalError(w, "get_discussion_post", err)
		}
		return nil, false
	}
	return post, true
}

func (h *Handler) CreatePost(w http.ResponseWriter, r *http.Request) {
	user, authenticated := h.requireUser(w, r)
	if !authenticated {
		return
	}

	eventID, err := parseEventID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "无效的活动 ID"})
		return
	}

	_, ok := h.getDiscussionEventOr404(w, eventID)
	if !ok {
		return
	}

	var req model.CreatePostReq
	if !decodeJSON(w, r, &req) {
		return
	}

	post, err := h.discussions.CreatePost(eventID, user, req.Title, req.Content)
	if err != nil {
		switch {
		case errors.Is(err, model.ErrNotRegistered):
			writeJSON(w, http.StatusForbidden, model.APIResp{Code: 403, Message: err.Error()})
		case errors.Is(err, service.ErrPostTitleRequired), errors.Is(err, service.ErrPostContentRequired):
			writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: err.Error()})
		default:
			writeInternalError(w, "create_post", err)
		}
		return
	}

	writeJSON(w, http.StatusCreated, model.APIResp{Code: 201, Message: "发帖成功", Data: post})
}

func (h *Handler) ListPosts(w http.ResponseWriter, r *http.Request) {
	eventID, err := parseEventID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "无效的活动 ID"})
		return
	}

	_, ok := h.getEventOr404(w, eventID)
	if !ok {
		return
	}

	page, pageSize := parsePagination(r)
	offset := (page - 1) * pageSize

	posts, total, err := h.store.ListPosts(eventID, offset, pageSize)
	if err != nil {
		writeInternalError(w, "list_posts", err)
		return
	}

	paginatedOK(w, posts, total, page, pageSize)
}

func (h *Handler) GetPost(w http.ResponseWriter, r *http.Request) {
	eventID, err := parseEventID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "无效的活动 ID"})
		return
	}

	postID, err := parsePostID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "无效的帖子 ID"})
		return
	}

	post, ok := h.getDiscussionPostOr404(w, eventID, postID)
	if !ok {
		return
	}

	replies, err := h.store.ListReplies(postID)
	if err != nil {
		writeInternalError(w, "list_replies", err)
		return
	}

	result := model.PostDetailResp{
		Post:    post,
		Replies: replies,
	}

	writeJSON(w, http.StatusOK, model.APIResp{Code: 200, Message: "ok", Data: result})
}

func (h *Handler) CreateReply(w http.ResponseWriter, r *http.Request) {
	user, authenticated := h.requireUser(w, r)
	if !authenticated {
		return
	}

	eventID, err := parseEventID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "无效的活动 ID"})
		return
	}

	postID, err := parsePostID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "无效的帖子 ID"})
		return
	}

	post, ok := h.getDiscussionPostOr404(w, eventID, postID)
	if !ok {
		return
	}

	var req model.CreateReplyReq
	if !decodeJSON(w, r, &req) {
		return
	}

	reply, err := h.discussions.CreateReply(post, user, req.Content)
	if err != nil {
		switch {
		case errors.Is(err, model.ErrNotRegistered):
			writeJSON(w, http.StatusForbidden, model.APIResp{Code: 403, Message: err.Error()})
		case errors.Is(err, service.ErrReplyContentRequired):
			writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: err.Error()})
		default:
			writeInternalError(w, "create_reply", err)
		}
		return
	}

	writeJSON(w, http.StatusCreated, model.APIResp{Code: 201, Message: "回复成功", Data: reply})
}
