package dto

import (
	"time"

	"github.com/qw2261/soulmarker/event_go/internal/api"
)

type Response = api.Response

type RegisterUserRequest struct {
	Name     string `json:"name"`
	Contact  string `json:"contact"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Contact  string `json:"contact"`
	Password string `json:"password"`
}

type PasswordResetRequest struct {
	Contact string `json:"contact"`
}

type PasswordResetConfirmRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

type CreateOrganizerRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Contact     string `json:"contact"`
	LogoURL     string `json:"logo_url"`
	Address     string `json:"address"`
	Website     string `json:"website"`
	Tags        string `json:"tags"`
}

type UpdateOrganizerRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Contact     *string `json:"contact"`
	LogoURL     *string `json:"logo_url"`
	Address     *string `json:"address"`
	Website     *string `json:"website"`
	Tags        *string `json:"tags"`
}

type CreateEventRequest struct {
	OrganizerID int64   `json:"organizer_id"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	CoverURL    string  `json:"cover_url"`
	EventTime   string  `json:"event_time"`
	Location    string  `json:"location"`
	Capacity    int     `json:"capacity"`
	Price       float64 `json:"price"`
}

type UpdateEventRequest struct {
	OrganizerID *int64   `json:"organizer_id"`
	Title       *string  `json:"title"`
	Description *string  `json:"description"`
	CoverURL    *string  `json:"cover_url"`
	EventTime   *string  `json:"event_time"`
	Location    *string  `json:"location"`
	Capacity    *int     `json:"capacity"`
	Price       *float64 `json:"price"`
	Status      *string  `json:"status"`
}

type CreateTicketRequest struct {
	Name  string  `json:"name"`
	Price float64 `json:"price"`
	Stock int     `json:"stock"`
}

type UpdateTicketRequest struct {
	Name  *string  `json:"name"`
	Price *float64 `json:"price"`
	Stock *int     `json:"stock"`
}

type RegisterEventRequest struct {
	TicketID *int64 `json:"ticket_id,omitempty"`
}

type CreatePostRequest struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

type CreateReplyRequest struct {
	Content string `json:"content"`
}

type UserResponse struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Contact   string    `json:"contact"`
	CreatedAt time.Time `json:"created_at"`
}

type LoginResponse struct {
	Token string       `json:"token"`
	User  UserResponse `json:"user"`
}

type AdminSessionResponse struct {
	Authenticated bool `json:"authenticated"`
}

type OrganizerResponse struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Contact     string    `json:"contact"`
	LogoURL     string    `json:"logo_url"`
	Address     string    `json:"address"`
	Website     string    `json:"website"`
	Tags        string    `json:"tags"`
	EventCount  int       `json:"event_count,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type EventResponse struct {
	ID            int64     `json:"id"`
	OrganizerID   int64     `json:"organizer_id"`
	OrganizerName string    `json:"organizer_name,omitempty"`
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	CoverURL      string    `json:"cover_url"`
	EventTime     string    `json:"event_time"`
	Location      string    `json:"location"`
	Capacity      int       `json:"capacity"`
	Price         float64   `json:"price"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type TicketResponse struct {
	ID        int64     `json:"id"`
	EventID   int64     `json:"event_id"`
	Name      string    `json:"name"`
	Price     float64   `json:"price"`
	Stock     int       `json:"stock"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type RegistrationResponse struct {
	ID             int64              `json:"id"`
	EventID        int64              `json:"event_id"`
	Name           string             `json:"name"`
	Contact        string             `json:"contact"`
	TicketID       *int64             `json:"ticket_id,omitempty"`
	TicketName     string             `json:"ticket_name,omitempty"`
	IdentityStatus string             `json:"identity_status,omitempty"`
	CreatedAt      time.Time          `json:"created_at"`
	Admission      *AdmissionResponse `json:"admission,omitempty"`
}

type AdmissionResponse struct {
	ID             int64      `json:"id"`
	EventID        int64      `json:"event_id"`
	TicketName     string     `json:"ticket_name,omitempty"`
	CredentialCode string     `json:"credential_code"`
	Credential     string     `json:"credential"`
	Status         string     `json:"status"`
	IssuedAt       time.Time  `json:"issued_at"`
	RevokedAt      *time.Time `json:"revoked_at,omitempty"`
	CheckedInAt    *time.Time `json:"checked_in_at,omitempty"`
}

type MyAdmissionResponse struct {
	AdmissionResponse
	EventTitle  string `json:"event_title"`
	EventTime   string `json:"event_time"`
	Location    string `json:"location"`
	EventStatus string `json:"event_status"`
}

type MyActivityResponse struct {
	ID             int64              `json:"id"`
	Kind           string             `json:"kind"`
	RegistrationID *int64             `json:"registration_id,omitempty"`
	EventID        int64              `json:"event_id"`
	EventTitle     string             `json:"event_title"`
	EventTime      string             `json:"event_time"`
	Location       string             `json:"location"`
	EventStatus    string             `json:"event_status"`
	TicketID       *int64             `json:"ticket_id,omitempty"`
	TicketName     string             `json:"ticket_name,omitempty"`
	JoinedAt       time.Time          `json:"joined_at"`
	Admission      *AdmissionResponse `json:"admission,omitempty"`
}

type CheckinRequest struct {
	Credential string `json:"credential"`
}

type CheckinResponse struct {
	ID             int64     `json:"id"`
	AdmissionID    int64     `json:"admission_id"`
	EventID        int64     `json:"event_id"`
	CredentialCode string    `json:"credential_code,omitempty"`
	UserName       string    `json:"user_name,omitempty"`
	UserContact    string    `json:"user_contact,omitempty"`
	CheckedInAt    time.Time `json:"checked_in_at"`
	CheckedInBy    string    `json:"checked_in_by"`
}

type CheckinResultResponse struct {
	Checkin          CheckinResponse `json:"checkin"`
	AlreadyCheckedIn bool            `json:"already_checked_in"`
}

type RegistrationStatusResponse struct {
	Registered bool               `json:"registered"`
	Admission  *AdmissionResponse `json:"admission,omitempty"`
}

type MyRegistrationResponse struct {
	ID          int64     `json:"id"`
	EventID     int64     `json:"event_id"`
	EventTitle  string    `json:"event_title"`
	EventTime   string    `json:"event_time"`
	Location    string    `json:"location"`
	EventStatus string    `json:"event_status"`
	TicketID    *int64    `json:"ticket_id,omitempty"`
	TicketName  string    `json:"ticket_name,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type PostResponse struct {
	ID         int64     `json:"id"`
	EventID    int64     `json:"event_id"`
	AuthorName string    `json:"author_name"`
	Title      string    `json:"title"`
	Content    string    `json:"content"`
	ReplyCount int       `json:"reply_count"`
	CreatedAt  time.Time `json:"created_at"`
}

type ReplyResponse struct {
	ID         int64     `json:"id"`
	PostID     int64     `json:"post_id"`
	AuthorName string    `json:"author_name"`
	Content    string    `json:"content"`
	CreatedAt  time.Time `json:"created_at"`
}

type PostDetailResponse struct {
	Post    PostResponse    `json:"post"`
	Replies []ReplyResponse `json:"replies"`
}

type IdentityEntityStatsResponse struct {
	Total      int `json:"total"`
	Verified   int `json:"verified"`
	Backfilled int `json:"backfilled"`
	Legacy     int `json:"legacy"`
}

type LegacyIdentityRecordResponse struct {
	EntityType string `json:"entity_type"`
	ID         int64  `json:"id"`
	ParentID   int64  `json:"parent_id"`
	Name       string `json:"name"`
	Contact    string `json:"contact"`
}

type IdentityMigrationReportResponse struct {
	Registrations IdentityEntityStatsResponse    `json:"registrations"`
	Posts         IdentityEntityStatsResponse    `json:"posts"`
	Replies       IdentityEntityStatsResponse    `json:"replies"`
	LegacyRecords []LegacyIdentityRecordResponse `json:"legacy_records"`
}

type HealthResponse struct {
	Status        string `json:"status"`
	Version       string `json:"version"`
	UptimeSeconds int64  `json:"uptime_seconds"`
	DB            string `json:"db"`
}
