package sensitive

import "testing"

func TestBuildSensitivePatternPrefersLongerWords(t *testing.T) {
	pattern := buildSensitivePattern([]string{"ab", "abc", "a.c"})
	if pattern == nil {
		t.Fatal("expected pattern")
	}

	got := pattern.ReplaceAllStringFunc("zzabc a.c ab", func(match string) string {
		return repeatMask(match)
	})
	want := "zz*** *** **"
	if got != want {
		t.Fatalf("unexpected filtered result: got %q want %q", got, want)
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

func repeatMask(value string) string {
	return buildSensitivePattern([]string{value}).ReplaceAllStringFunc(value, func(match string) string {
		return repeatAsterisk(match)
	})
}

func repeatAsterisk(value string) string {
	mask := ""
	for range value {
		mask += "*"
	}
	return mask
}
