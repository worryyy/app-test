package moderation

import "time"

const (
	ReportPending     = "pending"
	ReportReviewing   = "reviewing"
	ReportRejected    = "rejected"
	ReportActioned    = "actioned"
	ReportWithdrawn   = "withdrawn"
	AppealPending     = "pending"
	AppealApproved    = "approved"
	AppealRejected    = "rejected"
	PunishmentActive  = "active"
	PunishmentRevoked = "revoked"

	CapabilityContent     = "content"
	CapabilityTrade       = "trade"
	CapabilityReservation = "reservation"
	CapabilityAccount     = "account"
)

var targetTypes = map[string]struct{}{
	"user": {}, "topic": {}, "comment": {}, "chatMessage": {},
	"courseReview": {}, "material": {}, "marketplaceItem": {},
}

var capabilities = map[string]struct{}{
	CapabilityContent: {}, CapabilityTrade: {}, CapabilityReservation: {}, CapabilityAccount: {},
}

type Report struct {
	ID                 int64      `gorm:"column:id;primaryKey" json:"-"`
	ReporterRootUserID int64      `gorm:"column:reporter_root_user_id" json:"-"`
	ReporterUserID     int64      `gorm:"column:reporter_user_id" json:"-"`
	TargetType         string     `gorm:"column:target_type" json:"targetType"`
	TargetID           string     `gorm:"column:target_id" json:"targetId"`
	TargetRootUserID   int64      `gorm:"column:target_root_user_id" json:"-"`
	Reason             string     `gorm:"column:reason" json:"reason"`
	Description        string     `gorm:"column:description" json:"description"`
	Status             string     `gorm:"column:status" json:"status"`
	AssigneeAdminID    *int64     `gorm:"column:assignee_admin_id" json:"-"`
	CreatedAt          time.Time  `gorm:"column:created_at" json:"-"`
	UpdatedAt          time.Time  `gorm:"column:updated_at" json:"-"`
	WithdrawnAt        *time.Time `gorm:"column:withdrawn_at" json:"-"`
}

func (Report) TableName() string { return "moderation_reports" }

type ReportSnapshot struct {
	ID        int64     `gorm:"column:id;primaryKey"`
	ReportID  int64     `gorm:"column:report_id;uniqueIndex"`
	Payload   []byte    `gorm:"column:payload;type:json"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func (ReportSnapshot) TableName() string { return "moderation_report_snapshots" }

type Punishment struct {
	ID           int64      `gorm:"column:id;primaryKey" json:"-"`
	RootUserID   int64      `gorm:"column:root_user_id" json:"-"`
	ReportID     *int64     `gorm:"column:report_id" json:"-"`
	Capability   string     `gorm:"column:capability" json:"capability"`
	Reason       string     `gorm:"column:reason" json:"reason"`
	Status       string     `gorm:"column:status" json:"status"`
	StartsAt     time.Time  `gorm:"column:starts_at" json:"-"`
	EndsAt       *time.Time `gorm:"column:ends_at" json:"-"`
	CreatedBy    int64      `gorm:"column:created_by" json:"-"`
	CreatedAt    time.Time  `gorm:"column:created_at" json:"-"`
	RevokedBy    *int64     `gorm:"column:revoked_by" json:"-"`
	RevokedAt    *time.Time `gorm:"column:revoked_at" json:"-"`
	RevokeReason string     `gorm:"column:revoke_reason" json:"revokeReason,omitempty"`
}

func (Punishment) TableName() string { return "moderation_punishments" }

type Appeal struct {
	ID           int64     `gorm:"column:id;primaryKey" json:"-"`
	PunishmentID int64     `gorm:"column:punishment_id;uniqueIndex" json:"-"`
	RootUserID   int64     `gorm:"column:root_user_id" json:"-"`
	Reason       string    `gorm:"column:reason" json:"reason"`
	Status       string    `gorm:"column:status" json:"status"`
	Resolution   string    `gorm:"column:resolution" json:"resolution,omitempty"`
	ReviewedBy   *int64    `gorm:"column:reviewed_by" json:"-"`
	CreatedAt    time.Time `gorm:"column:created_at" json:"-"`
	UpdatedAt    time.Time `gorm:"column:updated_at" json:"-"`
}

func (Appeal) TableName() string { return "moderation_appeals" }

type AuditLog struct {
	ID           int64          `gorm:"column:id;primaryKey"`
	ReportID     *int64         `gorm:"column:report_id"`
	PunishmentID *int64         `gorm:"column:punishment_id"`
	AppealID     *int64         `gorm:"column:appeal_id"`
	AdminID      int64          `gorm:"column:admin_id"`
	Action       string         `gorm:"column:action"`
	Detail       map[string]any `gorm:"column:detail;serializer:json"`
	CreatedAt    time.Time      `gorm:"column:created_at"`
}

func (AuditLog) TableName() string { return "moderation_audit_logs" }
