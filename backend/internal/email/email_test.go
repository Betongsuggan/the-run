package email

import (
	"context"
	"strings"
	"testing"
)

func TestNoOpSender_Succeeds(t *testing.T) {
	err := NoOpSender{}.Send(context.Background(), Message{
		To:       "anyone@example.com",
		Subject:  "test",
		TextBody: "hi",
	})
	if err != nil {
		t.Fatalf("NoOp send: %v", err)
	}
}

func TestNewSenderFromEnv_ReturnsNoOpWhenUnset(t *testing.T) {
	t.Setenv("SES_SENDER_ADDRESS", "")
	t.Setenv("SES_CONFIGURATION_SET", "")
	s, ses, err := NewSenderFromEnv(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ses {
		t.Errorf("expected NoOp, got SES sender")
	}
	if _, ok := s.(NoOpSender); !ok {
		t.Errorf("expected NoOpSender, got %T", s)
	}
}

func TestSESSender_RejectsEmptyFields(t *testing.T) {
	s := &SESSender{cfg: Config{SenderAddress: "x", ConfigurationSet: "y"}}
	cases := []Message{
		{To: "", Subject: "s", TextBody: "b"},
		{To: "to@x", Subject: "", TextBody: "b"},
		{To: "to@x", Subject: "s", TextBody: ""},
	}
	for i, m := range cases {
		err := s.Send(context.Background(), m)
		if err == nil || !strings.Contains(err.Error(), "is required") {
			t.Errorf("case %d: want 'is required' error, got %v", i, err)
		}
	}
}
