package server

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/dennis/lexware-office-mcp/internal/lexware"
)

type Server struct {
	*mcp.Server
	client *lexware.Client
}

type createSimpleContactInput struct {
	Name            string `json:"name" jsonschema:"Display name used as the contact last name in the simple contact helper"`
	SourceReference string `json:"sourceReference,omitempty" jsonschema:"Optional source reference stored in the contact note, for example an order id"`
}

type searchContactsInput struct {
	Name     string `json:"name,omitempty" jsonschema:"Filter by contact name, at least 3 characters"`
	Email    string `json:"email,omitempty" jsonschema:"Filter by email address, at least 3 characters"`
	Number   int    `json:"number,omitempty" jsonschema:"Filter by customer or vendor number"`
	Customer *bool  `json:"customer,omitempty" jsonschema:"Only contacts with (true) or without (false) the customer role"`
	Vendor   *bool  `json:"vendor,omitempty" jsonschema:"Only contacts with (true) or without (false) the vendor role"`
	Page     int    `json:"page,omitempty" jsonschema:"Zero-based page index"`
	Size     int    `json:"size,omitempty" jsonschema:"Page size between 1 and 250, defaults to 25"`
}

type idInput struct {
	ID string `json:"id" jsonschema:"Lexware resource id (UUID)"`
}

type searchArticlesInput struct {
	ArticleNumber string `json:"articleNumber,omitempty"`
	GTIN          string `json:"gtin,omitempty"`
	Type          string `json:"type,omitempty" jsonschema:"PRODUCT or SERVICE"`
	Page          int    `json:"page,omitempty" jsonschema:"Zero-based page index"`
	Size          int    `json:"size,omitempty" jsonschema:"Page size between 1 and 250, defaults to 25"`
}

type listVouchersInput struct {
	VoucherTypes    []string `json:"voucherTypes" jsonschema:"Voucher types such as quotation, orderconfirmation, invoice, downpaymentinvoice, creditnote, deliverynote, salesinvoice, purchaseinvoice or any"`
	VoucherStatuses []string `json:"voucherStatuses" jsonschema:"Voucher statuses such as draft, open, accepted, rejected, paid, paidoff, overdue, voided or any"`
	Archived        *bool    `json:"archived,omitempty"`
	ContactID       string   `json:"contactId,omitempty" jsonschema:"Only vouchers of this contact"`
	VoucherDateFrom string   `json:"voucherDateFrom,omitempty" jsonschema:"Earliest voucher date as YYYY-MM-DD"`
	VoucherDateTo   string   `json:"voucherDateTo,omitempty" jsonschema:"Latest voucher date as YYYY-MM-DD"`
	Page            int      `json:"page,omitempty" jsonschema:"Zero-based page index"`
	Size            int      `json:"size,omitempty" jsonschema:"Page size between 1 and 250, defaults to 25"`
	Sort            string   `json:"sort,omitempty" jsonschema:"Sort expression such as voucherDate,DESC (default) or voucherNumber,ASC"`
}

type createQuotationInput struct {
	Quotation lexware.Quotation `json:"quotation" jsonschema:"Quotation payload"`
	Finalize  bool              `json:"finalize,omitempty" jsonschema:"Create the quotation as open instead of draft"`
}

type createOrderConfirmationInput struct {
	OrderConfirmation       lexware.OrderConfirmation `json:"orderConfirmation" jsonschema:"Order confirmation payload"`
	PrecedingSalesVoucherID string                    `json:"precedingSalesVoucherId,omitempty" jsonschema:"Optional quotation id to link this order confirmation to"`
}

type createOrderConfirmationFromQuotationInput struct {
	QuotationID        string                `json:"quotationId" jsonschema:"Id of the accepted quotation"`
	VoucherDate        string                `json:"voucherDate,omitempty" jsonschema:"Order confirmation date as YYYY-MM-DD, defaults to today in Europe/Berlin"`
	ShippingConditions *lexware.ShippingTerm `json:"shippingConditions,omitempty" jsonschema:"Planned date or period of execution"`
	DeliveryTerms      string                `json:"deliveryTerms,omitempty"`
	Title              string                `json:"title,omitempty"`
	Introduction       string                `json:"introduction,omitempty"`
	Remark             string                `json:"remark,omitempty"`
}

type createInvoiceInput struct {
	Invoice                 lexware.Invoice `json:"invoice" jsonschema:"Invoice payload"`
	PrecedingSalesVoucherID string          `json:"precedingSalesVoucherId,omitempty" jsonschema:"Optional quotation or order confirmation id to link this invoice to"`
	Finalize                *bool           `json:"finalize,omitempty" jsonschema:"Optional override for Lexware invoice finalization"`
}

type createInvoiceFromVoucherInput struct {
	PrecedingVoucherType string               `json:"precedingVoucherType" jsonschema:"quotation or orderconfirmation"`
	PrecedingVoucherID   string               `json:"precedingVoucherId" jsonschema:"Id of the quotation or order confirmation to invoice"`
	VoucherDate          string               `json:"voucherDate,omitempty" jsonschema:"Invoice date as YYYY-MM-DD, defaults to today in Europe/Berlin"`
	ShippingConditions   lexware.ShippingTerm `json:"shippingConditions" jsonschema:"Date or period of performance (Leistungsdatum or Leistungszeitraum), mandatory on invoices"`
	Title                string               `json:"title,omitempty"`
	Introduction         string               `json:"introduction,omitempty"`
	Remark               string               `json:"remark,omitempty"`
	Finalize             *bool                `json:"finalize,omitempty" jsonschema:"Optional override for Lexware invoice finalization; drafts are the safe default for review"`
}

func New(client *lexware.Client) *mcp.Server {
	srv := mcp.NewServer(&mcp.Implementation{
		Name:    "lexware-office-mcp",
		Version: "0.1.0",
	}, nil)

	wrapped := &Server{
		Server: srv,
		client: client,
	}

	wrapped.registerTools()
	return srv
}

func (s *Server) registerTools() {
	mcp.AddTool(s.Server, &mcp.Tool{
		Name:        "lexware_get_profile",
		Description: "Fetch the current Lexware profile for the configured API token.",
	}, s.getProfile)

	mcp.AddTool(s.Server, &mcp.Tool{
		Name:        "lexware_create_simple_contact",
		Description: "Create a simple customer contact.",
	}, s.createSimpleContact)

	mcp.AddTool(s.Server, &mcp.Tool{
		Name:        "lexware_create_contact",
		Description: "Create a private or business customer with billing address, email and phone numbers. Search for an existing contact first to avoid duplicates.",
	}, s.createContact)

	mcp.AddTool(s.Server, &mcp.Tool{
		Name:        "lexware_search_contacts",
		Description: "Search contacts by name, email, or customer/vendor number.",
	}, s.searchContacts)

	mcp.AddTool(s.Server, &mcp.Tool{
		Name:        "lexware_get_contact",
		Description: "Fetch a single contact with addresses, email addresses and phone numbers.",
	}, s.getContact)

	mcp.AddTool(s.Server, &mcp.Tool{
		Name:        "lexware_search_articles",
		Description: "Search articles by article number, GTIN, or type to reuse catalogue prices in line items.",
	}, s.searchArticles)

	mcp.AddTool(s.Server, &mcp.Tool{
		Name:        "lexware_list_vouchers",
		Description: "List vouchers with status, totals and open amount, for example open quotations, draft invoices, or unpaid invoices of a customer.",
	}, s.listVouchers)

	mcp.AddTool(s.Server, &mcp.Tool{
		Name:        "lexware_list_payment_conditions",
		Description: "List the payment conditions configured in Lexware, including the organization default.",
	}, s.listPaymentConditions)

	mcp.AddTool(s.Server, &mcp.Tool{
		Name:        "lexware_create_quotation",
		Description: "Create a quotation. Quantities may have decimals (m², lfm). Line items can be marked optional. Created as draft unless finalize is true.",
	}, s.createQuotation)

	mcp.AddTool(s.Server, &mcp.Tool{
		Name:        "lexware_create_order_confirmation",
		Description: "Create an order confirmation from a full payload, optionally linked to a quotation.",
	}, s.createOrderConfirmation)

	mcp.AddTool(s.Server, &mcp.Tool{
		Name:        "lexware_create_order_confirmation_from_quotation",
		Description: "Create a linked order confirmation that copies address, line items, prices and conditions from an accepted quotation. Returns the source totals for cross-checking.",
	}, s.createOrderConfirmationFromQuotation)

	mcp.AddTool(s.Server, &mcp.Tool{
		Name:        "lexware_create_invoice",
		Description: "Create an invoice with an optional finalize flag, optionally linked to a quotation or order confirmation.",
	}, s.createInvoice)

	mcp.AddTool(s.Server, &mcp.Tool{
		Name:        "lexware_create_invoice_from_voucher",
		Description: "Create a linked invoice that copies address, line items, prices and conditions from a quotation or order confirmation. Requires the date or period of performance. Returns the source totals for cross-checking.",
	}, s.createInvoiceFromVoucher)

	for _, tool := range []struct {
		name        string
		description string
		voucherType string
	}{
		{"lexware_get_quotation", "Fetch a quotation with line items, totals, status and linked vouchers.", lexware.VoucherTypeQuotation},
		{"lexware_get_order_confirmation", "Fetch an order confirmation with line items, totals, status and linked vouchers.", lexware.VoucherTypeOrderConfirmation},
		{"lexware_get_invoice", "Fetch an invoice with line items, totals, status and linked vouchers.", lexware.VoucherTypeInvoice},
		{"lexware_get_down_payment_invoice", "Fetch a down payment invoice. Lexware does not support creating down payment invoices through the API.", lexware.VoucherTypeDownPaymentInvoice},
	} {
		voucherType := tool.voucherType
		mcp.AddTool(s.Server, &mcp.Tool{
			Name:        tool.name,
			Description: tool.description,
		}, func(ctx context.Context, _ *mcp.CallToolRequest, input idInput) (*mcp.CallToolResult, map[string]any, error) {
			result, err := s.client.GetSalesVoucher(ctx, voucherType, input.ID)
			return s.workflowResult("get "+voucherType, result, err)
		})
	}
}

func (s *Server) getProfile(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, map[string]any, error) {
	result, err := s.client.GetProfile(ctx)
	return s.workflowResult("get profile", result, err)
}

func (s *Server) createSimpleContact(ctx context.Context, _ *mcp.CallToolRequest, input createSimpleContactInput) (*mcp.CallToolResult, map[string]any, error) {
	if strings.TrimSpace(input.Name) == "" {
		return nil, nil, fmt.Errorf("name is required")
	}

	result, err := s.client.CreateSimpleContact(ctx, input.Name, input.SourceReference)
	return s.workflowResult("create contact", result, err)
}

func (s *Server) createContact(ctx context.Context, _ *mcp.CallToolRequest, input lexware.CustomerDetails) (*mcp.CallToolResult, map[string]any, error) {
	contact, err := lexware.NewCustomerContact(input)
	if err != nil {
		return s.workflowResult("create contact", nil, err)
	}

	result, err := s.client.CreateContact(ctx, contact)
	return s.workflowResult("create contact", result, err)
}

func (s *Server) searchContacts(ctx context.Context, _ *mcp.CallToolRequest, input searchContactsInput) (*mcp.CallToolResult, map[string]any, error) {
	result, err := s.client.SearchContacts(ctx, lexware.ContactFilter{
		Email:    input.Email,
		Name:     input.Name,
		Number:   input.Number,
		Customer: input.Customer,
		Vendor:   input.Vendor,
		Page:     input.Page,
		Size:     input.Size,
	})
	return s.workflowResult("search contacts", result, err)
}

func (s *Server) getContact(ctx context.Context, _ *mcp.CallToolRequest, input idInput) (*mcp.CallToolResult, map[string]any, error) {
	result, err := s.client.GetContact(ctx, input.ID)
	return s.workflowResult("get contact", result, err)
}

func (s *Server) searchArticles(ctx context.Context, _ *mcp.CallToolRequest, input searchArticlesInput) (*mcp.CallToolResult, map[string]any, error) {
	result, err := s.client.SearchArticles(ctx, lexware.ArticleFilter{
		ArticleNumber: input.ArticleNumber,
		GTIN:          input.GTIN,
		Type:          input.Type,
		Page:          input.Page,
		Size:          input.Size,
	})
	return s.workflowResult("search articles", result, err)
}

func (s *Server) listVouchers(ctx context.Context, _ *mcp.CallToolRequest, input listVouchersInput) (*mcp.CallToolResult, map[string]any, error) {
	result, err := s.client.ListVouchers(ctx, lexware.VoucherListFilter{
		VoucherTypes:    input.VoucherTypes,
		VoucherStatuses: input.VoucherStatuses,
		Archived:        input.Archived,
		ContactID:       input.ContactID,
		VoucherDateFrom: input.VoucherDateFrom,
		VoucherDateTo:   input.VoucherDateTo,
		Page:            input.Page,
		Size:            input.Size,
		Sort:            input.Sort,
	})
	return s.workflowResult("list vouchers", result, err)
}

func (s *Server) listPaymentConditions(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, map[string]any, error) {
	result, err := s.client.ListPaymentConditions(ctx)
	return s.workflowResult("list payment conditions", result, err)
}

func (s *Server) createQuotation(ctx context.Context, _ *mcp.CallToolRequest, input createQuotationInput) (*mcp.CallToolResult, map[string]any, error) {
	result, err := s.client.CreateQuotation(ctx, input.Quotation, input.Finalize)
	return s.workflowResult("create quotation", result, err)
}

func (s *Server) createOrderConfirmation(ctx context.Context, _ *mcp.CallToolRequest, input createOrderConfirmationInput) (*mcp.CallToolResult, map[string]any, error) {
	result, err := s.client.CreateOrderConfirmation(ctx, input.OrderConfirmation, input.PrecedingSalesVoucherID)
	return s.workflowResult("create order confirmation", result, err)
}

func (s *Server) createOrderConfirmationFromQuotation(ctx context.Context, _ *mcp.CallToolRequest, input createOrderConfirmationFromQuotationInput) (*mcp.CallToolResult, map[string]any, error) {
	result, err := s.client.CreateOrderConfirmationFromQuotation(ctx, lexware.OrderConfirmationFromQuotation{
		QuotationID:   input.QuotationID,
		VoucherDate:   input.VoucherDate,
		ShippingTerms: input.ShippingConditions,
		DeliveryTerms: input.DeliveryTerms,
		Title:         input.Title,
		Introduction:  input.Introduction,
		Remark:        input.Remark,
	})
	return s.workflowResult("create order confirmation from quotation", result, err)
}

func (s *Server) createInvoice(ctx context.Context, _ *mcp.CallToolRequest, input createInvoiceInput) (*mcp.CallToolResult, map[string]any, error) {
	result, err := s.client.CreateInvoice(ctx, input.Invoice, input.PrecedingSalesVoucherID, input.Finalize)
	return s.workflowResult("create invoice", result, err)
}

func (s *Server) createInvoiceFromVoucher(ctx context.Context, _ *mcp.CallToolRequest, input createInvoiceFromVoucherInput) (*mcp.CallToolResult, map[string]any, error) {
	result, err := s.client.CreateInvoiceFromVoucher(ctx, lexware.InvoiceFromVoucher{
		PrecedingVoucherType: input.PrecedingVoucherType,
		PrecedingVoucherID:   input.PrecedingVoucherID,
		VoucherDate:          input.VoucherDate,
		ShippingTerms:        input.ShippingConditions,
		Title:                input.Title,
		Introduction:         input.Introduction,
		Remark:               input.Remark,
		Finalize:             input.Finalize,
	})
	return s.workflowResult("create invoice from voucher", result, err)
}

func (s *Server) workflowResult(action string, result any, err error) (*mcp.CallToolResult, map[string]any, error) {
	payload := map[string]any{
		"result": result,
	}

	if err != nil {
		text := fmt.Sprintf("Lexware %s failed", action)
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				&mcp.TextContent{Text: text + ": " + err.Error()},
			},
		}, payload, nil
	}

	return nil, payload, nil
}
