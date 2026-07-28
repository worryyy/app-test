package reservation

type VenueReq struct {
	Name                string `json:"name" binding:"required,max=128"`
	Description         string `json:"description" binding:"max=2000"`
	AdvanceDays         int    `json:"advanceDays"`
	SlotMinutes         int    `json:"slotMinutes"`
	DailyLimit          int    `json:"dailyLimit"`
	CancelBeforeMinutes int    `json:"cancelBeforeMinutes"`
}
type ResourceReq struct {
	Name     string `json:"name" binding:"required,max=128"`
	Capacity int    `json:"capacity" binding:"min=1"`
}
type RuleReq struct {
	Weekday     int `json:"weekday" binding:"required,min=1,max=7"`
	StartMinute int `json:"startMinute" binding:"min=0,max=1439"`
	EndMinute   int `json:"endMinute" binding:"required,min=1,max=1440"`
}
type ClosureReq struct {
	VenueID    string `json:"venueId"`
	ResourceID string `json:"resourceId"`
	StartAt    string `json:"startAt" binding:"required"`
	EndAt      string `json:"endAt" binding:"required"`
	Reason     string `json:"reason" binding:"required,max=255"`
}
type CreateBookingReq struct {
	SlotID string `json:"slotId" binding:"required"`
}
type CheckinReq struct {
	Code string `json:"code" binding:"required"`
}
