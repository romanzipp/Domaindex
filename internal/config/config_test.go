package config_test

import (
	"testing"

	"github.com/romanzipp/domaindex/internal/config"
)

func TestNotificationURLs(t *testing.T) {
	t.Setenv("NOTIFICATION_URLS", " pushover://shoutrrr:token@userkey/ , discord://token@id ,, ")

	got := config.Load().NotificationURLs
	want := []string{"pushover://shoutrrr:token@userkey/", "discord://token@id"}

	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}

	for i := range want {
		if got[i] != want[i] {
			t.Errorf("url %d: got %q, want %q", i, got[i], want[i])
		}
	}
}
