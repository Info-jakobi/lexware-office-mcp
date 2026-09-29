package server

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/dennis/lexware-office-mcp/internal/lexware"
)

func connect(t *testing.T, handler http.HandlerFunc) *mcp.ClientSession {
	t.Helper()

	api := httptest.NewServer(handler)
	t.Cleanup(api.Close)

	client := lexware.NewClient(lexware.Config{
		APIToken:    "test-token",
		BaseURL:     api.URL,
		UserAgent:   "lexware-office-mcp-test",
		HTTPTimeout: 5 * time.Second,
	})

	ctx := context.Background()
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverSession, err := New(client).Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("server connect: %v", err)
	}
	t.Cleanup(func() { _ = serverSession.Close() })

	session, err := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "0.0.0"}, nil).Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	t.Cleanup(func() { _ = session.Close() })
	return session
}

func textOf(result *mcp.CallToolResult) string {
	var parts []string
	for _, content := range result.Content {
		if text, ok := content.(*mcp.TextContent); ok {
			parts = append(parts, text.Text)
		}
	}
	return strings.Join(parts, "\n")
}

func TestToolSurface(t *testing.T) {
	session := connect(t, func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected Lexware request %s %s", r.Method, r.URL.Path)
	})

	result, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}

	var names []string
	for _, tool := range result.Tools {
		names = append(names, tool.Name)
	}
	slices.Sort(names)

	want := []string{
		"lexware_create_contact",
		"lexware_create_invoice",
		"lexware_create_invoice_from_voucher",
		"lexware_create_order_confirmation",
		"lexware_create_order_confirmation_from_quotation",
		"lexware_create_quotation",
		"lexware_create_simple_contact",
		"lexware_get_contact",
		"lexware_get_down_payment_invoice",
		"lexware_get_invoice",
		"lexware_get_order_confirmation",
		"lexware_get_profile",
		"lexware_get_quotation",
		"lexware_list_payment_conditions",
		"lexware_list_vouchers",
		"lexware_search_articles",
		"lexware_search_contacts",
	}
	if !slices.Equal(names, want) {
		t.Fatalf("tools = %v\nwant %v", names, want)
	}
}

func TestCreateQuotationValidationErrorIsReportedAsToolError(t *testing.T) {
	var calls atomic.Int32
	session := connect(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
	})

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "lexware_create_quotation",
		Arguments: map[string]any{
			"quotation": map[string]any{
				"voucherDate":    "2026-09-29",
				"expirationDate": "2026-10-29",
				"address":        map[string]any{"name": "Sample Customer", "countryCode": "DE"},
				"lineItems": []any{
					map[string]any{"type": "custom", "name": "Floor tiling", "unitName": "m²", "unitPrice": map[string]any{"netAmount": 58.0, "taxRatePercentage": 19}},
				},
				"taxConditions": map[string]any{"taxType": "net"},
			},
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !result.IsError || !strings.Contains(textOf(result), "quantity is required") {
		t.Fatalf("expected quantity validation error, got %+v", result)
	}
	if calls.Load() != 0 {
		t.Fatalf("invalid quotation must not reach Lexware")
	}
}

func TestListVouchersReturnsStructuredResult(t *testing.T) {
	session := connect(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/voucherlist" || r.URL.Query().Get("voucherStatus") != "open,overdue" {
			t.Errorf("unexpected request %s", r.URL)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"content":[{"id":"v-1","voucherType":"invoice","voucherStatus":"overdue","voucherNumber":"RE0815","voucherDate":"2026-08-01T00:00:00.000+02:00","totalAmount":2380,"openAmount":1190,"currency":"EUR","archived":false}],"totalPages":1,"totalElements":1,"size":25,"number":0}`)
	})

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "lexware_list_vouchers",
		Arguments: map[string]any{
			"voucherTypes":    []string{"invoice"},
			"voucherStatuses": []string{"open", "overdue"},
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %s", textOf(result))
	}

	raw, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatalf("marshal structured content: %v", err)
	}
	var payload struct {
		Result lexware.Page[lexware.VoucherListItem] `json:"result"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("unmarshal structured content: %v", err)
	}
	if len(payload.Result.Content) != 1 || payload.Result.Content[0].OpenAmount != 1190 {
		t.Fatalf("unexpected payload %s", raw)
	}
}
