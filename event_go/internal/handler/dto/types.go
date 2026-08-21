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

type RecoveryEmailRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RecoveryEmailConfirmRequest struct {
	Token string `json:"token"`
}

type CreateOrganizationRequest struct {
	Name               string `json:"name"`
	Slug               string `json:"slug"`
	ProfileName        string `json:"profile_name"`
	ProfileDescription string `json:"profile_description"`
	ProfileContact     string `json:"profile_contact"`
	ProfileLogoURL     string `json:"profile_logo_url"`
	ProfileAddress     string `json:"profile_address"`
	ProfileWebsite     string `json:"profile_website"`
	ProfileTags        string `json:"profile_tags"`
}

type CreateOrganizationInvitationRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

type AcceptOrganizationInvitationRequest struct {
	Token string `json:"token"`
}

type UpdateOrganizationMemberRequest struct {
	Role string `json:"role"`
}

type TransferOrganizationOwnerRequest struct {
	MemberID int64 `json:"member_id"`
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

type CreateContentReportRequest struct {
	Category string `json:"category"`
	Detail   string `json:"detail"`
}

type ResolveContentReportRequest struct {
	Resolution string `json:"resolution"`
	Note       string `json:"note"`
}

type ModerateContentRequest struct {
	Reason string `json:"reason"`
}

type UserResponse struct {
	ID                      int64      `json:"id"`
	Name                    string     `json:"name"`
	Contact                 string     `json:"contact"`
	RecoveryEmail           string     `json:"recovery_email,omitempty"`
	RecoveryEmailVerifiedAt *time.Time `json:"recovery_email_verified_at,omitempty"`
	CreatedAt               time.Time  `json:"created_at"`
}

// DataSubjectRequestResponse 是数据主体（隐私）请求的响应视图。
type DataSubjectRequestResponse struct {
	ID          int64      `json:"id"`
	UserID      int64      `json:"user_id"`
	RequestType string     `json:"request_type"`
	Status      string     `json:"status"`
	RequestedAt time.Time  `json:"requested_at"`
	ProcessedAt *time.Time `json:"processed_at,omitempty"`
	Resolution  string     `json:"resolution"`
	CreatedAt   time.Time  `json:"created_at"`
}

type NotificationResponse struct {
	ID        int64      `json:"id"`
	EventID   *int64     `json:"event_id,omitempty"`
	Type      string     `json:"type"`
	Title     string     `json:"title"`
	Body      string     `json:"body"`
	ActionURL string     `json:"action_url"`
	ReadAt    *time.Time `json:"read_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

type NotificationUnreadCountResponse struct {
	Unread int `json:"unread"`
}

type NotificationsMarkedReadResponse struct {
	Updated int `json:"updated"`
}

type LoginResponse struct {
	Token string       `json:"token"`
	User  UserResponse `json:"user"`
}

type AdminSessionResponse struct {
	Authenticated bool   `json:"authenticated"`
	PrincipalType string `json:"principal_type"`
}

type OrganizationContextResponse struct {
	OrganizationID     int64    `json:"organization_id"`
	OrganizationName   string   `json:"organization_name"`
	OrganizationSlug   string   `json:"organization_slug"`
	OrganizationStatus string   `json:"organization_status"`
	MembershipStatus   string   `json:"membership_status"`
	Role               string   `json:"role"`
	PrincipalType      string   `json:"principal_type"`
	Capabilities       []string `json:"capabilities"`
}

type OrganizationWorkspaceResponse struct {
	ID        int64             `json:"id"`
	Name      string            `json:"name"`
	Slug      string            `json:"slug"`
	Status    string            `json:"status"`
	Profile   OrganizerResponse `json:"profile"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

type OrganizationMemberResponse struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Name      string    `json:"name"`
	Contact   string    `json:"contact"`
	Role      string    `json:"role"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type OrganizationInvitationResponse struct {
	ID              int64      `json:"id"`
	Email           string     `json:"email"`
	Role            string     `json:"role"`
	Status          string     `json:"status"`
	ExpiresAt       time.Time  `json:"expires_at"`
	AcceptedAt      *time.Time `json:"accepted_at,omitempty"`
	RevokedAt       *time.Time `json:"revoked_at,omitempty"`
	InvitedByUserID int64      `json:"invited_by_user_id"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type OrganizationAuditResponse struct {
	ID             int64     `json:"id"`
	OrganizationID int64     `json:"organization_id"`
	ActorType      string    `json:"actor_type"`
	ActorID        *int64    `json:"actor_id,omitempty"`
	Action         string    `json:"action"`
	ResourceType   string    `json:"resource_type"`
	ResourceID     string    `json:"resource_id"`
	RequestID      string    `json:"request_id"`
	Outcome        string    `json:"outcome"`
	HTTPStatus     int       `json:"http_status"`
	CreatedAt      time.Time `json:"created_at"`
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

type ContentReportReceiptResponse struct {
	ID         int64     `json:"id"`
	TargetType string    `json:"target_type"`
	TargetID   int64     `json:"target_id"`
	Category   string    `json:"category"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
}

type ContentReportResponse struct {
	ID                     int64      `json:"id"`
	EventID                int64      `json:"event_id"`
	PostID                 int64      `json:"post_id"`
	TargetType             string     `json:"target_type"`
	TargetID               int64      `json:"target_id"`
	ReporterUserID         int64      `json:"reporter_user_id"`
	ReporterName           string     `json:"reporter_name"`
	Category               string     `json:"category"`
	Detail                 string     `json:"detail"`
	Status                 string     `json:"status"`
	CreatedAt              time.Time  `json:"created_at"`
	ResolvedAt             *time.Time `json:"resolved_at,omitempty"`
	ResolvedBy             string     `json:"resolved_by"`
	ResolutionNote         string     `json:"resolution_note"`
	TargetAuthorName       string     `json:"target_author_name"`
	TargetTitle            string     `json:"target_title"`
	TargetContent          string     `json:"target_content"`
	TargetModerationStatus string     `json:"target_moderation_status"`
}

type ContentModerationActionResponse struct {
	ID         int64     `json:"id"`
	ReportID   *int64    `json:"report_id,omitempty"`
	EventID    int64     `json:"event_id"`
	PostID     int64     `json:"post_id"`
	TargetType string    `json:"target_type"`
	TargetID   int64     `json:"target_id"`
	Action     string    `json:"action"`
	Actor      string    `json:"actor"`
	Reason     string    `json:"reason"`
	CreatedAt  time.Time `json:"created_at"`
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

// LivenessResponse 是 liveness 探针的响应，仅表示进程存活，不探测依赖。
type LivenessResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

// ReadinessResponse 是 readiness 探针的响应，反映依赖（数据库）是否就绪。
type ReadinessResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
	DB      string `json:"db"`
}
