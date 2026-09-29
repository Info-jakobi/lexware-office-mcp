package lexware

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
)

type Client struct {
	restClient       *resty.Client
	finalizeInvoices bool
}

func NewClient(cfg Config) *Client {
	pacer := &requestPacer{interval: cfg.MinRequestInterval}

	restClient := resty.New().
		SetBaseURL(cfg.BaseURL).
		SetTimeout(cfg.HTTPTimeout).
		SetHeader("Authorization", "Bearer "+cfg.APIToken).
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json").
		SetHeader("User-Agent", cfg.UserAgent).
		SetRetryCount(5).
		SetRetryWaitTime(10 * time.Second).
		SetRetryMaxWaitTime(60 * time.Second).
		AddRetryCondition(func(resp *resty.Response, err error) bool {
			return err == nil && resp != nil && resp.StatusCode() == http.StatusTooManyRequests
		}).
		// Pace every attempt, retries included, so bursts of tool calls stay
		// within the Lexware limit instead of relying on 429 retries.
		OnBeforeRequest(func(_ *resty.Client, req *resty.Request) error {
			return pacer.wait(req.Context())
		})

	return &Client{
		restClient:       restClient,
		finalizeInvoices: cfg.FinalizeInvoices,
	}
}

func wrapAPIError(action string, resp *resty.Response, err error) error {
	if err != nil {
		return fmt.Errorf("%s request failed: %w", action, err)
	}
	if resp == nil {
		return fmt.Errorf("%s failed without a response", action)
	}
	if resp.StatusCode() < http.StatusBadRequest {
		return nil
	}

	body := strings.TrimSpace(resp.String())
	if body == "" {
		return fmt.Errorf("%s failed with status %d", action, resp.StatusCode())
	}

	return fmt.Errorf("%s failed with status %d: %s", action, resp.StatusCode(), body)
}

func (c *Client) FinalizeInvoices() bool {
	return c.finalizeInvoices
}

func (c *Client) newRequest(ctx context.Context) *resty.Request {
	return c.restClient.R().SetContext(ctx)
}

// requestPacer spaces outgoing requests by a minimum interval.
type requestPacer struct {
	mu       sync.Mutex
	interval time.Duration
	next     time.Time
}

func (p *requestPacer) wait(ctx context.Context) error {
	if p.interval <= 0 {
		return nil
	}

	p.mu.Lock()
	now := time.Now()
	slot := p.next
	if slot.Before(now) {
		slot = now
	}
	p.next = slot.Add(p.interval)
	p.mu.Unlock()

	delay := time.Until(slot)
	if delay <= 0 {
		return nil
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
