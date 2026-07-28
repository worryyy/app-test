package moderation

type CreateReportReq struct {
	TargetType  string `json:"targetType" binding:"required"`
	TargetID    string `json:"targetId" binding:"required"`
	Reason      string `json:"reason" binding:"required,max=64"`
	Description string `json:"description" binding:"max=2000"`
}

type CreateAppealReq struct {
	Reason string `json:"reason" binding:"required,max=2000"`
}

type AdminDecisionReq struct {
	Action          string `json:"action" binding:"required,oneof=reject hide warn punish"`
	Reason          string `json:"reason" binding:"required,max=255"`
	Capability      string `json:"capability"`
	DurationMinutes *int64 `json:"durationMinutes"`
}

type RevokePunishmentReq struct {
	Reason string `json:"reason" binding:"required,max=255"`
}

type AppealDecisionReq struct {
	Approved   bool   `json:"approved"`
	Resolution string `json:"resolution" binding:"required,max=255"`
}
