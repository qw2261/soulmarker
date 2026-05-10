package handler

import (
	"encoding/json"
	"net/http"

	"github.com/qw2261/soulmarker/event_go/internal/model"
)

func (h *Handler) CreatePost(w http.ResponseWriter, r *http.Request) {
	eventID, err := parseEventID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "无效的活动 ID"})
		return
	}

	_, ok := h.getEventOr404(w, eventID)
	if !ok {
		return
	}

	var req model.CreatePostReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "请求体格式错误"})
		return
	}

	authorName, authorContact := getUserIdentity(r)
	if authorContact == "" {
		authorName = req.AuthorName
		authorContact = req.AuthorContact
	}

	if authorContact == "" {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "联系方式不能为空，请先登录或传入 author_contact"})
		return
	}
	if !h.checkRegistration(w, eventID, authorContact) {
		return
	}
	if req.Title == "" {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "帖子标题不能为空"})
		return
	}
	if req.Content == "" {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "帖子内容不能为空"})
		return
	}

	post := &model.Post{
		EventID:       eventID,
		AuthorName:    authorName,
		AuthorContact: authorContact,
		Title:         req.Title,
		Content:       req.Content,
	}
	if err := h.store.CreatePost(post); err != nil {
		writeJSON(w, http.StatusInternalServerError, model.APIResp{Code: 500, Message: err.Error()})
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
		writeJSON(w, http.StatusInternalServerError, model.APIResp{Code: 500, Message: err.Error()})
		return
	}

	paginatedOK(w, posts, total, page, pageSize)
}

func (h *Handler) GetPost(w http.ResponseWriter, r *http.Request) {
	postID, err := parsePostID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "无效的帖子 ID"})
		return
	}

	post, err := h.store.GetPost(postID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, model.APIResp{Code: 500, Message: err.Error()})
		return
	}
	if post == nil {
		writeJSON(w, http.StatusNotFound, model.APIResp{Code: 404, Message: "帖子不存在"})
		return
	}

	replies, err := h.store.ListReplies(postID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, model.APIResp{Code: 500, Message: err.Error()})
		return
	}

	result := map[string]interface{}{
		"post":    post,
		"replies": replies,
	}

	writeJSON(w, http.StatusOK, model.APIResp{Code: 200, Message: "ok", Data: result})
}

func (h *Handler) CreateReply(w http.ResponseWriter, r *http.Request) {
	postID, err := parsePostID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "无效的帖子 ID"})
		return
	}

	post, err := h.store.GetPost(postID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, model.APIResp{Code: 500, Message: err.Error()})
		return
	}
	if post == nil {
		writeJSON(w, http.StatusNotFound, model.APIResp{Code: 404, Message: "帖子不存在"})
		return
	}

	var req model.CreateReplyReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "请求体格式错误"})
		return
	}

	authorName, authorContact := getUserIdentity(r)
	if authorContact == "" {
		authorName = req.AuthorName
		authorContact = req.AuthorContact
	}

	if authorContact == "" {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "联系方式不能为空，请先登录或传入 author_contact"})
		return
	}
	if !h.checkRegistration(w, post.EventID, authorContact) {
		return
	}
	if req.Content == "" {
		writeJSON(w, http.StatusBadRequest, model.APIResp{Code: 400, Message: "回复内容不能为空"})
		return
	}

	reply := &model.Reply{
		PostID:        postID,
		AuthorName:    authorName,
		AuthorContact: authorContact,
		Content:       req.Content,
	}
	if err := h.store.CreateReply(reply); err != nil {
		writeJSON(w, http.StatusInternalServerError, model.APIResp{Code: 500, Message: err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, model.APIResp{Code: 201, Message: "回复成功", Data: reply})
}
