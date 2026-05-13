package sensitive

import (
	"context"
	"regexp"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode"
	"unicode/utf8"

	sensitivelib "github.com/importcjj/sensitive"
	"go.uber.org/zap"
	"golang.org/x/text/unicode/norm"
	"gorm.io/gorm"

	"github.com/Milchstrassse/Ecampus-go/internal/platform/bizerr"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/pagination"
)

const filterCacheTTL = 30 * time.Second

type Filter interface {
	FilterText(ctx context.Context, content string) (string, error)
}

const maxMaskHits = 30

type Service struct {
	repo      *Repository
	logger    *zap.Logger
	filter    atomic.Pointer[sensitivelib.Filter]
	stop      chan struct{}
	stopOnce  sync.Once
	rebuildMu sync.Mutex
}

func NewService(db *gorm.DB, logger *zap.Logger) *Service {
	if logger == nil {
		logger = zap.NewNop()
	}
	s := &Service{
		repo:   NewRepository(db),
		logger: logger,
		stop:   make(chan struct{}),
	}
	s.rebuildFilter()
	go s.reloadLoop()
	return s
}

func (s *Service) Close() {
	s.stopOnce.Do(func() { close(s.stop) })
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

func (s *Service) FindByPage(ctx context.Context, page, size int) (*pagination.PageResult[SensitiveWord], error) {
	page, size = normalizePageSize(page, size)

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

func (s *Service) FilterText(ctx context.Context, content string) (string, error) {
	if content == "" {
		return content, nil
	}
	f := s.filter.Load()
	if f == nil {
		return content, nil
	}

	normalized := normalizeText(content)
	cleaned := f.RemoveNoise(normalized)

	hits := f.FindAll(cleaned)
	if len(hits) == 0 {
		return content, nil
	}
	if len(hits) > maxMaskHits {
		hits = hits[:maxMaskHits]
	}

	s.logger.Info("sensitive_word_hit",
		zap.Strings("words", hits),
		zap.String("content_preview", truncate(content, 100)))

	// Mask on NFKC + invisible-stripped text: preserves case + spaces
	displayText := stripInvisible(norm.NFKC.String(content))
	return maskHits(displayText, hits), nil
}

func maskHits(text string, hitWords []string) string {
	sort.Slice(hitWords, func(i, j int) bool {
		return utf8.RuneCountInString(hitWords[i]) > utf8.RuneCountInString(hitWords[j])
	})

	seen := make(map[string]struct{}, len(hitWords))
	noiseGap := `[\s!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?]*`

	for _, word := range hitWords {
		if _, ok := seen[word]; ok {
			continue
		}
		seen[word] = struct{}{}

		runes := []rune(word)
		parts := make([]string, len(runes))
		for i, r := range runes {
			parts[i] = regexp.QuoteMeta(string(r))
		}
		pattern := "(?i)" + strings.Join(parts, noiseGap)
		re, err := regexp.Compile(pattern)
		if err != nil {
			continue
		}
		text = re.ReplaceAllStringFunc(text, func(match string) string {
			return strings.Repeat("*", utf8.RuneCountInString(match))
		})
	}
	return text
}

func (s *Service) reloadLoop() {
	ticker := time.NewTicker(filterCacheTTL)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.rebuildFilter()
		case <-s.stop:
			return
		}
	}
}

func (s *Service) rebuildFilter() {
	s.rebuildMu.Lock()
	defer s.rebuildMu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	words, err := s.repo.FindAll(ctx)
	if err != nil {
		s.logger.Warn("rebuild sensitive filter failed", zap.Error(err))
		return
	}
	raw := make([]string, 0, len(words))
	for _, w := range words {
		raw = append(raw, w.Word)
	}
	f := buildFilter(raw)
	s.filter.Store(f)
}

func (s *Service) invalidateCache() {
	s.rebuildFilter()
}

func buildFilter(words []string) *sensitivelib.Filter {
	normalized := normalizeWordList(words)
	if len(normalized) == 0 {
		return nil
	}

	lowerWords := make([]string, 0, len(normalized))
	for _, w := range normalized {
		lw := strings.ToLower(norm.NFKC.String(w))
		if lw != "" {
			lowerWords = append(lowerWords, lw)
		}
	}
	if len(lowerWords) == 0 {
		return nil
	}

	f := sensitivelib.New()
	f.AddWord(lowerWords...)
	f.UpdateNoisePattern(`[\s!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?]+`)
	return f
}

func normalizeText(s string) string {
	s = norm.NFKC.String(s)
	s = stripInvisible(s)
	s = strings.ToLower(s)
	return s
}

func stripInvisible(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.Is(unicode.Cf, r) || unicode.Is(unicode.Mn, r) {
			return -1
		}
		return r
	}, s)
}

func truncate(s string, maxRunes int) string {
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	return string(runes[:maxRunes]) + "..."
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

func normalizePageSize(page, size int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 15
	}
	return page, size
}
