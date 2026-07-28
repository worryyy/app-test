package comment

import "context"

func (s *Service) ReportTarget(ctx context.Context, commentID string) (*Comment, error) {
	return s.getCommentByID(ctx, commentID)
}

func (s *Service) HideForModeration(ctx context.Context, commentID string) error {
	comment, err := s.getCommentByID(ctx, commentID)
	if err != nil {
		return err
	}
	return s.DeleteCommentAdmin(ctx, comment.TopicID, commentID)
}
