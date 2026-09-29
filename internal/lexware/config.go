package lexware

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultBaseURL   = "https://api.lexware.io"
	defaultUserAgent = "lexware-office-mcp/0.1.0"

	// Lexware allows two requests per second per client.
	defaultMinRequestInterval = 500 * time.Millisecond
)

type Config struct {
	APIToken           string
	BaseURL            string
	UserAgent          string
	HTTPTimeout        time.Duration
	MinRequestInterval time.Duration
	FinalizeInvoices   bool
}

func LoadConfigFromEnv() (Config, error) {
	token := strings.TrimSpace(os.Getenv("LEXWARE_API_TOKEN"))
	if token == "" {
		return Config{}, fmt.Errorf("LEXWARE_API_TOKEN is required")
	}

	baseURL := strings.TrimSpace(os.Getenv("LEXWARE_BASE_URL"))
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	userAgent := strings.TrimSpace(os.Getenv("LEXWARE_USER_AGENT"))
	if userAgent == "" {
		userAgent = defaultUserAgent
	}

	return Config{
		APIToken:           token,
		BaseURL:            strings.TrimRight(baseURL, "/"),
		UserAgent:          userAgent,
		HTTPTimeout:        30 * time.Second,
		MinRequestInterval: defaultMinRequestInterval,
		FinalizeInvoices:   parseBoolEnv("LEXWARE_FINALIZE_INVOICES", false),
	}, nil
}

func parseBoolEnv(key string, fallback bool) bool {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}

	value, err := strconv.ParseBool(raw)
	if err != nil {
		return fallback
	}

	return value
}
