package reservation

import (
	"strconv"
	"time"
)

type VenueResponse struct {
	ID                  string `json:"id"`
	Name                string `json:"name"`
	Description         string `json:"description"`
	AdvanceDays         int    `json:"advanceDays"`
	SlotMinutes         int    `json:"slotMinutes"`
	DailyLimit          int    `json:"dailyLimit"`
	CancelBeforeMinutes int    `json:"cancelBeforeMinutes"`
}
type ResourceResponse struct {
	ID       string `json:"id"`
	VenueID  string `json:"venueId"`
	Name     string `json:"name"`
	Capacity int    `json:"capacity"`
}
type SlotResponse struct {
	ID        string `json:"id"`
	StartAt   string `json:"startAt"`
	EndAt     string `json:"endAt"`
	Capacity  int    `json:"capacity"`
	Available int    `json:"available"`
	Status    string `json:"status"`
}
type BookingResponse struct {
	ID          string       `json:"id"`
	Status      string       `json:"status"`
	CheckinCode string       `json:"checkinCode"`
	Slot        SlotResponse `json:"slot"`
	CreatedAt   string       `json:"createdAt"`
}

func venueResponse(v Venue) VenueResponse {
	return VenueResponse{ID: id(v.ID), Name: v.Name, Description: v.Description, AdvanceDays: v.AdvanceDays, SlotMinutes: v.SlotMinutes, DailyLimit: v.DailyLimit, CancelBeforeMinutes: v.CancelBeforeMinutes}
}
func resourceResponse(v Resource) ResourceResponse {
	return ResourceResponse{ID: id(v.ID), VenueID: id(v.VenueID), Name: v.Name, Capacity: v.Capacity}
}
func slotResponse(v Slot) SlotResponse {
	available := v.Capacity - v.ReservedCount
	if available < 0 {
		available = 0
	}
	return SlotResponse{ID: id(v.ID), StartAt: format(v.StartAt), EndAt: format(v.EndAt), Capacity: v.Capacity, Available: available, Status: v.Status}
}
func bookingResponse(v Booking) BookingResponse {
	return BookingResponse{ID: id(v.ID), Status: v.Status, CheckinCode: v.CheckinCode, Slot: slotResponse(v.Slot), CreatedAt: format(v.CreatedAt)}
}
func id(v int64) string { return strconv.FormatInt(v, 10) }
func format(v time.Time) string {
	if v.IsZero() {
		return ""
	}
	loc, e := time.LoadLocation("Asia/Shanghai")
	if e != nil {
		loc = time.FixedZone("Asia/Shanghai", 8*60*60)
	}
	return v.In(loc).Format(time.RFC3339)
}
