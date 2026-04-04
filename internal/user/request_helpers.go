package user

import "strings"

func firstNonBlank(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func normalizeUserEditReq(req UserEditReq) UserEditReq {
	if firstNonBlank(req.Nickname) == "" {
		req.Nickname = ""
	}
	if firstNonBlank(req.Avatar) == "" {
		req.Avatar = ""
	}
	if firstNonBlank(req.Gender) == "" {
		req.Gender = ""
	}
	if firstNonBlank(req.Signature) == "" {
		req.Signature = ""
	}
	return req
}
