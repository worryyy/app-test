package other

import (
	"context"
	"fmt"
	"time"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
)

func (s *Service) ListVotes(ctx context.Context, page, size int) (*result.PageResult[VoteInfo], error) {
	var total int64
	if err := s.db.WithContext(ctx).Model(&VoteInfo{}).Count(&total).Error; err != nil {
		return nil, fmt.Errorf("count vote infos: %w", err)
	}
	var list []VoteInfo
	if err := s.db.WithContext(ctx).Offset((page - 1) * size).Limit(size).Order("id DESC").Find(&list).Error; err != nil {
		return nil, fmt.Errorf("list vote infos: %w", err)
	}
	return result.NewPage(list, total, page, size), nil
}

func (s *Service) GetVoteOptions(ctx context.Context, voteInfoID int64) ([]VoteOption, error) {
	var options []VoteOption
	if err := s.db.WithContext(ctx).Where("voteInfoId = ?", voteInfoID).Order("id ASC").Find(&options).Error; err != nil {
		return nil, fmt.Errorf("get vote options: %w", err)
	}
	return options, nil
}

func (s *Service) AcceptVoteOptions(ctx context.Context, voteInfoID int64, optionIDs []int64) error {
	if err := s.db.WithContext(ctx).Model(&VoteOption{}).Where("voteInfoId = ?", voteInfoID).Update("isOk", false).Error; err != nil {
		return fmt.Errorf("reset vote options: %w", err)
	}
	if len(optionIDs) > 0 {
		if err := s.db.WithContext(ctx).Model(&VoteOption{}).Where("voteInfoId = ? AND id IN ?", voteInfoID, optionIDs).Update("isOk", true).Error; err != nil {
			return fmt.Errorf("accept vote options: %w", err)
		}
	}
	if err := s.db.WithContext(ctx).Model(&VoteInfo{}).Where("id = ?", voteInfoID).Update("accessDraft", false).Error; err != nil {
		return fmt.Errorf("update vote info accessDraft: %w", err)
	}
	return nil
}

func (s *Service) CreateVoteInfo(ctx context.Context, info *VoteInfo) error {
	if err := s.db.WithContext(ctx).Create(info).Error; err != nil {
		return fmt.Errorf("create vote info: %w", err)
	}
	return nil
}

func (s *Service) AddVoteOption(ctx context.Context, option *VoteOption) error {
	if err := s.db.WithContext(ctx).Create(option).Error; err != nil {
		return fmt.Errorf("add vote option: %w", err)
	}
	return nil
}

func (s *Service) Vote(ctx context.Context, voteInfoID, voteUserID int64, optionIDs []int64) error {
	tx := s.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Where("voteInfoId = ? AND voteUserId = ?", voteInfoID, voteUserID).Delete(&VoteAns{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("clear old vote answers: %w", err)
	}

	for _, optionID := range optionIDs {
		ans := VoteAns{
			VoteInfoID:   voteInfoID,
			VoteDate:     time.Now(),
			VoteUserID:   voteUserID,
			VoteOptionID: optionID,
		}
		if err := tx.Create(&ans).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("create vote answer: %w", err)
		}
	}
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("commit vote transaction: %w", err)
	}
	return nil
}
