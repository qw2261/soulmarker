package handler

import (
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
	if !decodeJSON(w, r, &req) {
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
		writeInternalError(w, "create_post", err)
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

	post, ok := h.getPostForEventOr404(w, eventID, postID)
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

	post, ok := h.getPostForEventOr404(w, eventID, postID)
	if !ok {
		return
	}

	var req model.CreateReplyReq
	if !decodeJSON(w, r, &req) {
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
		writeInternalError(w, "create_reply", err)
		return
	}

	writeJSON(w, http.StatusCreated, model.APIResp{Code: 201, Message: "回复成功", Data: reply})
}
