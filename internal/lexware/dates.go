package lexware

import (
	"fmt"
	"strings"
	"time"

	// Embed the time zone database so Europe/Berlin resolves on Windows and
	// in minimal container images without zoneinfo.
	_ "time/tzdata"
)

// lexwareTimestampLayout is the timestamp format documented by Lexware,
// for example 2026-04-07T00:00:00.000+02:00.
const lexwareTimestampLayout = "2006-01-02T15:04:05.000-07:00"

const dateOnlyLayout = "2006-01-02"

var berlin = mustLoadLocation("Europe/Berlin")

func mustLoadLocation(name string) *time.Location {
	location, err := time.LoadLocation(name)
	if err != nil {
		panic(fmt.Sprintf("load time zone %s: %v", name, err))
	}
	return location
}

// NormalizeDate converts a YYYY-MM-DD date (interpreted as midnight in
// Europe/Berlin) or an RFC 3339 timestamp into the Lexware timestamp format.
// An empty value stays empty.
func NormalizeDate(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", nil
	}

	if parsed, err := time.ParseInLocation(dateOnlyLayout, trimmed, berlin); err == nil {
		return parsed.Format(lexwareTimestampLayout), nil
	}
	if parsed, err := time.Parse(time.RFC3339Nano, trimmed); err == nil {
		return parsed.Format(lexwareTimestampLayout), nil
	}

	return "", fmt.Errorf("invalid date %q: use YYYY-MM-DD or an RFC 3339 timestamp", value)
}

// Today returns the current date in Europe/Berlin as YYYY-MM-DD.
func Today() string {
	return time.Now().In(berlin).Format(dateOnlyLayout)
}

func validateDateOnly(field, value string) error {
	if value == "" {
		return nil
	}
	if _, err := time.Parse(dateOnlyLayout, value); err != nil {
		return fmt.Errorf("%s must use the format YYYY-MM-DD, got %q", field, value)
	}
	return nil
}
