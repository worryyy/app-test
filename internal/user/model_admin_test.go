package user

import "testing"

func TestAdminListUsersQueryFilterUsesPreferredFields(t *testing.T) {
	query := adminListUsersQuery{
		Page:     2,
		Size:     20,
		ID:       1001,
		StuNum:   "20230001",
		NickName: "new-name",
	}

	filter := query.Filter()
	if filter.ID != 1001 {
		t.Fatalf("unexpected filter id: %d", filter.ID)
	}
	if filter.StuNum != "20230001" {
		t.Fatalf("unexpected filter stu num: %s", filter.StuNum)
	}
	if filter.Nickname != "new-name" {
		t.Fatalf("expected nickname from NickName field, got %s", filter.Nickname)
	}
}

func TestAdminListUsersQueryFilterTrimsNickName(t *testing.T) {
	filter := (adminListUsersQuery{
		NickName: " legacy-name ",
	}).Filter()

	if filter.Nickname != "legacy-name" {
		t.Fatalf("expected trimmed nickname, got %s", filter.Nickname)
	}
}

func TestAdminListUsersFilterSearchUserIDFromNickName(t *testing.T) {
	filter := AdminListUsersFilter{
		Nickname: " 12345 ",
	}

	if got := filter.SearchKeyword(); got != "12345" {
		t.Fatalf("expected trimmed search keyword, got %s", got)
	}
	if got := filter.SearchUserID(); got != 12345 {
		t.Fatalf("expected search user id 12345, got %d", got)
	}
}

func TestAdminListUsersFilterSearchUserIDIgnoresNonNumericNickName(t *testing.T) {
	filter := AdminListUsersFilter{
		Nickname: "2023A001",
	}

	if got := filter.SearchUserID(); got != 0 {
		t.Fatalf("expected non-numeric nickname search to not produce user id, got %d", got)
	}
}
