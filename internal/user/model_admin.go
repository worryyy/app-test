package user

import "strings"

type AdminListUsersFilter struct {
	ID       int64
	StuNum   string
	Nickname string
}

type AdminBatchPreAuthReq struct {
	Password string             `json:"password" binding:"required"`
	Items    []AdminPreAuthItem `json:"items"`
}

type AdminPreAuthItem struct {
	UserID   int64  `json:"userId"`
	NickName string `json:"nickName"`
}

type AdminBatchPreAuthResp struct {
	Total        int                  `json:"total"`
	SuccessCount int                  `json:"successCount"`
	FailureCount int                  `json:"failureCount"`
	Results      []AdminPreAuthResult `json:"results"`
}

type AdminPreAuthResult struct {
	UserID  int64  `json:"userId"`
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func (q adminListUsersQuery) Filter() AdminListUsersFilter {
	nickname := strings.TrimSpace(q.Nickname)
	if nickname == "" {
		nickname = strings.TrimSpace(q.LegacyNickName)
	}
	return AdminListUsersFilter{
		ID:       q.ID,
		StuNum:   strings.TrimSpace(q.StuNum),
		Nickname: nickname,
	}
}
