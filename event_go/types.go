package main

import (
	"errors"
	"time"
)

const (
	port         = "8080"
	timeFormat   = time.RFC3339
	timeParseMsg = "格式错误，请使用 RFC3339 格式，例如：2026-12-31T18:00:00+08:00"
	defaultDB    = "data/event_go.db"
)

var (
	ErrNotFound  = errors.New("活动不存在")
	ErrDuplicate = errors.New("该联系方式已报名本活动")
	ErrFull      = errors.New("活动报名已满")
)

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
	ID        int64     `json:"id"`
	EventID   int64     `json:"event_id"`
	Name      string    `json:"name"`
	Contact   string    `json:"contact"`
	CreatedAt time.Time `json:"created_at"`
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
	Name    string `json:"name"`
	Contact string `json:"contact"`
}

type APIResp struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}
