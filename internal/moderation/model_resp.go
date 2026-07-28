package moderation

import (
	"strconv"
	"time"

	"github.com/Milchstrassse/Ecampus-go/internal/platform/pagination"
)

type ReportResponse struct {
	ID          string `json:"id"`
	TargetType  string `json:"targetType"`
	TargetID    string `json:"targetId"`
	Reason      string `json:"reason"`
	Description string `json:"description"`
	Status      string `json:"status"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

type PunishmentResponse struct {
	ID           string  `json:"id"`
	Capability   string  `json:"capability"`
	Reason       string  `json:"reason"`
	Status       string  `json:"status"`
	StartsAt     string  `json:"startsAt"`
	EndsAt       *string `json:"endsAt"`
	RevokeReason string  `json:"revokeReason,omitempty"`
}

type AppealResponse struct {
	ID           string `json:"id"`
	PunishmentID string `json:"punishmentId"`
	Reason       string `json:"reason"`
	Status       string `json:"status"`
	Resolution   string `json:"resolution,omitempty"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
}

func reportPage(list []Report, total int64, page, size int) *pagination.PageResult[ReportResponse] {
	items := make([]ReportResponse, 0, len(list))
	for _, item := range list {
		items = append(items, ReportResponse{ID: id(item.ID), TargetType: item.TargetType, TargetID: item.TargetID, Reason: item.Reason, Description: item.Description, Status: item.Status, CreatedAt: formatDateTime(item.CreatedAt), UpdatedAt: formatDateTime(item.UpdatedAt)})
	}
	return pagination.NewPageResult(items, total, page, size)
}

func punishmentPage(list []Punishment, total int64, page, size int) *pagination.PageResult[PunishmentResponse] {
	items := make([]PunishmentResponse, 0, len(list))
	for _, item := range list {
		var endsAt *string
		if item.EndsAt != nil {
			value := formatDateTime(*item.EndsAt)
			endsAt = &value
		}
		items = append(items, PunishmentResponse{ID: id(item.ID), Capability: item.Capability, Reason: item.Reason, Status: item.Status, StartsAt: formatDateTime(item.StartsAt), EndsAt: endsAt, RevokeReason: item.RevokeReason})
	}
	return pagination.NewPageResult(items, total, page, size)
}

func appealPage(list []Appeal, total int64, page, size int) *pagination.PageResult[AppealResponse] {
	items := make([]AppealResponse, 0, len(list))
	for _, item := range list {
		items = append(items, AppealResponse{ID: id(item.ID), PunishmentID: id(item.PunishmentID), Reason: item.Reason, Status: item.Status, Resolution: item.Resolution, CreatedAt: formatDateTime(item.CreatedAt), UpdatedAt: formatDateTime(item.UpdatedAt)})
	}
	return pagination.NewPageResult(items, total, page, size)
}

func id(value int64) string { return strconv.FormatInt(value, 10) }

func formatDateTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.FixedZone("Asia/Shanghai", 8*60*60)
	}
	return value.In(loc).Format(time.RFC3339)
}
