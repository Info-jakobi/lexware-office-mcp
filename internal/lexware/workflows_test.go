package lexware

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"
)

type recordedRequest struct {
	Method      string
	Path        string
	EscapedPath string
	Query       url.Values
	Body        map[string]any
}

type fakeLexware struct {
	mu        sync.Mutex
	requests  []recordedRequest
	responses map[string]string
}

func newFakeLexware(t *testing.T, responses map[string]string) (*fakeLexware, *Client) {
	t.Helper()

	fake := &fakeLexware{responses: responses}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("Authorization header = %q", got)
		}

		recorded := recordedRequest{Method: r.Method, Path: r.URL.Path, EscapedPath: r.URL.EscapedPath(), Query: r.URL.Query()}
		raw, _ := io.ReadAll(r.Body)
		if len(raw) > 0 {
			if err := json.Unmarshal(raw, &recorded.Body); err != nil {
				t.Errorf("request body is not valid JSON: %v", err)
			}
		}

		fake.mu.Lock()
		fake.requests = append(fake.requests, recorded)
		fake.mu.Unlock()

		body, ok := responses[r.Method+" "+r.URL.Path]
		if !ok {
			http.Error(w, `{"message":"unexpected request"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(server.Close)

	client := NewClient(Config{
		APIToken:    "test-token",
		BaseURL:     server.URL,
		UserAgent:   "lexware-office-mcp-test",
		HTTPTimeout: 5 * time.Second,
	})
	return fake, client
}

func (f *fakeLexware) request(t *testing.T, method, path string) recordedRequest {
	t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, request := range f.requests {
		if request.Method == method && request.Path == path {
			return request
		}
	}
	t.Fatalf("no %s %s request recorded; got %d requests", method, path, len(f.requests))
	return recordedRequest{}
}

func (f *fakeLexware) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.requests)
}

// field walks a decoded JSON document by object keys and array indexes.
func field(t *testing.T, document any, path ...any) any {
	t.Helper()
	current := document
	for _, step := range path {
		switch key := step.(type) {
		case string:
			object, ok := current.(map[string]any)
			if !ok {
				t.Fatalf("expected object at %v, got %T", key, current)
			}
			current = object[key]
		case int:
			array, ok := current.([]any)
			if !ok || key >= len(array) {
				t.Fatalf("expected array with index %d, got %v", key, current)
			}
			current = array[key]
		}
	}
	return current
}

func amount(value float64) *float64 {
	return &value
}

func sampleQuotation() Quotation {
	return Quotation{
		VoucherDate:    "2026-09-29",
		ExpirationDate: "2026-10-29",
		Address: Address{
			Name:        "Sample Customer",
			Street:      "Sample Street 1",
			Zip:         "85049",
			City:        "Ingolstadt",
			CountryCode: "DE",
		},
		LineItems: []LineItem{
			{Type: "text", Name: "Bathroom ground floor"},
			{
				Type:      "custom",
				Name:      "Wall tiling",
				Quantity:  23.45,
				UnitName:  "m²",
				UnitPrice: &UnitPrice{NetAmount: amount(65.5), TaxRatePercentage: 19},
			},
			{
				Type:      "custom",
				Name:      "Niche tiling",
				Quantity:  1,
				UnitName:  "Stk",
				UnitPrice: &UnitPrice{NetAmount: amount(180), TaxRatePercentage: 19},
				Optional:  true,
			},
		},
		TaxConditions: TaxCondition{TaxType: "net"},
	}
}

func TestCreateQuotationSendsNormalizedPayload(t *testing.T) {
	fake, client := newFakeLexware(t, map[string]string{
		"POST /v1/quotations": `{"id":"q-1","resourceUri":"https://api.lexware.io/v1/quotations/q-1","version":1}`,
	})

	quotation := sampleQuotation()
	result, err := client.CreateQuotation(context.Background(), quotation, false)
	if err != nil {
		t.Fatalf("CreateQuotation returned error: %v", err)
	}
	if result.ID != "q-1" || result.Version != 1 {
		t.Fatalf("unexpected result %+v", result)
	}

	request := fake.request(t, http.MethodPost, "/v1/quotations")
	if got := request.Query.Get("finalize"); got != "false" {
		t.Errorf("finalize = %q, want false", got)
	}
	body := request.Body
	if got := field(t, body, "voucherDate"); got != "2026-09-29T00:00:00.000+02:00" {
		t.Errorf("voucherDate = %v", got)
	}
	if got := field(t, body, "expirationDate"); got != "2026-10-29T00:00:00.000+01:00" {
		t.Errorf("expirationDate = %v", got)
	}
	if got := field(t, body, "totalPrice", "currency"); got != "EUR" {
		t.Errorf("totalPrice.currency = %v", got)
	}
	if got := field(t, body, "lineItems", 1, "quantity"); got != 23.45 {
		t.Errorf("decimal quantity = %v", got)
	}
	unitPrice := field(t, body, "lineItems", 1, "unitPrice").(map[string]any)
	if unitPrice["currency"] != "EUR" || unitPrice["netAmount"] != 65.5 {
		t.Errorf("unexpected unit price %v", unitPrice)
	}
	if _, present := unitPrice["grossAmount"]; present {
		t.Errorf("grossAmount must be omitted when only netAmount is given: %v", unitPrice)
	}
	if got := field(t, body, "lineItems", 2, "optional"); got != true {
		t.Errorf("optional flag = %v", got)
	}
	textItem := field(t, body, "lineItems", 0).(map[string]any)
	for _, key := range []string{"quantity", "unitName", "unitPrice"} {
		if _, present := textItem[key]; present {
			t.Errorf("text line item must not contain %s: %v", key, textItem)
		}
	}
	if _, present := body["paymentConditions"]; present {
		t.Errorf("paymentConditions must be omitted when not given")
	}

	if quotation.LineItems[1].UnitPrice.Currency != "" {
		t.Errorf("CreateQuotation must not mutate the caller's line items")
	}
}

func TestQuotationValidation(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*Quotation)
		wantErr string
	}{
		{"missing expiration date", func(q *Quotation) { q.ExpirationDate = "" }, "expirationDate is required"},
		{"expiration before voucher date", func(q *Quotation) { q.ExpirationDate = "2026-09-01" }, "must not be before voucherDate"},
		{"invalid date format", func(q *Quotation) { q.VoucherDate = "29.09.2026" }, "voucherDate"},
		{"address without contact or name", func(q *Quotation) { q.Address = Address{CountryCode: "DE"} }, "contactId or name"},
		{"address without country", func(q *Quotation) { q.Address.CountryCode = "" }, "countryCode"},
		{"unknown tax type", func(q *Quotation) { q.TaxConditions.TaxType = "reverse" }, "taxType"},
		{"no line items", func(q *Quotation) { q.LineItems = nil }, "at least one line item"},
		{"zero quantity placeholder", func(q *Quotation) { q.LineItems[1].Quantity = 0 }, "quantity is required"},
		{"missing unit price", func(q *Quotation) { q.LineItems[1].UnitPrice = nil }, "netAmount or grossAmount"},
		{"missing unit name", func(q *Quotation) { q.LineItems[1].UnitName = "" }, "unitName"},
		{"service without article id", func(q *Quotation) { q.LineItems[1].Type = "service" }, "must reference a Lexware article"},
		{"unknown line item type", func(q *Quotation) { q.LineItems[1].Type = "labour" }, "type must be one of"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			quotation := sampleQuotation()
			tt.mutate(&quotation)
			_, err := quotation.prepare()
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("prepare() error = %v, want it to contain %q", err, tt.wantErr)
			}
		})
	}
}

func TestInvoiceRejectsOptionalLineItemsAndMissingPerformanceDate(t *testing.T) {
	quotation := sampleQuotation()
	invoice := Invoice{
		VoucherDate:   "2026-09-29",
		Address:       quotation.Address,
		LineItems:     quotation.LineItems,
		TaxConditions: quotation.TaxConditions,
		ShippingTerms: ShippingTerm{ShippingType: "serviceperiod", ShippingDate: "2026-09-01", ShippingEndDate: "2026-09-20"},
	}
	if _, err := invoice.prepare(); err == nil || !strings.Contains(err.Error(), "only supported for quotations") {
		t.Fatalf("expected optional line item error, got %v", err)
	}

	invoice.LineItems = quotation.LineItems[:2]
	invoice.ShippingTerms = ShippingTerm{ShippingType: "serviceperiod", ShippingDate: "2026-09-01"}
	if _, err := invoice.prepare(); err == nil || !strings.Contains(err.Error(), "shippingEndDate are required") {
		t.Fatalf("expected missing end date error, got %v", err)
	}

	invoice.ShippingTerms = ShippingTerm{ShippingType: "serviceperiod", ShippingDate: "2026-09-20", ShippingEndDate: "2026-09-01"}
	if _, err := invoice.prepare(); err == nil || !strings.Contains(err.Error(), "must not be before shippingDate") {
		t.Fatalf("expected reversed period error, got %v", err)
	}
}

func TestListVouchersBuildsQuery(t *testing.T) {
	fake, client := newFakeLexware(t, map[string]string{
		"GET /v1/voucherlist": `{"content":[{"id":"v-1","voucherType":"quotation","voucherStatus":"open","voucherNumber":"AG0001","voucherDate":"2026-09-01T00:00:00.000+02:00","contactName":"Sample Customer","totalAmount":1960.2,"openAmount":1960.2,"currency":"EUR","archived":false}],"totalPages":1,"totalElements":1,"size":25,"number":0,"first":true,"last":true}`,
	})

	result, err := client.ListVouchers(context.Background(), VoucherListFilter{
		VoucherTypes:    []string{"quotation", " invoice "},
		VoucherStatuses: []string{"open"},
		ContactID:       "c-1",
		VoucherDateFrom: "2026-01-01",
	})
	if err != nil {
		t.Fatalf("ListVouchers returned error: %v", err)
	}
	if len(result.Content) != 1 || result.Content[0].VoucherNumber != "AG0001" || result.Content[0].OpenAmount != 1960.2 {
		t.Fatalf("unexpected result %+v", result)
	}

	query := fake.request(t, http.MethodGet, "/v1/voucherlist").Query
	want := map[string]string{
		"voucherType":     "quotation,invoice",
		"voucherStatus":   "open",
		"contactId":       "c-1",
		"voucherDateFrom": "2026-01-01",
		"sort":            "voucherDate,DESC",
		"page":            "0",
		"size":            "25",
	}
	for key, value := range want {
		if got := query.Get(key); got != value {
			t.Errorf("query %s = %q, want %q", key, got, value)
		}
	}
	if query.Has("archived") || query.Has("voucherDateTo") {
		t.Errorf("unset filters must not be sent: %v", query)
	}
}

func TestListVouchersValidation(t *testing.T) {
	fake, client := newFakeLexware(t, nil)

	for _, filter := range []VoucherListFilter{
		{VoucherStatuses: []string{"open"}},
		{VoucherTypes: []string{"invoice"}},
		{VoucherTypes: []string{"invoice"}, VoucherStatuses: []string{"open"}, VoucherDateTo: "30.09.2026"},
		{VoucherTypes: []string{"invoice"}, VoucherStatuses: []string{"open"}, Size: 500},
	} {
		if _, err := client.ListVouchers(context.Background(), filter); err == nil {
			t.Errorf("expected validation error for %+v", filter)
		}
	}
	if fake.count() != 0 {
		t.Fatalf("invalid filters must not reach Lexware")
	}
}

func TestSearchContactsRequiresThreeCharacters(t *testing.T) {
	fake, client := newFakeLexware(t, map[string]string{
		"GET /v1/contacts": `{"content":[{"id":"c-1","version":3,"roles":{"customer":{"number":10042}},"person":{"lastName":"Sample"}}],"totalPages":1,"totalElements":1,"size":25,"number":0}`,
	})

	if _, err := client.SearchContacts(context.Background(), ContactFilter{Name: "Mu"}); err == nil {
		t.Fatal("expected short name filter to be rejected")
	}

	customer := true
	result, err := client.SearchContacts(context.Background(), ContactFilter{Name: "Sample", Customer: &customer})
	if err != nil {
		t.Fatalf("SearchContacts returned error: %v", err)
	}
	if result.Content[0].Roles.Customer.Number != 10042 {
		t.Fatalf("unexpected result %+v", result.Content[0])
	}

	query := fake.request(t, http.MethodGet, "/v1/contacts").Query
	if query.Get("name") != "Sample" || query.Get("customer") != "true" {
		t.Fatalf("unexpected query %v", query)
	}
}

func TestNewCustomerContact(t *testing.T) {
	person, err := NewCustomerContact(CustomerDetails{
		Salutation: "Frau",
		FirstName:  "Erika",
		LastName:   "Sample",
		Street:     "Sample Street 1",
		Zip:        "85049",
		City:       "Ingolstadt",
		Email:      "erika@example.com",
		Mobile:     "+49 170 0000000",
	})
	if err != nil {
		t.Fatalf("NewCustomerContact returned error: %v", err)
	}
	if person.Person == nil || person.Company != nil || person.Roles.Customer == nil {
		t.Fatalf("expected private customer, got %+v", person)
	}
	if person.Addresses.Billing[0].CountryCode != "DE" {
		t.Errorf("country code should default to DE")
	}
	if person.EmailAddresses.Private[0] != "erika@example.com" || person.PhoneNumbers.Mobile[0] != "+49 170 0000000" {
		t.Errorf("unexpected communication data %+v %+v", person.EmailAddresses, person.PhoneNumbers)
	}

	company, err := NewCustomerContact(CustomerDetails{CompanyName: "Sample Property Management GmbH", LastName: "Sample", Email: "office@example.com", Phone: "0841 000000"})
	if err != nil {
		t.Fatalf("NewCustomerContact returned error: %v", err)
	}
	if company.Company == nil || company.Person != nil || len(company.Company.ContactPersons) != 1 {
		t.Fatalf("expected company with contact person, got %+v", company)
	}
	if company.EmailAddresses.Business[0] != "office@example.com" || company.PhoneNumbers.Business[0] != "0841 000000" {
		t.Errorf("unexpected communication data %+v %+v", company.EmailAddresses, company.PhoneNumbers)
	}
	if company.Addresses != nil {
		t.Errorf("no address should be sent when none is given")
	}

	if _, err := NewCustomerContact(CustomerDetails{FirstName: "Erika"}); err == nil {
		t.Fatal("expected missing last name to be rejected")
	}
}

const acceptedQuotationJSON = `{
  "id": "q-1",
  "version": 4,
  "voucherStatus": "accepted",
  "voucherNumber": "AG0042",
  "voucherDate": "2026-09-01T00:00:00.000+02:00",
  "expirationDate": "2026-10-01T00:00:00.000+02:00",
  "address": {"contactId": "c-1", "name": "Sample Customer", "street": "Sample Street 1", "zip": "85049", "city": "Ingolstadt", "countryCode": "DE"},
  "lineItems": [
    {"type": "text", "name": "Bathroom ground floor", "description": "Full renovation"},
    {"id": "li-1", "type": "custom", "name": "Wall tiling", "quantity": 23.45, "unitName": "m²", "unitPrice": {"currency": "EUR", "netAmount": 65.5, "grossAmount": 77.95, "taxRatePercentage": 19}, "discountPercentage": 5, "lineItemAmount": 1459.18},
    {"id": "art-1", "type": "service", "name": "Silicone joint", "quantity": 12.5, "unitName": "lfm", "unitPrice": {"currency": "EUR", "netAmount": 8.9, "grossAmount": 10.59, "taxRatePercentage": 19}, "lineItemAmount": 111.25}
  ],
  "totalPrice": {"currency": "EUR", "totalNetAmount": 1570.43, "totalGrossAmount": 1868.81, "totalTaxAmount": 298.38},
  "taxConditions": {"taxType": "net"},
  "paymentConditions": {"paymentTermLabel": "Payable within 14 days", "paymentTermDuration": 14}
}`

func TestCreateOrderConfirmationFromQuotation(t *testing.T) {
	fake, client := newFakeLexware(t, map[string]string{
		"GET /v1/quotations/q-1":       acceptedQuotationJSON,
		"POST /v1/order-confirmations": `{"id":"oc-1","version":1}`,
	})

	result, err := client.CreateOrderConfirmationFromQuotation(context.Background(), OrderConfirmationFromQuotation{
		QuotationID:   "q-1",
		VoucherDate:   "2026-09-29",
		ShippingTerms: &ShippingTerm{ShippingType: "serviceperiod", ShippingDate: "2026-10-12", ShippingEndDate: "2026-10-23"},
		DeliveryTerms: "Execution in calendar weeks 42 and 43",
	})
	if err != nil {
		t.Fatalf("CreateOrderConfirmationFromQuotation returned error: %v", err)
	}
	if result.Created.ID != "oc-1" {
		t.Errorf("created id = %q", result.Created.ID)
	}
	if result.Source.VoucherNumber != "AG0042" || result.Source.TotalNetAmount != 1570.43 || result.Source.LineItemCount != 3 {
		t.Errorf("unexpected source summary %+v", result.Source)
	}

	request := fake.request(t, http.MethodPost, "/v1/order-confirmations")
	if got := request.Query.Get("precedingSalesVoucherId"); got != "q-1" {
		t.Errorf("precedingSalesVoucherId = %q", got)
	}
	body := request.Body
	if got := field(t, body, "voucherDate"); got != "2026-09-29T00:00:00.000+02:00" {
		t.Errorf("voucherDate = %v", got)
	}
	if got := field(t, body, "address", "contactId"); got != "c-1" {
		t.Errorf("address.contactId = %v", got)
	}
	if got := field(t, body, "shippingConditions", "shippingEndDate"); got != "2026-10-23T00:00:00.000+02:00" {
		t.Errorf("shippingEndDate = %v", got)
	}
	if got := field(t, body, "deliveryTerms"); got != "Execution in calendar weeks 42 and 43" {
		t.Errorf("deliveryTerms = %v", got)
	}
	if got := field(t, body, "paymentConditions", "paymentTermDuration"); got != float64(14) {
		t.Errorf("paymentTermDuration = %v", got)
	}

	custom := field(t, body, "lineItems", 1).(map[string]any)
	if _, present := custom["id"]; present {
		t.Errorf("custom line items must not carry an id: %v", custom)
	}
	if custom["quantity"] != 23.45 || custom["discountPercentage"] != float64(5) {
		t.Errorf("unexpected custom line item %v", custom)
	}
	unitPrice := custom["unitPrice"].(map[string]any)
	if unitPrice["netAmount"] != 65.5 {
		t.Errorf("netAmount = %v", unitPrice["netAmount"])
	}
	if _, present := unitPrice["grossAmount"]; present {
		t.Errorf("only the leading net price may be sent: %v", unitPrice)
	}
	if got := field(t, body, "lineItems", 2, "id"); got != "art-1" {
		t.Errorf("service line item id = %v", got)
	}
	if _, present := field(t, body, "lineItems", 2).(map[string]any)["lineItemAmount"]; present {
		t.Errorf("read-only lineItemAmount must not be sent")
	}
}

func TestCreateOrderConfirmationFromQuotationRejectsOptionalPositions(t *testing.T) {
	quotation := strings.Replace(acceptedQuotationJSON, `"name": "Silicone joint",`, `"name": "Silicone joint", "optional": true,`, 1)
	fake, client := newFakeLexware(t, map[string]string{
		"GET /v1/quotations/q-1": quotation,
	})

	_, err := client.CreateOrderConfirmationFromQuotation(context.Background(), OrderConfirmationFromQuotation{QuotationID: "q-1"})
	if err == nil || !strings.Contains(err.Error(), "optional or alternative") {
		t.Fatalf("expected optional position error, got %v", err)
	}
	if fake.count() != 1 {
		t.Fatalf("no order confirmation may be created, got %d requests", fake.count())
	}
}

func TestCreateInvoiceFromOrderConfirmationUsesGrossLeadingPrice(t *testing.T) {
	grossConfirmation := strings.NewReplacer(`"taxType": "net"`, `"taxType": "gross"`, `"id": "q-1"`, `"id": "oc-1"`).Replace(acceptedQuotationJSON)
	fake, client := newFakeLexware(t, map[string]string{
		"GET /v1/order-confirmations/oc-1": grossConfirmation,
		"POST /v1/invoices":                `{"id":"inv-1","version":1}`,
	})

	result, err := client.CreateInvoiceFromVoucher(context.Background(), InvoiceFromVoucher{
		PrecedingVoucherType: VoucherTypeOrderConfirmation,
		PrecedingVoucherID:   "oc-1",
		VoucherDate:          "2026-10-30",
		ShippingTerms:        ShippingTerm{ShippingType: "serviceperiod", ShippingDate: "2026-10-12", ShippingEndDate: "2026-10-23"},
	})
	if err != nil {
		t.Fatalf("CreateInvoiceFromVoucher returned error: %v", err)
	}
	if result.Created.ID != "inv-1" || result.Source.VoucherType != VoucherTypeOrderConfirmation {
		t.Fatalf("unexpected result %+v", result)
	}

	request := fake.request(t, http.MethodPost, "/v1/invoices")
	if request.Query.Get("finalize") != "false" || request.Query.Get("precedingSalesVoucherId") != "oc-1" {
		t.Errorf("unexpected query %v", request.Query)
	}
	unitPrice := field(t, request.Body, "lineItems", 1, "unitPrice").(map[string]any)
	if unitPrice["grossAmount"] != 77.95 {
		t.Errorf("grossAmount = %v", unitPrice["grossAmount"])
	}
	if _, present := unitPrice["netAmount"]; present {
		t.Errorf("only the leading gross price may be sent: %v", unitPrice)
	}
}

func TestCreateInvoiceFromVoucherValidation(t *testing.T) {
	fake, client := newFakeLexware(t, map[string]string{
		"GET /v1/quotations/q-1": acceptedQuotationJSON,
	})

	_, err := client.CreateInvoiceFromVoucher(context.Background(), InvoiceFromVoucher{
		PrecedingVoucherType: VoucherTypeDownPaymentInvoice,
		PrecedingVoucherID:   "dp-1",
		ShippingTerms:        ShippingTerm{ShippingType: "service", ShippingDate: "2026-10-12"},
	})
	if err == nil || !strings.Contains(err.Error(), "precedingVoucherType") {
		t.Fatalf("expected voucher type error, got %v", err)
	}

	_, err = client.CreateInvoiceFromVoucher(context.Background(), InvoiceFromVoucher{
		PrecedingVoucherType: VoucherTypeQuotation,
		PrecedingVoucherID:   "q-1",
		ShippingTerms:        ShippingTerm{ShippingType: "service"},
	})
	if err == nil || !strings.Contains(err.Error(), "shippingDate is required") {
		t.Fatalf("expected missing performance date error, got %v", err)
	}
	if fake.count() != 1 {
		t.Fatalf("only the quotation lookup may reach Lexware, got %d requests", fake.count())
	}
}

func TestGetSalesVoucherEscapesID(t *testing.T) {
	fake, client := newFakeLexware(t, nil)

	if _, err := client.GetSalesVoucher(context.Background(), VoucherTypeInvoice, " "); err == nil {
		t.Fatal("expected empty id to be rejected")
	}
	if _, err := client.GetSalesVoucher(context.Background(), "creditnote", "x"); err == nil {
		t.Fatal("expected unsupported voucher type to be rejected")
	}

	_, err := client.GetSalesVoucher(context.Background(), VoucherTypeInvoice, "../contacts")
	if err == nil {
		t.Fatal("expected 404 from fake server")
	}
	if got := fake.request(t, http.MethodGet, "/v1/invoices/../contacts").EscapedPath; got != "/v1/invoices/..%2Fcontacts" {
		t.Fatalf("id must be escaped as a single path segment, got %q", got)
	}
}

func TestAPIErrorIncludesLexwareMessage(t *testing.T) {
	_, client := newFakeLexware(t, nil)

	_, err := client.GetSalesVoucher(context.Background(), VoucherTypeQuotation, "missing")
	if err == nil || !strings.Contains(err.Error(), "status 404") || !strings.Contains(err.Error(), "unexpected request") {
		t.Fatalf("unexpected error %v", err)
	}
}
