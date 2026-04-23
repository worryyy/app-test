package agentchat

import (
	"errors"
	"testing"

	"github.com/Milchstrassse/Ecampus-go/internal/platform/bizerr"
)

func TestSafeWSErrorMessageUsesBizMessage(t *testing.T) {
	err := bizerr.InternalWrap("保存 agent 会话失败", errors.New("dial tcp 10.0.0.5:3306: connect: connection refused"))

	if got := safeWSErrorMessage(err); got != "保存 agent 会话失败" {
		t.Fatalf("safeWSErrorMessage() = %q, want 保存 agent 会话失败", got)
	}
}

func TestSafeWSErrorMessageUsesGenericMessageForNonBizError(t *testing.T) {
	if got := safeWSErrorMessage(errors.New("dial tcp 10.0.0.5:3306: connect: connection refused")); got != "请求处理失败" {
		t.Fatalf("safeWSErrorMessage() = %q, want 请求处理失败", got)
	}
}

func TestSessionTryStartTurnHonorsLimit(t *testing.T) {
	session := &Session{}

	if !session.tryStartTurn(2) {
		t.Fatalf("first tryStartTurn should succeed")
	}
	if !session.tryStartTurn(2) {
		t.Fatalf("second tryStartTurn should succeed")
	}
	if session.tryStartTurn(2) {
		t.Fatalf("third tryStartTurn should be rejected")
	}

	session.finishTurn()
	if !session.tryStartTurn(2) {
		t.Fatalf("tryStartTurn should succeed after finishTurn")
	}
}

func TestWSAuthFailedEventUsesFixedMessage(t *testing.T) {
	event := wsAuthFailedEvent()
	if event.Message != "鉴权失败" {
		t.Fatalf("message = %q, want 鉴权失败", event.Message)
	}
}
