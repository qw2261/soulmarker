package main

import (
	"errors"
	"os"
	"time"
)

const (
	defaultPort  = "8080"
	timeFormat   = time.RFC3339
	timeParseMsg = "格式错误，请使用 RFC3339 格式，例如：2026-12-31T18:00:00+08:00"
	defaultDB    = "data/event_go.db"
)

func getPort() string {
	if p := os.Getenv("PORT"); p != "" {
		return p
	}
	return defaultPort
}

var (
	ErrNotFound       = errors.New("活动不存在")
	ErrDuplicate      = errors.New("该联系方式已报名本活动")
	ErrFull           = errors.New("活动报名已满")
	ErrNotRegistered  = errors.New("未报名该活动，无法参与讨论")
	ErrNotParticipant = errors.New("只有报名者才能发帖或回复")
	ErrTicketNotFound = errors.New("门票不存在")
	ErrTicketSoldOut  = errors.New("门票已售罄")
	ErrUnauthorized   = errors.New("认证失败，请提供有效的管理员令牌")
)

func getAdminToken() string {
	return os.Getenv("ADMIN_TOKEN")
}

type Event struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	EventTime   string    `json:"event_time"`
	Location    string    `json:"location"`
	Capacity    int       `json:"capacity"`
	Price       float64   `json:"price"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Registration struct {
	ID         int64     `json:"id"`
	EventID    int64     `json:"event_id"`
	Name       string    `json:"name"`
	Contact    string    `json:"contact"`
	TicketID   *int64    `json:"ticket_id,omitempty"`
	TicketName string    `json:"ticket_name,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

type CreateEventReq struct {
	Title       string  `json:"title"`
	Description string  `json:"description"`
	EventTime   string  `json:"event_time"`
	Location    string  `json:"location"`
	Capacity    int     `json:"capacity"`
	Price       float64 `json:"price"`
}

type UpdateEventReq struct {
	Title       *string  `json:"title"`
	Description *string  `json:"description"`
	EventTime   *string  `json:"event_time"`
	Location    *string  `json:"location"`
	Capacity    *int     `json:"capacity"`
	Price       *float64 `json:"price"`
	Status      *string  `json:"status"`
}

type RegisterReq struct {
	Name     string `json:"name"`
	Contact  string `json:"contact"`
	TicketID *int64 `json:"ticket_id,omitempty"`
}

type APIResp struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type Post struct {
	ID            int64     `json:"id"`
	EventID       int64     `json:"event_id"`
	AuthorName    string    `json:"author_name"`
	AuthorContact string    `json:"-"`
	Title         string    `json:"title"`
	Content       string    `json:"content"`
	ReplyCount    int       `json:"reply_count"`
	CreatedAt     time.Time `json:"created_at"`
}

type Reply struct {
	ID            int64     `json:"id"`
	PostID        int64     `json:"post_id"`
	AuthorName    string    `json:"author_name"`
	AuthorContact string    `json:"-"`
	Content       string    `json:"content"`
	CreatedAt     time.Time `json:"created_at"`
}

type CreatePostReq struct {
	AuthorName    string `json:"author_name"`
	AuthorContact string `json:"author_contact"`
	Title         string `json:"title"`
	Content       string `json:"content"`
}

type CreateReplyReq struct {
	AuthorName    string `json:"author_name"`
	AuthorContact string `json:"author_contact"`
	Content       string `json:"content"`
}

type Ticket struct {
	ID        int64     `json:"id"`
	EventID   int64     `json:"event_id"`
	Name      string    `json:"name"`
	Price     float64   `json:"price"`
	Stock     int       `json:"stock"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateTicketReq struct {
	Name  string  `json:"name"`
	Price float64 `json:"price"`
	Stock int     `json:"stock"`
}

type UpdateTicketReq struct {
	Name  *string  `json:"name"`
	Price *float64 `json:"price"`
	Stock *int     `json:"stock"`
}
