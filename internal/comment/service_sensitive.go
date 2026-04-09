package comment

import (
	"context"

	"github.com/Milchstrassse/Ecampus-go/internal/platform/bizerr"
)

func (s *Service) filterText(ctx context.Context, content string) (string, error) {
	if s == nil || s.sensitiveFilter == nil || content == "" {
		return content, nil
	}

	filtered, err := s.sensitiveFilter.FilterText(ctx, content)
	if err != nil {
		return "", bizerr.InternalWrap("过滤评论敏感词失败", err)
	}
	return filtered, nil
}
