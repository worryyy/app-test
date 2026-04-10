package school

import (
	"context"

	"github.com/Milchstrassse/Ecampus-go/internal/platform/bizerr"
)

func (s *Service) AddTerm(ctx context.Context, term *Term) (*Term, error) {
	if err := validateTerm(term); err != nil {
		return nil, err
	}

	exists, err := s.repo.CountTermsByValue(ctx, term.Term)
	if err != nil {
		return nil, bizerr.InternalWrap("check term failed", err)
	}
	if exists > 0 {
		return nil, bizerr.Biz("term already exists: " + term.Term)
	}

	if err := s.repo.CreateTerm(ctx, term); err != nil {
		return nil, bizerr.InternalWrap("create term failed", err)
	}
	return term, nil
}

func (s *Service) DeleteTerm(ctx context.Context, termID string) error {
	term, err := s.termByID(ctx, termID)
	if err != nil {
		return err
	}

	current, err := s.currentTermRecord(ctx)
	if err != nil {
		return err
	}
	if current != nil && current.Term == term.Term {
		return bizerr.Param("switch current term before deleting it")
	}

	deleted, err := s.repo.DeleteTermByID(ctx, term.ID)
	if err != nil {
		return bizerr.InternalWrap("delete term failed", err)
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

	current, err := s.currentTermRecord(ctx)
	if err != nil {
		return nil, err
	}
	if current == nil {
		current = &CurTerm{Term: term.Term}
		if err := s.repo.CreateCurrentTerm(ctx, current); err != nil {
			return nil, bizerr.InternalWrap("set current term failed", err)
		}
		return current, nil
	}
	if current.Term == term.Term {
		return current, nil
	}

	if err := s.repo.UpdateCurrentTerm(ctx, current.ID, term.Term); err != nil {
		return nil, bizerr.InternalWrap("set current term failed", err)
	}
	current.Term = term.Term
	return current, nil
}

func (s *Service) ListTerms(ctx context.Context) ([]Term, error) {
	terms, err := s.repo.ListTerms(ctx)
	if err != nil {
		return nil, bizerr.InternalWrap("list terms failed", err)

	}
	return terms, nil
}
