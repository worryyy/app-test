package user

import "testing"

func TestAvatarURL(t *testing.T) {
	const cdn = "https://cdn.fangfangfang.top/"

	cases := []struct {
		name    string
		baseCDN string
		avatar  string
		want    string
	}{
		{"裸key拼CDN前缀", cdn, "5e39bbaac50af63e6d221b4f9e7fbee8", "https://cdn.fangfangfang.top/5e39bbaac50af63e6d221b4f9e7fbee8"},
		{"CDN无尾斜杠也补全", "https://cdn.fangfangfang.top", "abc", "https://cdn.fangfangfang.top/abc"},
		{"已是完整URL原样返回", cdn, "https://fangfangfang.top/file/abc", "https://fangfangfang.top/file/abc"},
		{"空值保持空", cdn, "", ""},
		{"未配置CDN时保留裸key", "", "abc", "abc"},
		{"首尾空白被裁剪", cdn, "  abc  ", "https://cdn.fangfangfang.top/abc"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := avatarURL(tc.baseCDN, tc.avatar); got != tc.want {
				t.Fatalf("avatarURL(%q, %q) = %q, want %q", tc.baseCDN, tc.avatar, got, tc.want)
			}
		})
	}
}
