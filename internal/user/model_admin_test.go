package user

import "testing"

func TestAdminListUsersQueryFilterUsesPreferredFields(t *testing.T) {
	query := adminListUsersQuery{
		Page:           2,
		Size:           20,
		ID:             1001,
		StuNum:         "20230001",
		Nickname:       "new-name",
		LegacyNickName: "legacy-name",
	}

	filter := query.Filter()
	if filter.ID != 1001 {
		t.Fatalf("unexpected filter id: %d", filter.ID)
	}
	if filter.StuNum != "20230001" {
		t.Fatalf("unexpected filter stu num: %s", filter.StuNum)
	}
	if filter.Nickname != "new-name" {
		t.Fatalf("expected nickname to prefer new field, got %s", filter.Nickname)
	}
}

func TestAdminListUsersQueryFilterFallsBackToLegacyNickName(t *testing.T) {
	filter := (adminListUsersQuery{
		LegacyNickName: "legacy-name",
	}).Filter()

	if filter.Nickname != "legacy-name" {
		t.Fatalf("expected legacy nickname fallback, got %s", filter.Nickname)
	}
}
