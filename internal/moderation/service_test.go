package moderation

import (
	"context"
	"errors"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type targetStub struct{}

func (targetStub) ResolveTarget(context.Context, int64, int64, string, string) (*TargetSnapshot, error) {
	return &TargetSnapshot{OwnerRootUserID: 200, Payload: map[string]any{"content": "snapshot"}}, nil
}
func (targetStub) HideTarget(context.Context, string, string) error { return nil }

func TestReportPunishmentAndAppealFlow(t *testing.T) {
	db := moderationTestDB(t)
	svc := NewService(db, nil, nil)
	svc.SetTargetResolver(targetStub{})
	now := time.Date(2026, 7, 28, 12, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	svc.now = func() time.Time { return now }

	report, err := svc.CreateReport(context.Background(), 100, 101, CreateReportReq{TargetType: "topic", TargetID: "abc", Reason: "spam"})
	if err != nil {
		t.Fatalf("create report: %v", err)
	}
	if report.Status != ReportPending {
		t.Fatalf("status = %s", report.Status)
	}
	if err := svc.ClaimReport(context.Background(), report.ID, 9); err != nil {
		t.Fatalf("claim report: %v", err)
	}
	if err := svc.ClaimReport(context.Background(), report.ID, 10); !errors.Is(err, ErrReportNotFound) {
		t.Fatalf("second claim error = %v", err)
	}

	duration := int64(60)
	if err := svc.DecideReport(context.Background(), report.ID, 9, AdminDecisionReq{Action: "punish", Reason: "confirmed", Capability: CapabilityContent, DurationMinutes: &duration}); err != nil {
		t.Fatalf("decide report: %v", err)
	}
	if err := svc.Check(context.Background(), 200, CapabilityContent); !errors.Is(err, ErrCapabilityDenied) {
		t.Fatalf("capability error = %v", err)
	}

	punishments, err := svc.ListMyPunishments(context.Background(), 200, 1, 10)
	if err != nil || len(punishments.Data) != 1 {
		t.Fatalf("punishments = %+v, err = %v", punishments, err)
	}
	appeal, err := svc.CreateAppeal(context.Background(), 200, punishments.Data[0].ID, CreateAppealReq{Reason: "please review"})
	if err != nil {
		t.Fatalf("create appeal: %v", err)
	}
	if _, err := svc.CreateAppeal(context.Background(), 200, punishments.Data[0].ID, CreateAppealReq{Reason: "again"}); !errors.Is(err, ErrAppealAlreadyExists) {
		t.Fatalf("duplicate appeal error = %v", err)
	}
	if err := svc.DecideAppeal(context.Background(), appeal.ID, 9, AppealDecisionReq{Approved: true, Resolution: "approved"}); err != nil {
		t.Fatalf("approve appeal: %v", err)
	}
	if err := svc.Check(context.Background(), 200, CapabilityContent); err != nil {
		t.Fatalf("capability should be restored: %v", err)
	}
}

func TestPendingReportCanBeWithdrawnOnlyByReporter(t *testing.T) {
	svc := NewService(moderationTestDB(t), nil, nil)
	svc.SetTargetResolver(targetStub{})
	report, err := svc.CreateReport(context.Background(), 100, 101, CreateReportReq{TargetType: "user", TargetID: "200", Reason: "abuse"})
	if err != nil {
		t.Fatalf("create report: %v", err)
	}
	if err := svc.WithdrawReport(context.Background(), 999, report.ID); !errors.Is(err, ErrReportNotFound) {
		t.Fatalf("other user withdraw error = %v", err)
	}
	if err := svc.WithdrawReport(context.Background(), 100, report.ID); err != nil {
		t.Fatalf("withdraw: %v", err)
	}
}

func moderationTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&Report{}, &ReportSnapshot{}, &Punishment{}, &Appeal{}, &AuditLog{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}
