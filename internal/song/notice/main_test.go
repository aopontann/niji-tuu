package songnotice

import "testing"

func TestNotifyFromDiscord(t *testing.T) {
	err := NotifyFromDiscord("TpGxDY4YmAI")
	if err != nil {
		t.Errorf("expected error, got %v", err)
	}
}
