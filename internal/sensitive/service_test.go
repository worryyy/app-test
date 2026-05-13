package sensitive

import (
	"context"
	"strings"
	"sync"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func newTestFilter(words []string) *Service {
	s := &Service{
		logger: zap.NewNop(),
		stop:   make(chan struct{}),
	}
	f := buildFilter(words)
	s.filter.Store(f)
	return s
}

func TestFilterText_ExactMatch(t *testing.T) {
	s := newTestFilter([]string{"赌博", "色情"})
	got, err := s.FilterText(context.Background(), "赌博是违法的")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "**") {
		t.Fatalf("expected masked output, got %q", got)
	}
	if strings.Contains(got, "赌博") {
		t.Fatalf("sensitive word should be masked, got %q", got)
	}
}

func TestFilterText_LongestMatch(t *testing.T) {
	s := newTestFilter([]string{"中国", "中国共产党"})
	got, err := s.FilterText(context.Background(), "中国共产党万岁")
	if err != nil {
		t.Fatal(err)
	}
	// "中国共产党" (5 runes) should be masked, not just "中国" (2 runes)
	if !strings.Contains(got, "*****") {
		t.Fatalf("expected longest match masked (5 stars), got %q", got)
	}
}

func TestFilterText_SpaceBypass(t *testing.T) {
	s := newTestFilter([]string{"赌博"})
	got, err := s.FilterText(context.Background(), "赌 博是违法的")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, "赌") || strings.Contains(got, "博") {
		t.Fatalf("space bypass should be detected, got %q", got)
	}
}

func TestFilterText_ZeroWidthBypass(t *testing.T) {
	s := newTestFilter([]string{"赌博"})
	input := "赌​博是违法的"
	got, err := s.FilterText(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, "赌") || strings.Contains(got, "博") {
		t.Fatalf("zero-width bypass should be detected, got %q", got)
	}
}

func TestFilterText_CaseBypass(t *testing.T) {
	s := newTestFilter([]string{"fuck"})
	cases := []string{"FUCK", "Fuck", "fuck", "fUcK"}
	for _, input := range cases {
		got, err := s.FilterText(context.Background(), input)
		if err != nil {
			t.Fatal(err)
		}
		if got != "****" {
			t.Fatalf("case bypass %q should be masked to ****, got %q", input, got)
		}
	}
}

func TestFilterText_FullWidthBypass(t *testing.T) {
	s := newTestFilter([]string{"fuck"})
	input := "ｆｕｃｋ"
	got, err := s.FilterText(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if got != "****" {
		t.Fatalf("full-width bypass should be masked to ****, got %q", got)
	}
}

func TestFilterText_NoiseCharBypass(t *testing.T) {
	s := newTestFilter([]string{"赌博"})
	cases := []string{"赌*博", "赌@博", "赌$博"}
	for _, input := range cases {
		got, err := s.FilterText(context.Background(), input)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(got, "赌") || strings.Contains(got, "博") {
			t.Fatalf("noise char bypass %q should be detected, got %q", input, got)
		}
	}
}

func TestFilterText_NoMatch(t *testing.T) {
	s := newTestFilter([]string{"赌博"})
	original := "这是正常内容"
	got, err := s.FilterText(context.Background(), original)
	if err != nil {
		t.Fatal(err)
	}
	if got != original {
		t.Fatalf("no-match should return original, got %q want %q", got, original)
	}
}

func TestFilterText_EmptyContent(t *testing.T) {
	s := newTestFilter([]string{"赌博"})
	got, err := s.FilterText(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Fatalf("empty content should passthrough, got %q", got)
	}
}

func TestFilterText_NilFilter(t *testing.T) {
	s := &Service{logger: zap.NewNop(), stop: make(chan struct{})}
	got, err := s.FilterText(context.Background(), "赌博")
	if err != nil {
		t.Fatal(err)
	}
	if got != "赌博" {
		t.Fatalf("nil filter should passthrough, got %q", got)
	}
}

func TestFilterText_ConcurrentSafety(t *testing.T) {
	s := newTestFilter([]string{"赌博", "色情"})
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = s.FilterText(context.Background(), "赌博和色情")
		}()
	}
	wg.Wait()
}

func TestFilterText_HitLogging(t *testing.T) {
	core, logs := observer.New(zapcore.InfoLevel)
	s := &Service{
		logger: zap.New(core),
		stop:   make(chan struct{}),
	}
	f := buildFilter([]string{"赌博"})
	s.filter.Store(f)

	_, err := s.FilterText(context.Background(), "赌博是违法的")
	if err != nil {
		t.Fatal(err)
	}

	entries := logs.FilterMessage("sensitive_word_hit").All()
	if len(entries) == 0 {
		t.Fatal("expected sensitive_word_hit log entry")
	}
}

func TestFilterText_PreservesSurroundingCase(t *testing.T) {
	s := newTestFilter([]string{"赌博"})
	got, err := s.FilterText(context.Background(), "Hello 赌博 World")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "Hello") || !strings.Contains(got, "World") {
		t.Fatalf("surrounding text should preserve original case, got %q", got)
	}
	if strings.Contains(got, "赌博") {
		t.Fatalf("sensitive word should be masked, got %q", got)
	}
}

func TestNormalizeWordList(t *testing.T) {
	got := normalizeWordList([]string{"  spam  ", "", "spam", "ham"})
	if len(got) != 2 {
		t.Fatalf("unexpected length: %d", len(got))
	}
	if got[0] != "spam" || got[1] != "ham" {
		t.Fatalf("unexpected words: %#v", got)
	}
}

func TestNormalizePageSize(t *testing.T) {
	page, size := normalizePageSize(0, -1)
	if page != 1 {
		t.Fatalf("page = %d, want 1", page)
	}
	if size != 15 {
		t.Fatalf("size = %d, want 15", size)
	}
}

func TestBuildFilter_EmptyWords(t *testing.T) {
	f := buildFilter(nil)
	if f != nil {
		t.Fatal("expected nil filter for empty words")
	}
	f = buildFilter([]string{"", "  "})
	if f != nil {
		t.Fatal("expected nil filter for blank words")
	}
}

func TestNormalizeText(t *testing.T) {
	got := normalizeText("ＨＥＬＬＯ World")
	if got != "hello world" {
		t.Fatalf("expected normalized, got %q", got)
	}
}

func TestStripInvisible(t *testing.T) {
	got := stripInvisible("hello​world‌")
	if got != "helloworld" {
		t.Fatalf("expected invisible chars stripped, got %q", got)
	}
}

func TestTruncate(t *testing.T) {
	got := truncate("abcdef", 3)
	if got != "abc..." {
		t.Fatalf("expected truncated, got %q", got)
	}
	got = truncate("ab", 3)
	if got != "ab" {
		t.Fatalf("expected no truncation, got %q", got)
	}
}
