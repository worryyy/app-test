package user

import (
	"strconv"
	"strings"
)

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

func (f AdminListUsersFilter) SearchKeyword() string {
	return strings.TrimSpace(f.Nickname)
}

func (f AdminListUsersFilter) SearchUserID() int64 {
	keyword := f.SearchKeyword()
	if keyword == "" {
		return 0
	}

	id, err := strconv.ParseInt(keyword, 10, 64)
	if err != nil || id <= 0 {
		return 0
	}
	return id
}

func (q adminListUsersQuery) Filter() AdminListUsersFilter {
	return AdminListUsersFilter{
		ID:       q.ID,
		StuNum:   strings.TrimSpace(q.StuNum),
		Nickname: strings.TrimSpace(q.NickName),
	}
}
