package vote

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/config"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
)

type Service struct {
	db  *gorm.DB
	cfg *config.Config
}

func NewService(db *gorm.DB, cfg *config.Config) *Service {
	return &Service{
		db:  db,
		cfg: cfg,
	}
}

func (s *Service) defaultPageSize() int {
	if s != nil && s.cfg != nil && s.cfg.Custom.PageSize > 0 {
		return s.cfg.Custom.PageSize
	}
	return 15
}

func (s *Service) ListVotes(ctx context.Context, page, size int) (*result.PageResult[VoteInfo], error) {
	if size <= 0 {
		size = s.defaultPageSize()
	}
	var total int64
	if err := s.db.WithContext(ctx).Model(&VoteInfo{}).Count(&total).Error; err != nil {
		return nil, fmt.Errorf("count vote infos: %w", err)
	}
	var list []VoteInfo
	if err := s.db.WithContext(ctx).
		Offset((page - 1) * size).
		Limit(size).
		Order("updated_at DESC").
		Find(&list).Error; err != nil {
		return nil, fmt.Errorf("list vote infos: %w", err)
	}
	return result.NewPage(list, total, page, size), nil
}

func (s *Service) GetVoteOptions(
	ctx context.Context,
	voteInfoID, userID int64,
	page, size, isOK int,
) (*result.PageResult[VoteOption], error) {
	if size <= 0 {
		size = s.defaultPageSize()
	}
	info, err := s.getVoteInfo(ctx, voteInfoID)
	if err != nil {
		return nil, err
	}
	if info.CreatedBy != userID {
		return nil, result.ErrNotExisted
	}

	query := s.db.WithContext(ctx).Model(&VoteOption{}).
		Where("vote_info_id = ?", voteInfoID).
		Where("is_ok = ?", isOK)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("count vote options: %w", err)
	}

	var options []VoteOption
	if err := query.Offset((page - 1) * size).Limit(size).Order("id ASC").Find(&options).Error; err != nil {
		return nil, fmt.Errorf("list vote options: %w", err)
	}
	return result.NewPage(options, total, page, size), nil
}

func (s *Service) AcceptVoteOptions(ctx context.Context, voteInfoID, userID int64, optionIDs []int64) error {
	info, err := s.getVoteInfo(ctx, voteInfoID)
	if err != nil {
		return err
	}
	if info.CreatedBy != userID || len(optionIDs) == 0 {
		return result.ErrNotExisted
	}
	if info.AccessDraft != 1 {
		return result.NewBizError(result.CodeFail, "当前投票不接受投稿")
	}
	if err := s.db.WithContext(ctx).
		Model(&VoteOption{}).
		Where("vote_info_id = ?", voteInfoID).
		Update("is_ok", 1).Error; err != nil {
		return fmt.Errorf("accept vote options: %w", err)
	}
	return nil
}

func (s *Service) CreateVoteInfo(ctx context.Context, userID int64, req *VoteCreateReq) error {
	if req == nil {
		return result.ErrParam
	}
	if err := validateVoteInfo(req.Info, len(req.Options)); err != nil {
		return err
	}

	req.Info.CreatedBy = userID
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&req.Info).Error; err != nil {
			return fmt.Errorf("create vote info: %w", err)
		}
		if len(req.Options) == 0 {
			return nil
		}

		for i := range req.Options {
			req.Options[i].VoteInfoID = req.Info.ID
			req.Options[i].CreatedBy = userID
			req.Options[i].IsOk = 1
		}
		if err := tx.Create(&req.Options).Error; err != nil {
			return fmt.Errorf("create vote options: %w", err)
		}
		return nil
	})
}

func (s *Service) AddVoteOption(ctx context.Context, voteInfoID, userID int64, option *VoteOption) error {
	info, err := s.getVoteInfo(ctx, voteInfoID)
	if err != nil {
		return err
	}
	if info.AccessDraft != 1 {
		return result.NewBizError(result.CodeFail, "当前投票不接受投稿")
	}
	if info.AccessEndTime.Before(time.Now()) {
		return result.NewBizError(result.CodeFail, "投稿已截止")
	}

	option.VoteInfoID = voteInfoID
	option.CreatedBy = userID
	if err := s.db.WithContext(ctx).Create(option).Error; err != nil {
		return fmt.Errorf("add vote option: %w", err)
	}
	return nil
}

func (s *Service) Vote(ctx context.Context, voteInfoID, voteUserID int64, optionIDs []int64) error {
	info, err := s.getVoteInfo(ctx, voteInfoID)
	if err != nil {
		return err
	}
	now := time.Now()
	if info.VoteStart.After(now) {
		return result.NewBizError(result.CodeFail, "投票还没开始")
	}
	if info.VoteEnd.Before(now) {
		return result.NewBizError(result.CodeFail, "投票已截止")
	}
	if len(optionIDs) == 0 {
		return result.NewBizError(result.CodeFail, "请选择正确的选项")
	}
	if info.OptionType != 2 && len(optionIDs) > 1 {
		return result.NewBizError(result.CodeFail, "当前投票只能选一个")
	}

	validIDs, err := s.findVoteOptionIDs(ctx, optionIDs)
	if err != nil {
		return err
	}
	if len(validIDs) == 0 {
		return result.NewBizError(result.CodeFail, "请确认投票选项")
	}

	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, optionID := range validIDs {
			ans := VoteAns{
				VoteInfoID:   voteInfoID,
				VoteDate:     today,
				VoteUserID:   voteUserID,
				VoteOptionID: optionID,
			}
			if err := tx.Clauses(clause.Insert{Modifier: "IGNORE"}).Create(&ans).Error; err != nil {
				return fmt.Errorf("insert vote answer: %w", err)
			}
		}
		return nil
	})
}

func (s *Service) getVoteInfo(ctx context.Context, voteInfoID int64) (*VoteInfo, error) {
	var info VoteInfo
	if err := s.db.WithContext(ctx).First(&info, voteInfoID).Error; err != nil {
		return nil, fmt.Errorf("get vote info: %w", err)
	}
	return &info, nil
}

func (s *Service) findVoteOptionIDs(ctx context.Context, optionIDs []int64) ([]int64, error) {
	var options []VoteOption
	if err := s.db.WithContext(ctx).Where("id IN ?", optionIDs).Find(&options).Error; err != nil {
		return nil, fmt.Errorf("list vote options: %w", err)
	}

	ids := make([]int64, 0, len(options))
	for _, option := range options {
		ids = append(ids, option.ID)
	}
	return ids, nil
}

func validateVoteInfo(info VoteInfo, optionCount int) error {
	if info.AccessDraft == 0 && optionCount == 0 {
		return result.NewBizError(result.CodeFail, "不接受投稿的投票需要至少一个投票选项")
	}
	if info.VoteStart.After(info.VoteEnd) {
		return result.NewBizError(result.CodeFail, "投票开始时间不能比投票截止时间晚")
	}
	if info.AccessDraft != 0 && info.AccessEndTime.After(info.VoteStart) {
		return result.NewBizError(result.CodeFail, "接受投稿的投票的投稿截止时间不能比开始投票时间晚")
	}
	return nil
}
