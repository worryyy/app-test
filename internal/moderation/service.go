package moderation

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/Milchstrassse/Ecampus-go/internal/platform/bizerr"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/pagination"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/snowflake"
)

const capabilityCacheTTL = 5 * time.Minute

var (
	ErrReportNotFound      = bizerr.NotFound("举报不存在或状态已变化")
	ErrReportTargetInvalid = bizerr.NotFound("举报对象不存在或无权举报")
	ErrAppealAlreadyExists = bizerr.Biz("该处罚已提交申诉")
	ErrPunishmentNotFound  = bizerr.NotFound("处罚不存在")
	ErrCapabilityDenied    = bizerr.Forbidden("当前账号能力已被限制")
)

type TargetSnapshot struct {
	OwnerRootUserID int64
	Payload         any
}

type TargetResolver interface {
	ResolveTarget(ctx context.Context, reporterRootUserID, reporterUserID int64, targetType, targetID string) (*TargetSnapshot, error)
	HideTarget(ctx context.Context, targetType, targetID string) error
}

type Notifier interface {
	NotifyModeration(ctx context.Context, rootUserID int64, eventType, title, content, resourceType, resourceID string) error
}

type Service struct {
	repo     *Repository
	redis    *redis.Client
	logger   *zap.Logger
	targets  TargetResolver
	notifier Notifier
	now      func() time.Time
}

func NewService(db *gorm.DB, rds *redis.Client, logger *zap.Logger) *Service {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Service{repo: NewRepository(db), redis: rds, logger: logger, now: time.Now}
}

func (s *Service) SetTargetResolver(resolver TargetResolver) { s.targets = resolver }
func (s *Service) SetNotifier(notifier Notifier)             { s.notifier = notifier }

func (s *Service) CreateReport(ctx context.Context, rootUserID, userID int64, req CreateReportReq) (*ReportResponse, error) {
	req.TargetType, req.TargetID, req.Reason = strings.TrimSpace(req.TargetType), strings.TrimSpace(req.TargetID), strings.TrimSpace(req.Reason)
	if rootUserID <= 0 || userID <= 0 || req.TargetID == "" || req.Reason == "" {
		return nil, bizerr.Param("举报参数错误")
	}
	if _, ok := targetTypes[req.TargetType]; !ok || s.targets == nil {
		return nil, ErrReportTargetInvalid
	}
	snapshot, err := s.targets.ResolveTarget(ctx, rootUserID, userID, req.TargetType, req.TargetID)
	if err != nil || snapshot == nil || snapshot.OwnerRootUserID <= 0 {
		if err != nil {
			s.logger.Info("resolve report target failed", zap.Error(err))
		}
		return nil, ErrReportTargetInvalid
	}
	payload, err := json.Marshal(snapshot.Payload)
	if err != nil {
		return nil, bizerr.InternalWrap("保存举报快照失败", err)
	}
	now := s.now()
	report := &Report{ID: snowflake.Generate().Int64(), ReporterRootUserID: rootUserID, ReporterUserID: userID, TargetType: req.TargetType, TargetID: req.TargetID, TargetRootUserID: snapshot.OwnerRootUserID, Reason: req.Reason, Description: strings.TrimSpace(req.Description), Status: ReportPending, CreatedAt: now, UpdatedAt: now}
	reportSnapshot := &ReportSnapshot{ID: snowflake.Generate().Int64(), ReportID: report.ID, Payload: payload, CreatedAt: now}
	if err := s.repo.CreateReport(ctx, report, reportSnapshot); err != nil {
		return nil, bizerr.InternalWrap("提交举报失败", err)
	}
	response := reportPage([]Report{*report}, 1, 1, 1).Data[0]
	return &response, nil
}

func (s *Service) ListMyReports(ctx context.Context, rootUserID int64, page, size int) (*pagination.PageResult[ReportResponse], error) {
	page, size = normalize(page, size)
	list, total, err := s.repo.ListReports(ctx, "reporter_root_user_id = ?", []any{rootUserID}, page, size)
	if err != nil {
		return nil, bizerr.InternalWrap("查询举报失败", err)
	}
	return reportPage(list, total, page, size), nil
}

func (s *Service) WithdrawReport(ctx context.Context, rootUserID int64, reportID string) error {
	idValue, err := parseID(reportID)
	if err != nil {
		return err
	}
	ok, err := s.repo.WithdrawReport(ctx, idValue, rootUserID, s.now())
	if err != nil {
		return bizerr.InternalWrap("撤回举报失败", err)
	}
	if !ok {
		return ErrReportNotFound
	}
	return nil
}

func (s *Service) ListMyPunishments(ctx context.Context, rootUserID int64, page, size int) (*pagination.PageResult[PunishmentResponse], error) {
	page, size = normalize(page, size)
	list, total, err := s.repo.ListPunishments(ctx, rootUserID, page, size)
	if err != nil {
		return nil, bizerr.InternalWrap("查询处罚失败", err)
	}
	return punishmentPage(list, total, page, size), nil
}

func (s *Service) CreateAppeal(ctx context.Context, rootUserID int64, punishmentID string, req CreateAppealReq) (*AppealResponse, error) {
	idValue, err := parseID(punishmentID)
	if err != nil {
		return nil, err
	}
	item, err := s.repo.FindPunishment(ctx, idValue)
	if err != nil {
		return nil, bizerr.InternalWrap("查询处罚失败", err)
	}
	if item == nil || item.RootUserID != rootUserID {
		return nil, ErrPunishmentNotFound
	}
	now := s.now()
	appeal := &Appeal{ID: snowflake.Generate().Int64(), PunishmentID: idValue, RootUserID: rootUserID, Reason: strings.TrimSpace(req.Reason), Status: AppealPending, CreatedAt: now, UpdatedAt: now}
	if appeal.Reason == "" {
		return nil, bizerr.Param("申诉理由不能为空")
	}
	if err := s.repo.CreateAppeal(ctx, appeal); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return nil, ErrAppealAlreadyExists
		}
		return nil, bizerr.InternalWrap("提交申诉失败", err)
	}
	response := appealPage([]Appeal{*appeal}, 1, 1, 1).Data[0]
	return &response, nil
}

func (s *Service) ListMyAppeals(ctx context.Context, rootUserID int64, page, size int) (*pagination.PageResult[AppealResponse], error) {
	page, size = normalize(page, size)
	list, total, err := s.repo.ListAppeals(ctx, &rootUserID, page, size)
	if err != nil {
		return nil, bizerr.InternalWrap("查询申诉失败", err)
	}
	return appealPage(list, total, page, size), nil
}

func (s *Service) ListReportsAdmin(ctx context.Context, status string, page, size int) (*pagination.PageResult[ReportResponse], error) {
	page, size = normalize(page, size)
	var query any
	var args []any
	if strings.TrimSpace(status) != "" {
		query, args = "status = ?", []any{status}
	}
	list, total, err := s.repo.ListReports(ctx, query, args, page, size)
	if err != nil {
		return nil, bizerr.InternalWrap("查询举报失败", err)
	}
	return reportPage(list, total, page, size), nil
}

func (s *Service) ClaimReport(ctx context.Context, reportID string, adminID int64) error {
	idValue, err := parseID(reportID)
	if err != nil {
		return err
	}
	ok, err := s.repo.ClaimReport(ctx, idValue, adminID, s.now())
	if err != nil {
		return bizerr.InternalWrap("领取举报失败", err)
	}
	if !ok {
		return ErrReportNotFound
	}
	return nil
}

func (s *Service) DecideReport(ctx context.Context, reportID string, adminID int64, req AdminDecisionReq) error {
	idValue, err := parseID(reportID)
	if err != nil {
		return err
	}
	report, err := s.repo.FindReport(ctx, idValue)
	if err != nil {
		return bizerr.InternalWrap("查询举报失败", err)
	}
	if report == nil || report.Status != ReportReviewing || report.AssigneeAdminID == nil || *report.AssigneeAdminID != adminID {
		return ErrReportNotFound
	}
	if req.Action == "hide" {
		if s.targets == nil || s.targets.HideTarget(ctx, report.TargetType, report.TargetID) != nil {
			return bizerr.Biz("下架举报对象失败")
		}
	}
	now := s.now()
	status := ReportActioned
	if req.Action == "reject" {
		status = ReportRejected
	}
	var punishment *Punishment
	if req.Action == "punish" {
		if _, ok := capabilities[req.Capability]; !ok {
			return bizerr.Param("处罚能力不合法")
		}
		var endsAt *time.Time
		if req.DurationMinutes != nil {
			if *req.DurationMinutes <= 0 {
				return bizerr.Param("处罚时长不合法")
			}
			value := now.Add(time.Duration(*req.DurationMinutes) * time.Minute)
			endsAt = &value
		}
		reportRef := report.ID
		punishment = &Punishment{ID: snowflake.Generate().Int64(), RootUserID: report.TargetRootUserID, ReportID: &reportRef, Capability: req.Capability, Reason: strings.TrimSpace(req.Reason), Status: PunishmentActive, StartsAt: now, EndsAt: endsAt, CreatedBy: adminID, CreatedAt: now}
	}
	audit := &AuditLog{ID: snowflake.Generate().Int64(), ReportID: &report.ID, AdminID: adminID, Action: req.Action, Detail: map[string]any{"reason": req.Reason}, CreatedAt: now}
	if punishment != nil {
		audit.PunishmentID = &punishment.ID
	}
	ok, err := s.repo.CompleteReview(ctx, report.ID, adminID, status, punishment, audit, now)
	if err != nil {
		return bizerr.InternalWrap("处理举报失败", err)
	}
	if !ok {
		return ErrReportNotFound
	}
	if punishment != nil {
		s.invalidate(ctx, punishment.RootUserID)
	}
	s.notify(ctx, report.TargetRootUserID, "report."+req.Action, "举报处理结果", req.Reason, "report", idValue)
	return nil
}

func (s *Service) RevokePunishment(ctx context.Context, punishmentID string, adminID int64, reason string) error {
	idValue, err := parseID(punishmentID)
	if err != nil {
		return err
	}
	audit := &AuditLog{ID: snowflake.Generate().Int64(), AdminID: adminID, Action: "revoke", Detail: map[string]any{"reason": reason}, CreatedAt: s.now()}
	rootUserID, ok, err := s.repo.RevokePunishment(ctx, idValue, adminID, reason, s.now(), audit)
	if err != nil {
		return bizerr.InternalWrap("撤销处罚失败", err)
	}
	if !ok {
		return ErrPunishmentNotFound
	}
	s.invalidate(ctx, rootUserID)
	s.notify(ctx, rootUserID, "punishment.revoked", "处罚已撤销", reason, "punishment", idValue)
	return nil
}

func (s *Service) ListAppealsAdmin(ctx context.Context, page, size int) (*pagination.PageResult[AppealResponse], error) {
	page, size = normalize(page, size)
	list, total, err := s.repo.ListAppeals(ctx, nil, page, size)
	if err != nil {
		return nil, bizerr.InternalWrap("查询申诉失败", err)
	}
	return appealPage(list, total, page, size), nil
}

func (s *Service) DecideAppeal(ctx context.Context, appealID string, adminID int64, req AppealDecisionReq) error {
	idValue, err := parseID(appealID)
	if err != nil {
		return err
	}
	action := "appeal.reject"
	if req.Approved {
		action = "appeal.approve"
	}
	audit := &AuditLog{ID: snowflake.Generate().Int64(), AdminID: adminID, Action: action, Detail: map[string]any{"resolution": req.Resolution}, CreatedAt: s.now()}
	rootUserID, ok, err := s.repo.DecideAppeal(ctx, idValue, adminID, req.Approved, req.Resolution, s.now(), audit)
	if err != nil {
		return bizerr.InternalWrap("处理申诉失败", err)
	}
	if !ok {
		return bizerr.NotFound("申诉不存在或已处理")
	}
	if req.Approved {
		s.invalidate(ctx, rootUserID)
	}
	s.notify(ctx, rootUserID, action, "申诉处理结果", req.Resolution, "appeal", idValue)
	return nil
}

func (s *Service) Check(ctx context.Context, rootUserID int64, capability string) error {
	if rootUserID <= 0 {
		return bizerr.Param("主账号参数错误")
	}
	active, err := s.activeCapabilities(ctx, rootUserID)
	if err != nil {
		return bizerr.InternalWrap("查询账号处罚失败", err)
	}
	if active[CapabilityAccount] || active[capability] {
		return ErrCapabilityDenied
	}
	return nil
}

func (s *Service) AccountBlocked(ctx context.Context, rootUserID int64) (bool, error) {
	active, err := s.activeCapabilities(ctx, rootUserID)
	return active[CapabilityAccount], err
}

func (s *Service) activeCapabilities(ctx context.Context, rootUserID int64) (map[string]bool, error) {
	key := "campus:moderation:capabilities:" + strconv.FormatInt(rootUserID, 10)
	if s.redis != nil {
		var cached map[string]bool
		if raw, err := s.redis.Get(ctx, key).Bytes(); err == nil && json.Unmarshal(raw, &cached) == nil {
			return cached, nil
		}
	}
	now := s.now()
	list, err := s.repo.ActivePunishments(ctx, rootUserID, now)
	if err != nil {
		return nil, err
	}
	active := make(map[string]bool, len(list))
	ttl := capabilityCacheTTL
	for _, item := range list {
		active[item.Capability] = true
		if item.EndsAt != nil && item.EndsAt.Sub(now) < ttl {
			ttl = item.EndsAt.Sub(now)
		}
	}
	if ttl <= 0 {
		ttl = time.Second
	}
	if s.redis != nil {
		if raw, err := json.Marshal(active); err == nil {
			_ = s.redis.Set(ctx, key, raw, ttl).Err()
		}
	}
	return active, nil
}

func (s *Service) invalidate(ctx context.Context, rootUserID int64) {
	if s.redis != nil {
		_ = s.redis.Del(ctx, "campus:moderation:capabilities:"+strconv.FormatInt(rootUserID, 10)).Err()
	}
}

func (s *Service) notify(ctx context.Context, rootUserID int64, eventType, title, content, resourceType string, resourceID int64) {
	if s.notifier != nil {
		_ = s.notifier.NotifyModeration(ctx, rootUserID, eventType, title, content, resourceType, id(resourceID))
	}
}

func parseID(raw string) (int64, error) {
	value, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || value <= 0 {
		return 0, bizerr.Param("ID格式错误")
	}
	return value, nil
}

func normalize(page, size int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 15
	}
	if size > 100 {
		size = 100
	}
	return page, size
}
