package reservation

import (
	"context"
	"errors"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/bizerr"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/pagination"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/snowflake"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"strconv"
	"strings"
	"time"
)

var (
	ErrVenueNotFound    = bizerr.NotFound("场馆不存在")
	ErrResourceNotFound = bizerr.NotFound("场地资源不存在")
	ErrSlotUnavailable  = bizerr.Biz("时段不可预约")
	ErrBookingConflict  = bizerr.Biz("预约时间冲突")
	ErrDailyLimit       = bizerr.Biz("已达到每日预约上限")
	ErrBookingNotFound  = bizerr.NotFound("预约不存在")
	ErrCancelDeadline   = bizerr.Biz("已超过最晚取消时间")
	ErrCheckinInvalid   = bizerr.Biz("核销码无效或已使用")
)

type CapabilityChecker interface {
	CheckCapability(context.Context, int64, int64, string) error
}
type Notifier interface {
	NotifyReservation(context.Context, int64, string, string, string, string) error
}
type Service struct {
	repo         *Repository
	capabilities CapabilityChecker
	notifier     Notifier
	logger       *zap.Logger
	now          func() time.Time
	location     *time.Location
}

func NewService(db *gorm.DB, logger *zap.Logger) *Service {
	if logger == nil {
		logger = zap.NewNop()
	}
	loc, e := time.LoadLocation("Asia/Shanghai")
	if e != nil {
		loc = time.FixedZone("Asia/Shanghai", 8*60*60)
	}
	return &Service{repo: NewRepository(db), logger: logger, now: time.Now, location: loc}
}
func (s *Service) SetCapabilityChecker(v CapabilityChecker) { s.capabilities = v }
func (s *Service) SetNotifier(v Notifier)                   { s.notifier = v }

func (s *Service) ListVenues(ctx context.Context) ([]VenueResponse, error) {
	list, e := s.repo.ListVenues(ctx)
	if e != nil {
		return nil, bizerr.InternalWrap("查询场馆失败", e)
	}
	items := make([]VenueResponse, 0, len(list))
	for _, v := range list {
		items = append(items, venueResponse(v))
	}
	return items, nil
}
func (s *Service) ListResources(ctx context.Context, venueID string) ([]ResourceResponse, error) {
	id, e := parseID(venueID)
	if e != nil {
		return nil, e
	}
	list, e := s.repo.ListResources(ctx, id)
	if e != nil {
		return nil, bizerr.InternalWrap("查询场地资源失败", e)
	}
	items := make([]ResourceResponse, 0, len(list))
	for _, v := range list {
		items = append(items, resourceResponse(v))
	}
	return items, nil
}
func (s *Service) AvailableSlots(ctx context.Context, resourceID, dateRaw string) ([]SlotResponse, error) {
	id, e := parseID(resourceID)
	if e != nil {
		return nil, e
	}
	date, e := time.ParseInLocation("2006-01-02", strings.TrimSpace(dateRaw), s.location)
	if e != nil {
		return nil, bizerr.Param("日期格式错误")
	}
	resource, e := s.repo.FindResource(ctx, id)
	if e != nil || resource == nil || resource.Status != StatusActive {
		return nil, ErrResourceNotFound
	}
	venue, e := s.repo.FindVenue(ctx, resource.VenueID)
	if e != nil || venue == nil || venue.Status != StatusActive {
		return nil, ErrVenueNotFound
	}
	today := dayStart(s.now().In(s.location))
	if date.Before(today) || date.After(today.AddDate(0, 0, venue.AdvanceDays)) {
		return []SlotResponse{}, nil
	}
	weekday := int(date.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	rules, e := s.repo.ListRules(ctx, id, weekday)
	if e != nil {
		return nil, e
	}
	now := s.now()
	slots := []Slot{}
	for _, rule := range rules {
		for minute := rule.StartMinute; minute+venue.SlotMinutes <= rule.EndMinute; minute += venue.SlotMinutes {
			start := date.Add(time.Duration(minute) * time.Minute)
			end := start.Add(time.Duration(venue.SlotMinutes) * time.Minute)
			closed, e := s.repo.HasClosure(ctx, venue.ID, resource.ID, start, end)
			if e != nil {
				return nil, e
			}
			status := SlotOpen
			if closed {
				status = SlotClosed
			}
			slots = append(slots, Slot{ID: snowflake.Generate().Int64(), ResourceID: id, StartAt: start, EndAt: end, Capacity: resource.Capacity, Status: status, CreatedAt: now, UpdatedAt: now})
		}
	}
	if e := s.repo.EnsureSlots(ctx, slots); e != nil {
		return nil, bizerr.InternalWrap("生成预约时段失败", e)
	}
	list, e := s.repo.ListSlots(ctx, id, date, date.AddDate(0, 0, 1))
	if e != nil {
		return nil, e
	}
	items := make([]SlotResponse, 0, len(list))
	for _, slot := range list {
		items = append(items, slotResponse(slot))
	}
	return items, nil
}

func (s *Service) CreateBooking(ctx context.Context, userID, rootUserID int64, req CreateBookingReq) (*BookingResponse, error) {
	if s.capabilities != nil {
		if e := s.capabilities.CheckCapability(ctx, userID, rootUserID, "reservation"); e != nil {
			return nil, e
		}
	}
	slotID, e := parseID(req.SlotID)
	if e != nil {
		return nil, e
	}
	now := s.now()
	item := &Booking{ID: snowflake.Generate().Int64(), RootUserID: rootUserID, UserID: userID, SlotID: slotID, Status: BookingReserved, CheckinCode: snowflake.Generate().String(), CreatedAt: now, UpdatedAt: now}
	slot, _, e := s.repo.CreateBooking(ctx, item, now)
	switch {
	case errors.Is(e, errSlotUnavailable):
		return nil, ErrSlotUnavailable
	case errors.Is(e, errBookingConflict):
		return nil, ErrBookingConflict
	case errors.Is(e, errDailyLimit):
		return nil, ErrDailyLimit
	case e != nil:
		return nil, bizerr.InternalWrap("创建预约失败", e)
	}
	item.Slot = *slot
	response := bookingResponse(*item)
	return &response, nil
}
func (s *Service) ListBookings(ctx context.Context, rootUserID int64, page, size int) (*pagination.PageResult[BookingResponse], error) {
	page, size = normalize(page, size)
	list, total, e := s.repo.ListBookings(ctx, rootUserID, page, size)
	if e != nil {
		return nil, e
	}
	items := make([]BookingResponse, 0, len(list))
	for _, v := range list {
		items = append(items, bookingResponse(v))
	}
	return pagination.NewPageResult(items, total, page, size), nil
}
func (s *Service) CancelBooking(ctx context.Context, rootUserID int64, bookingID string) error {
	id, e := parseID(bookingID)
	if e != nil {
		return e
	}
	ok, e := s.repo.CancelBooking(ctx, id, rootUserID, s.now())
	if errors.Is(e, errCancelDeadline) {
		return ErrCancelDeadline
	}
	if e != nil {
		return bizerr.InternalWrap("取消预约失败", e)
	}
	if !ok {
		return ErrBookingNotFound
	}
	return nil
}

func (s *Service) CreateVenue(ctx context.Context, req VenueReq) (*VenueResponse, error) {
	if req.AdvanceDays <= 0 {
		req.AdvanceDays = 7
	}
	if req.SlotMinutes <= 0 {
		req.SlotMinutes = 60
	}
	if req.DailyLimit <= 0 {
		req.DailyLimit = 2
	}
	if req.CancelBeforeMinutes <= 0 {
		req.CancelBeforeMinutes = 120
	}
	now := s.now()
	item := &Venue{ID: snowflake.Generate().Int64(), Name: strings.TrimSpace(req.Name), Description: strings.TrimSpace(req.Description), Status: StatusActive, AdvanceDays: req.AdvanceDays, SlotMinutes: req.SlotMinutes, DailyLimit: req.DailyLimit, CancelBeforeMinutes: req.CancelBeforeMinutes, CreatedAt: now, UpdatedAt: now}
	if e := s.repo.CreateVenue(ctx, item); e != nil {
		return nil, bizerr.InternalWrap("创建场馆失败", e)
	}
	r := venueResponse(*item)
	return &r, nil
}
func (s *Service) CreateResource(ctx context.Context, venueID string, req ResourceReq) (*ResourceResponse, error) {
	id, e := parseID(venueID)
	if e != nil {
		return nil, e
	}
	if req.Capacity <= 0 {
		req.Capacity = 1
	}
	now := s.now()
	item := &Resource{ID: snowflake.Generate().Int64(), VenueID: id, Name: strings.TrimSpace(req.Name), Capacity: req.Capacity, Status: StatusActive, CreatedAt: now, UpdatedAt: now}
	if e := s.repo.CreateResource(ctx, item); e != nil {
		return nil, bizerr.InternalWrap("创建场地资源失败", e)
	}
	r := resourceResponse(*item)
	return &r, nil
}
func (s *Service) CreateRule(ctx context.Context, resourceID string, req RuleReq) error {
	id, e := parseID(resourceID)
	if e != nil {
		return e
	}
	if req.StartMinute < 0 || req.EndMinute > 1440 || req.EndMinute <= req.StartMinute {
		return bizerr.Param("开放时间不合法")
	}
	now := s.now()
	return s.repo.CreateRule(ctx, &WeeklyRule{ID: snowflake.Generate().Int64(), ResourceID: id, Weekday: req.Weekday, StartMinute: req.StartMinute, EndMinute: req.EndMinute, Status: StatusActive, CreatedAt: now, UpdatedAt: now})
}
func (s *Service) CreateClosure(ctx context.Context, adminID int64, req ClosureReq) error {
	var venueID, resourceID *int64
	var e error
	if strings.TrimSpace(req.VenueID) != "" {
		v, err := parseID(req.VenueID)
		if err != nil {
			return err
		}
		venueID = &v
	}
	if strings.TrimSpace(req.ResourceID) != "" {
		v, err := parseID(req.ResourceID)
		if err != nil {
			return err
		}
		resourceID = &v
	}
	if venueID == nil && resourceID == nil {
		return bizerr.Param("场馆或资源不能为空")
	}
	start, e := time.Parse(time.RFC3339, req.StartAt)
	if e != nil {
		return bizerr.Param("闭馆开始时间格式错误")
	}
	end, e := time.Parse(time.RFC3339, req.EndAt)
	if e != nil || !end.After(start) {
		return bizerr.Param("闭馆结束时间格式错误")
	}
	now := s.now()
	roots, e := s.repo.CreateClosure(ctx, &Closure{ID: snowflake.Generate().Int64(), VenueID: venueID, ResourceID: resourceID, StartAt: start, EndAt: end, Reason: req.Reason, CreatedBy: adminID, CreatedAt: now}, now)
	if e != nil {
		return bizerr.InternalWrap("创建闭馆失败", e)
	}
	for _, root := range roots {
		s.notify(ctx, root, "reservation.closure", "预约因闭馆取消", req.Reason, "")
	}
	return nil
}
func (s *Service) Checkin(ctx context.Context, code string) error {
	ok, e := s.repo.Checkin(ctx, strings.TrimSpace(code), s.now())
	if e != nil {
		return e
	}
	if !ok {
		return ErrCheckinInvalid
	}
	return nil
}
func (s *Service) RunDueJobs(ctx context.Context) error {
	items, e := s.repo.MarkNoShows(ctx, s.now())
	if e != nil {
		return e
	}
	for _, item := range items {
		s.notify(ctx, item.RootUserID, "reservation.noShow", "预约已标记为爽约", "预约已结束但未完成核销", id(item.ID))
	}
	if len(items) > 0 {
		s.logger.Info("reservation no-shows marked", zap.Int("count", len(items)))
	}
	return nil
}
func (s *Service) notify(ctx context.Context, root int64, event, title, content, resource string) {
	if s.notifier != nil {
		_ = s.notifier.NotifyReservation(ctx, root, event, title, content, resource)
	}
}
func parseID(raw string) (int64, error) {
	v, e := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if e != nil || v <= 0 {
		return 0, bizerr.Param("ID格式错误")
	}
	return v, nil
}
func dayStart(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}
func normalize(p, z int) (int, int) {
	if p <= 0 {
		p = 1
	}
	if z <= 0 {
		z = 15
	}
	if z > 100 {
		z = 100
	}
	return p, z
}
