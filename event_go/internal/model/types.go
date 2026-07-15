package model

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/qw2261/soulmarker/event_go/internal/api"
)

const TimeFormat = time.RFC3339

const (
	IdentityStatusLegacy     = "legacy"
	IdentityStatusBackfilled = "backfilled"
	IdentityStatusVerified   = "verified"
)

var (
	ErrNotFound               = errors.New("活动不存在")
	ErrDuplicate              = errors.New("该联系方式已报名本活动")
	ErrFull                   = errors.New("活动报名已满")
	ErrNotRegistered          = errors.New("未报名该活动，无法参与讨论")
	ErrNotParticipant         = errors.New("只有报名者才能发帖或回复")
	ErrTicketNotFound         = errors.New("门票不存在")
	ErrTicketSoldOut          = errors.New("门票已售罄")
	ErrUnauthorized           = errors.New("认证失败，请提供有效的管理员令牌")
	ErrCancelDeadlineExceeded = errors.New("已过取消截止时间，无法取消报名")
	ErrUserExists             = errors.New("该联系方式已注册")
	ErrInvalidCreds           = errors.New("联系方式或密码错误")
	ErrOrganizerNotFound      = errors.New("门店不存在")
	ErrAdmissionNotFound      = errors.New("入场凭证不存在")
	ErrAdmissionRevoked       = errors.New("入场凭证已失效")
	ErrAdmissionCheckedIn     = errors.New("入场凭证已核销")
	ErrEventHasAdmissions     = errors.New("活动已有入场凭证，不能删除")
	ErrPasswordResetInvalid   = errors.New("密码重置链接无效或已过期")
	ErrPasswordResetRateLimit = errors.New("密码重置请求过于频繁")
)

type Organizer struct {
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

type UpdateOrganizerReq struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Contact     *string `json:"contact"`
	LogoURL     *string `json:"logo_url"`
	Address     *string `json:"address"`
	Website     *string `json:"website"`
	Tags        *string `json:"tags"`
}

type Event struct {
	ID            int64     `json:"id"`
	OrganizerID   int64     `json:"organizer_id"`
	OrganizerName string    `json:"organizer_name,omitempty"`
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	EventTime     string    `json:"event_time"`
	Location      string    `json:"location"`
	Capacity      int       `json:"capacity"`
	Price         float64   `json:"price"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type Registration struct {
	ID             int64      `json:"id"`
	EventID        int64      `json:"event_id"`
	UserID         *int64     `json:"-"`
	Name           string     `json:"name"`
	Contact        string     `json:"contact"`
	TicketID       *int64     `json:"ticket_id,omitempty"`
	TicketName     string     `json:"ticket_name,omitempty"`
	IdentityStatus string     `json:"identity_status,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	Admission      *Admission `json:"-"`
}

const (
	AdmissionStatusActive     = "active"
	AdmissionStatusRevoked    = "revoked"
	AdmissionCredentialPrefix = "soulmark:admission:"
)

type Admission struct {
	ID             int64
	RegistrationID *int64
	EventID        int64
	UserID         int64
	TicketName     string
	CredentialCode string
	Status         string
	IssuedAt       time.Time
	RevokedAt      *time.Time
	CheckedInAt    *time.Time
}

type MyAdmission struct {
	Admission
	EventTitle  string
	EventTime   string
	Location    string
	EventStatus string
}

const (
	ActivityKindAdmission    = "admission"
	ActivityKindRegistration = "registration"
)

type MyActivity struct {
	ID             int64
	Kind           string
	RegistrationID *int64
	EventID        int64
	EventTitle     string
	EventTime      string
	Location       string
	EventStatus    string
	TicketID       *int64
	TicketName     string
	JoinedAt       time.Time
	Admission      *Admission
}

type Checkin struct {
	ID             int64
	AdmissionID    int64
	EventID        int64
	CredentialCode string
	UserName       string
	UserContact    string
	CheckedInAt    time.Time
	CheckedInBy    string
}

type UpdateEventReq struct {
	OrganizerID *int64   `json:"organizer_id"`
	Title       *string  `json:"title"`
	Description *string  `json:"description"`
	EventTime   *string  `json:"event_time"`
	Location    *string  `json:"location"`
	Capacity    *int     `json:"capacity"`
	Price       *float64 `json:"price"`
	Status      *string  `json:"status"`
}

type User struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	Contact      string    `json:"contact"`
	PasswordHash string    `json:"-"`
	AuthVersion  int       `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

type UserClaims struct {
	UserID      int64  `json:"user_id"`
	Name        string `json:"name"`
	Contact     string `json:"contact"`
	AuthVersion int    `json:"auth_version"`
	jwt.RegisteredClaims
}

type contextKey string

const UserContextKey contextKey = "user"

type ListEventsParams struct {
	Status      string
	PriceType   string
	Keyword     string
	OrganizerID int64
	Offset      int
	Limit       int
}

// APIResp 保留为测试与历史内部调用的兼容别名；HTTP 层使用 internal/api.Response。
type APIResp = api.Response

type MyRegistration struct {
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

type Post struct {
	ID             int64     `json:"id"`
	EventID        int64     `json:"event_id"`
	UserID         *int64    `json:"-"`
	AuthorName     string    `json:"author_name"`
	AuthorContact  string    `json:"-"`
	IdentityStatus string    `json:"-"`
	Title          string    `json:"title"`
	Content        string    `json:"content"`
	ReplyCount     int       `json:"reply_count"`
	CreatedAt      time.Time `json:"created_at"`
}

type Reply struct {
	ID             int64     `json:"id"`
	PostID         int64     `json:"post_id"`
	UserID         *int64    `json:"-"`
	AuthorName     string    `json:"author_name"`
	AuthorContact  string    `json:"-"`
	IdentityStatus string    `json:"-"`
	Content        string    `json:"content"`
	CreatedAt      time.Time `json:"created_at"`
}

type IdentityEntityStats struct {
	Total      int `json:"total"`
	Verified   int `json:"verified"`
	Backfilled int `json:"backfilled"`
	Legacy     int `json:"legacy"`
}

type LegacyIdentityRecord struct {
	EntityType string `json:"entity_type"`
	ID         int64  `json:"id"`
	ParentID   int64  `json:"parent_id"`
	Name       string `json:"name"`
	Contact    string `json:"contact"`
}

type IdentityMigrationReport struct {
	Registrations IdentityEntityStats    `json:"registrations"`
	Posts         IdentityEntityStats    `json:"posts"`
	Replies       IdentityEntityStats    `json:"replies"`
	LegacyRecords []LegacyIdentityRecord `json:"legacy_records"`
}

func UserFromContext(ctx context.Context) (*UserClaims, bool) {
	claims, ok := ctx.Value(UserContextKey).(*UserClaims)
	return claims, ok
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

type UpdateTicketReq struct {
	Name  *string  `json:"name"`
	Price *float64 `json:"price"`
	Stock *int     `json:"stock"`
}
