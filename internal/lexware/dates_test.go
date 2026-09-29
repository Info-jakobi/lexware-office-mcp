package lexware

import (
	"context"
	"testing"
	"time"
)

func TestNormalizeDate(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"empty", "", ""},
		{"summer date uses CEST", "2026-04-07", "2026-04-07T00:00:00.000+02:00"},
		{"winter date uses CET", "2026-01-15", "2026-01-15T00:00:00.000+01:00"},
		{"RFC 3339 keeps offset", "2026-04-07T10:30:00+02:00", "2026-04-07T10:30:00.000+02:00"},
		{"UTC is written as offset", "2026-04-07T08:30:00Z", "2026-04-07T08:30:00.000+00:00"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeDate(tt.input)
			if err != nil {
				t.Fatalf("NormalizeDate(%q) returned error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Fatalf("NormalizeDate(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeDateRejectsInvalidInput(t *testing.T) {
	for _, input := range []string{"07.04.2026", "2026-13-01", "tomorrow"} {
		if _, err := NormalizeDate(input); err == nil {
			t.Errorf("NormalizeDate(%q) expected an error", input)
		}
	}
}

func TestRequestPacerSpacesRequests(t *testing.T) {
	pacer := &requestPacer{interval: 40 * time.Millisecond}
	start := time.Now()
	for range 3 {
		if err := pacer.wait(context.Background()); err != nil {
			t.Fatalf("wait returned error: %v", err)
		}
	}
	if elapsed := time.Since(start); elapsed < 80*time.Millisecond {
		t.Fatalf("three paced requests took %v, want at least 80ms", elapsed)
	}
}

func TestRequestPacerHonorsContextCancellation(t *testing.T) {
	pacer := &requestPacer{interval: time.Hour}
	if err := pacer.wait(context.Background()); err != nil {
		t.Fatalf("first wait returned error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := pacer.wait(ctx); err == nil {
		t.Fatal("expected cancelled context to abort the wait")
	}
}
