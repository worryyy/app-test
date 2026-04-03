package school

import (
	"context"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/bizerr"
)

func (s *Service) AddTerm(ctx context.Context, term *Term) (*Term, error) {
	if err := validateTerm(term); err != nil {
		return nil, err
	}

	exists, err := s.repo.CountTermsByValue(ctx, term.Term)
	if err != nil {
		return nil, bizerr.InternalWrap("检查学期失败", err)
	}
	if exists > 0 {
		return nil, bizerr.Biz("term: " + term.Term + "已存在")
	}

	if err := s.repo.CreateTerm(ctx, term); err != nil {
		return nil, bizerr.InternalWrap("新增学期失败", err)
	}
	return term, nil
}

func (s *Service) DeleteTerm(ctx context.Context, termID string) error {
	term, err := s.termByID(ctx, termID)
	if err != nil {
		return err
	}

	current, err := s.repo.FindCurrentTerm(ctx)
	if err != nil {
		return bizerr.InternalWrap("查询当前学期失败", err)
	}
	if current != nil && current.Term == term.Term {
		return bizerr.Param("请先更新当前学期为其他学期后重新删除")
	}

	deleted, err := s.repo.DeleteTermByID(ctx, term.ID)
	if err != nil {
		return bizerr.InternalWrap("删除学期失败", err)
	}
	if !deleted {
		return ErrTermNotFound
	}
	return nil
}

func (s *Service) SetCurrentTerm(ctx context.Context, termID string) (*CurTerm, error) {
	term, err := s.termByID(ctx, termID)
	if err != nil {
		return nil, err
	}

	current, err := s.repo.FindCurrentTerm(ctx)
	if err != nil {
		return nil, bizerr.InternalWrap("查询当前学期失败", err)
	}
	if current == nil {
		current = &CurTerm{Term: term.Term}
		if err := s.repo.CreateCurrentTerm(ctx, current); err != nil {
			return nil, bizerr.InternalWrap("设置当前学期失败", err)
		}
		return current, nil
	}
	if current.Term == term.Term {
		return current, nil
	}

	if err := s.repo.UpdateCurrentTerm(ctx, current.ID, term.Term); err != nil {
		return nil, bizerr.InternalWrap("设置当前学期失败", err)
	}
	current.Term = term.Term
	return current, nil
}
