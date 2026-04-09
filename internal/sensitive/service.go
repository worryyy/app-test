package sensitive

import (
	"context"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/Milchstrassse/Ecampus-go/internal/platform/bizerr"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/pagination"
)

const filterCacheTTL = 30 * time.Second

type Filter interface {
	FilterText(ctx context.Context, content string) (string, error)
}

type Service struct {
	repo   *Repository
	logger *zap.Logger

	cacheMu        sync.RWMutex
	cacheExpiresAt time.Time
	cachePattern   *regexp.Regexp
}

func NewService(db *gorm.DB, logger *zap.Logger) *Service {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Service{
		repo:   NewRepository(db),
		logger: logger,
	}
}

func (s *Service) FindAll(ctx context.Context) ([]SensitiveWord, error) {
	list, err := s.repo.FindAll(ctx)
	if err != nil {
		return nil, bizerr.InternalWrap("查询敏感词失败", err)
	}
	return list, nil
}

func (s *Service) FindByWord(ctx context.Context, word string) (*SensitiveWord, error) {
	word = strings.TrimSpace(word)
	if word == "" {
		return nil, bizerr.Param(errMsgInvalidParam)
	}

	item, err := s.repo.FindByWord(ctx, word)
	if err != nil {
		return nil, bizerr.InternalWrap("查询敏感词失败", err)
	}
	if item == nil {
		return nil, ErrSensitiveWordNotFound
	}
	return item, nil
}

func (s *Service) DeleteByWord(ctx context.Context, word string) error {
	word = strings.TrimSpace(word)
	if word == "" {
		return bizerr.Param(errMsgInvalidParam)
	}

	deleted, err := s.repo.DeleteByWord(ctx, word)
	if err != nil {
		return bizerr.InternalWrap("删除敏感词失败", err)
	}
	if !deleted {
		return ErrSensitiveWordNotFound
	}

	s.invalidateCache()
	return nil
}

func (s *Service) DeleteByList(ctx context.Context, words []string) error {
	normalized := normalizeWordList(words)
	if len(normalized) == 0 {
		return bizerr.Param(errMsgInvalidParam)
	}

	if _, err := s.repo.DeleteByWords(ctx, normalized); err != nil {
		return bizerr.InternalWrap("批量删除敏感词失败", err)
	}
	s.invalidateCache()
	return nil
}

func (s *Service) Add(ctx context.Context, word string) error {
	word = strings.TrimSpace(word)
	if word == "" {
		return bizerr.Param(errMsgInvalidParam)
	}

	exists, err := s.repo.FindByWord(ctx, word)
	if err != nil {
		return bizerr.InternalWrap("新增敏感词失败", err)
	}
	if exists != nil {
		return ErrSensitiveWordExists
	}

	if err := s.repo.Create(ctx, &SensitiveWord{Word: word}); err != nil {
		return bizerr.InternalWrap("新增敏感词失败", err)
	}
	s.invalidateCache()
	return nil
}

func (s *Service) AddByList(ctx context.Context, words []string) error {
	normalized := normalizeWordList(words)
	if len(normalized) == 0 {
		return bizerr.Param(errMsgInvalidParam)
	}

	existing, err := s.repo.FindByWords(ctx, normalized)
	if err != nil {
		return bizerr.InternalWrap("批量新增敏感词失败", err)
	}

	existingSet := make(map[string]struct{}, len(existing))
	for _, item := range existing {
		existingSet[item.Word] = struct{}{}
	}

	items := make([]SensitiveWord, 0, len(normalized))
	for _, word := range normalized {
		if _, ok := existingSet[word]; ok {
			continue
		}
		items = append(items, SensitiveWord{Word: word})
	}

	if len(items) == 0 {
		return nil
	}

	if err := s.repo.CreateBatch(ctx, items); err != nil {
		return bizerr.InternalWrap("批量新增敏感词失败", err)
	}
	s.invalidateCache()
	return nil
}

func (s *Service) FindByPage(ctx context.Context, page, size int) (*pagination.PageResult[SensitiveWord], error) {
	if page <= 0 || size <= 0 {
		return nil, bizerr.Param(errMsgInvalidParam)
	}

	list, total, err := s.repo.FindPage(ctx, page, size)
	if err != nil {
		return nil, bizerr.InternalWrap("查询敏感词失败", err)
	}
	return pagination.NewPageResult(list, total, page, size), nil
}

func (s *Service) FindByLike(ctx context.Context, word string) ([]SensitiveWord, error) {
	word = strings.TrimSpace(word)
	if word == "" {
		return nil, bizerr.Param(errMsgInvalidParam)
	}

	list, err := s.repo.FindByLike(ctx, word)
	if err != nil {
		return nil, bizerr.InternalWrap("查询敏感词失败", err)
	}
	return list, nil
}

func (s *Service) UpdateByWord(ctx context.Context, word, updateWord string) error {
	word = strings.TrimSpace(word)
	updateWord = strings.TrimSpace(updateWord)
	if word == "" || updateWord == "" {
		return bizerr.Param(errMsgInvalidParam)
	}

	if word == updateWord {
		return nil
	}

	target, err := s.repo.FindByWord(ctx, word)
	if err != nil {
		return bizerr.InternalWrap("更新敏感词失败", err)
	}
	if target == nil {
		return ErrSensitiveWordNotFound
	}

	exists, err := s.repo.FindByWord(ctx, updateWord)
	if err != nil {
		return bizerr.InternalWrap("更新敏感词失败", err)
	}
	if exists != nil && exists.ID != target.ID {
		return ErrSensitiveWordExists
	}

	updated, err := s.repo.UpdateByWord(ctx, word, updateWord)
	if err != nil {
		return bizerr.InternalWrap("更新敏感词失败", err)
	}
	if !updated {
		return ErrSensitiveWordNotFound
	}

	s.invalidateCache()
	return nil
}

func (s *Service) FilterText(ctx context.Context, content string) (string, error) {
	if content == "" {
		return content, nil
	}

	pattern, err := s.filterPattern(ctx)
	if err != nil {
		return "", bizerr.InternalWrap("过滤敏感词失败", err)
	}
	if pattern == nil {
		return content, nil
	}

	return pattern.ReplaceAllStringFunc(content, func(match string) string {
		return strings.Repeat("*", utf8.RuneCountInString(match))
	}), nil
}

func (s *Service) filterPattern(ctx context.Context) (*regexp.Regexp, error) {
	now := time.Now()

	s.cacheMu.RLock()
	if now.Before(s.cacheExpiresAt) {
		pattern := s.cachePattern
		s.cacheMu.RUnlock()
		return pattern, nil
	}
	s.cacheMu.RUnlock()

	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()

	if now.Before(s.cacheExpiresAt) {
		return s.cachePattern, nil
	}

	words, err := s.repo.FindAll(ctx)
	if err != nil {
		return nil, err
	}

	pattern := buildSensitivePattern(extractWords(words))
	s.cachePattern = pattern
	s.cacheExpiresAt = now.Add(filterCacheTTL)
	return s.cachePattern, nil
}

func (s *Service) invalidateCache() {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()

	s.cachePattern = nil
	s.cacheExpiresAt = time.Time{}
}

func extractWords(list []SensitiveWord) []string {
	words := make([]string, 0, len(list))
	for _, item := range list {
		words = append(words, item.Word)
	}
	return words
}

func normalizeWordList(words []string) []string {
	if len(words) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(words))
	normalized := make([]string, 0, len(words))
	for _, word := range words {
		word = strings.TrimSpace(word)
		if word == "" {
			continue
		}
		if _, ok := seen[word]; ok {
			continue
		}
		seen[word] = struct{}{}
		normalized = append(normalized, word)
	}
	return normalized
}

func buildSensitivePattern(words []string) *regexp.Regexp {
	normalized := normalizeWordList(words)
	if len(normalized) == 0 {
		return nil
	}

	sort.SliceStable(normalized, func(i, j int) bool {
		li := utf8.RuneCountInString(normalized[i])
		lj := utf8.RuneCountInString(normalized[j])
		if li != lj {
			return li > lj
		}
		return normalized[i] < normalized[j]
	})

	escaped := make([]string, 0, len(normalized))
	for _, word := range normalized {
		escaped = append(escaped, regexp.QuoteMeta(word))
	}

	return regexp.MustCompile(strings.Join(escaped, "|"))
}
