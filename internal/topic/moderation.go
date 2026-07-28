package topic

import "context"

func (s *Service) ReportTarget(ctx context.Context, topicID string) (*Topic, error) {
	return s.getTopicByID(ctx, topicID, false)
}

func (s *Service) HideForModeration(ctx context.Context, topicID string) error {
	return s.DeleteAdmin(ctx, topicID)
}
