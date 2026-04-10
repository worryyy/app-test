package topic

import (
	"testing"

	"go.mongodb.org/mongo-driver/bson"
)

func TestBuildAdminTopicMarksTopicCheckedAndKeepsRawContent(t *testing.T) {
	author := &topicAuthor{
		ID:          42,
		Nickname:    "admin",
		Avatar:      "avatar.png",
		AccountType: topicAccountTypeBase,
	}
	req := &CreateTopicReq{
		ThemeID:     "theme-1",
		Title:       "",
		Content:     "raw sensitive text",
		Imgs:        []string{"a.png"},
		Ext:         map[string]any{"k": "v"},
		AccountType: topicAccountTypeBase,
	}

	topic := buildAdminTopic(author, req)

	if !topic.HasCheck {
		t.Fatal("expected admin topic to be immediately visible")
	}
	if topic.Title != " " {
		t.Fatalf("expected blank title to fall back to a single space, got %q", topic.Title)
	}
	if topic.Content != req.Content {
		t.Fatalf("expected content to stay unfiltered, got %q want %q", topic.Content, req.Content)
	}
	if topic.UserID != "42" {
		t.Fatalf("unexpected user id: %s", topic.UserID)
	}
	if len(topic.Imgs) != 1 || topic.Imgs[0] != "a.png" {
		t.Fatalf("unexpected imgs: %#v", topic.Imgs)
	}
}

func TestBuildAdminTopicUpdateIgnoresHasCheckChanges(t *testing.T) {
	hasCheck := false
	ext := ""
	req := &UpdateTopicReq{
		HasCheck: &hasCheck,
		Ext:      &ext,
	}

	update := buildAdminTopicUpdate(req)

	if len(update) != 0 {
		t.Fatalf("expected empty update when only hasCheck/empty ext provided, got %#v", update)
	}
}

func TestBuildAdminTopicUpdateIncludesPatchableFieldsOnly(t *testing.T) {
	hasCheck := false
	ext := "meta"
	req := &UpdateTopicReq{
		Title:    "new title",
		Content:  "new content",
		Imgs:     []string{"1.png", "2.png"},
		Ext:      &ext,
		HasCheck: &hasCheck,
	}

	update := buildAdminTopicUpdate(req)

	if got := update["title"]; got != "new title" {
		t.Fatalf("unexpected title update: %#v", got)
	}
	if got := update["content"]; got != "new content" {
		t.Fatalf("unexpected content update: %#v", got)
	}
	imgs, ok := update["imgs"].([]string)
	if !ok || len(imgs) != 2 {
		t.Fatalf("unexpected imgs update: %#v", update["imgs"])
	}
	if got := update["ext"]; got != "meta" {
		t.Fatalf("unexpected ext update: %#v", got)
	}
	if _, ok := update["hasCheck"]; ok {
		t.Fatalf("hasCheck should not be patchable in admin update: %#v", update)
	}
}

func TestBuildAdminTopicUpdateReturnsBSONMap(t *testing.T) {
	update := buildAdminTopicUpdate(&UpdateTopicReq{Title: "t"})
	if _, ok := any(update).(bson.M); !ok {
		t.Fatalf("expected bson.M, got %T", update)
	}
}
