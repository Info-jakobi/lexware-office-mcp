package lexware

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

const (
	VoucherTypeQuotation          = "quotation"
	VoucherTypeOrderConfirmation  = "orderconfirmation"
	VoucherTypeInvoice            = "invoice"
	VoucherTypeDownPaymentInvoice = "downpaymentinvoice"
)

const (
	defaultPageSize = 25
	maxPageSize     = 250
)

// salesVoucherPaths maps voucher types to their Lexware resource paths.
var salesVoucherPaths = map[string]string{
	VoucherTypeQuotation:          "/v1/quotations",
	VoucherTypeOrderConfirmation:  "/v1/order-confirmations",
	VoucherTypeInvoice:            "/v1/invoices",
	VoucherTypeDownPaymentInvoice: "/v1/down-payment-invoices",
}

func (c *Client) GetProfile(ctx context.Context) (*Profile, error) {
	result := &Profile{}
	resp, err := c.newRequest(ctx).
		SetResult(result).
		Get("/v1/profile")
	if apiErr := wrapAPIError("get profile", resp, err); apiErr != nil {
		return nil, apiErr
	}

	return result, nil
}

func (c *Client) CreateSimpleContact(ctx context.Context, name, sourceReference string) (*ResourceReference, error) {
	contact := Contact{
		Version: 0,
		Roles: Roles{
			Customer: &ContactRole{},
		},
		Person: &Person{
			LastName: name,
		},
	}
	if strings.TrimSpace(sourceReference) != "" {
		contact.Note = sourceReference
	}

	return c.CreateContact(ctx, contact)
}

func (c *Client) CreateContact(ctx context.Context, contact Contact) (*ResourceReference, error) {
	result := &ResourceReference{}
	resp, err := c.newRequest(ctx).
		SetBody(contact).
		SetResult(result).
		Post("/v1/contacts")
	if apiErr := wrapAPIError("create contact", resp, err); apiErr != nil {
		return result, apiErr
	}

	return result, nil
}

// CustomerDetails is the flat description of a private or business customer
// used to build a Lexware contact.
type CustomerDetails struct {
	CompanyName string `json:"companyName,omitempty" jsonschema:"Company name for business customers; leave empty for private customers"`
	Salutation  string `json:"salutation,omitempty" jsonschema:"Salutation such as Herr or Frau"`
	FirstName   string `json:"firstName,omitempty"`
	LastName    string `json:"lastName,omitempty" jsonschema:"Last name; required for private customers, optional contact person for companies"`
	Street      string `json:"street,omitempty"`
	Zip         string `json:"zip,omitempty"`
	City        string `json:"city,omitempty"`
	CountryCode string `json:"countryCode,omitempty" jsonschema:"ISO 3166 alpha-2 country code; defaults to DE when an address is given"`
	Supplement  string `json:"supplement,omitempty" jsonschema:"Address supplement, for example c/o or building part"`
	Email       string `json:"email,omitempty"`
	Phone       string `json:"phone,omitempty"`
	Mobile      string `json:"mobile,omitempty"`
	Note        string `json:"note,omitempty"`
}

// NewCustomerContact builds a customer contact from flat customer details.
func NewCustomerContact(details CustomerDetails) (Contact, error) {
	contact := Contact{
		Roles: Roles{Customer: &ContactRole{}},
		Note:  strings.TrimSpace(details.Note),
	}

	companyName := strings.TrimSpace(details.CompanyName)
	lastName := strings.TrimSpace(details.LastName)
	email := strings.TrimSpace(details.Email)
	phone := strings.TrimSpace(details.Phone)
	mobile := strings.TrimSpace(details.Mobile)

	if companyName != "" {
		company := &Company{Name: companyName}
		if lastName != "" {
			company.ContactPersons = []ContactPerson{{
				Salutation:   strings.TrimSpace(details.Salutation),
				FirstName:    strings.TrimSpace(details.FirstName),
				LastName:     lastName,
				Primary:      true,
				EmailAddress: email,
				PhoneNumber:  phone,
			}}
		}
		contact.Company = company
	} else {
		if lastName == "" {
			return contact, errors.New("lastName is required for private customers")
		}
		contact.Person = &Person{
			Salutation: strings.TrimSpace(details.Salutation),
			FirstName:  strings.TrimSpace(details.FirstName),
			LastName:   lastName,
		}
	}

	if details.Street != "" || details.Zip != "" || details.City != "" {
		countryCode := strings.ToUpper(strings.TrimSpace(details.CountryCode))
		if countryCode == "" {
			countryCode = "DE"
		}
		contact.Addresses = &ContactAddresses{Billing: []ContactAddress{{
			Supplement:  strings.TrimSpace(details.Supplement),
			Street:      strings.TrimSpace(details.Street),
			Zip:         strings.TrimSpace(details.Zip),
			City:        strings.TrimSpace(details.City),
			CountryCode: countryCode,
		}}}
	}

	if email != "" {
		if companyName != "" {
			contact.EmailAddresses = &ContactEmailAddresses{Business: []string{email}}
		} else {
			contact.EmailAddresses = &ContactEmailAddresses{Private: []string{email}}
		}
	}
	if phone != "" || mobile != "" {
		numbers := &ContactPhoneNumbers{}
		if phone != "" {
			if companyName != "" {
				numbers.Business = []string{phone}
			} else {
				numbers.Private = []string{phone}
			}
		}
		if mobile != "" {
			numbers.Mobile = []string{mobile}
		}
		contact.PhoneNumbers = numbers
	}

	return contact, nil
}

type ContactFilter struct {
	Email    string
	Name     string
	Number   int
	Customer *bool
	Vendor   *bool
	Page     int
	Size     int
}

func (c *Client) SearchContacts(ctx context.Context, filter ContactFilter) (*Page[Contact], error) {
	for _, field := range []struct{ name, value string }{{"email", filter.Email}, {"name", filter.Name}} {
		if field.value != "" && len([]rune(strings.TrimSpace(field.value))) < 3 {
			return nil, fmt.Errorf("%s filter needs at least 3 characters", field.name)
		}
	}

	req := c.newRequest(ctx)
	setQueryIfNotEmpty(req.QueryParam, "email", strings.TrimSpace(filter.Email))
	setQueryIfNotEmpty(req.QueryParam, "name", strings.TrimSpace(filter.Name))
	if filter.Number > 0 {
		req.SetQueryParam("number", strconv.Itoa(filter.Number))
	}
	if filter.Customer != nil {
		req.SetQueryParam("customer", strconv.FormatBool(*filter.Customer))
	}
	if filter.Vendor != nil {
		req.SetQueryParam("vendor", strconv.FormatBool(*filter.Vendor))
	}
	if err := setPaging(req.QueryParam, filter.Page, filter.Size); err != nil {
		return nil, err
	}

	result := &Page[Contact]{}
	resp, err := req.SetResult(result).Get("/v1/contacts")
	if apiErr := wrapAPIError("search contacts", resp, err); apiErr != nil {
		return nil, apiErr
	}

	return result, nil
}

func (c *Client) GetContact(ctx context.Context, id string) (*Contact, error) {
	result := &Contact{}
	if err := c.getByID(ctx, "get contact", "/v1/contacts", id, result); err != nil {
		return nil, err
	}
	return result, nil
}

type ArticleFilter struct {
	ArticleNumber string
	GTIN          string
	Type          string
	Page          int
	Size          int
}

func (c *Client) SearchArticles(ctx context.Context, filter ArticleFilter) (*Page[Article], error) {
	articleType := strings.ToUpper(strings.TrimSpace(filter.Type))
	if articleType != "" && articleType != "PRODUCT" && articleType != "SERVICE" {
		return nil, fmt.Errorf("type must be PRODUCT or SERVICE, got %q", filter.Type)
	}

	req := c.newRequest(ctx)
	setQueryIfNotEmpty(req.QueryParam, "articleNumber", strings.TrimSpace(filter.ArticleNumber))
	setQueryIfNotEmpty(req.QueryParam, "gtin", strings.TrimSpace(filter.GTIN))
	setQueryIfNotEmpty(req.QueryParam, "type", articleType)
	if err := setPaging(req.QueryParam, filter.Page, filter.Size); err != nil {
		return nil, err
	}

	result := &Page[Article]{}
	resp, err := req.SetResult(result).Get("/v1/articles")
	if apiErr := wrapAPIError("search articles", resp, err); apiErr != nil {
		return nil, apiErr
	}

	return result, nil
}

type VoucherListFilter struct {
	VoucherTypes    []string
	VoucherStatuses []string
	Archived        *bool
	ContactID       string
	VoucherDateFrom string
	VoucherDateTo   string
	Page            int
	Size            int
	Sort            string
}

func (c *Client) ListVouchers(ctx context.Context, filter VoucherListFilter) (*Page[VoucherListItem], error) {
	voucherTypes := joinNonEmpty(filter.VoucherTypes)
	if voucherTypes == "" {
		return nil, errors.New("at least one voucher type is required")
	}
	voucherStatuses := joinNonEmpty(filter.VoucherStatuses)
	if voucherStatuses == "" {
		return nil, errors.New("at least one voucher status is required")
	}
	if err := validateDateOnly("voucherDateFrom", filter.VoucherDateFrom); err != nil {
		return nil, err
	}
	if err := validateDateOnly("voucherDateTo", filter.VoucherDateTo); err != nil {
		return nil, err
	}

	sort := strings.TrimSpace(filter.Sort)
	if sort == "" {
		sort = "voucherDate,DESC"
	}

	req := c.newRequest(ctx).
		SetQueryParam("voucherType", voucherTypes).
		SetQueryParam("voucherStatus", voucherStatuses).
		SetQueryParam("sort", sort)
	if filter.Archived != nil {
		req.SetQueryParam("archived", strconv.FormatBool(*filter.Archived))
	}
	setQueryIfNotEmpty(req.QueryParam, "contactId", strings.TrimSpace(filter.ContactID))
	setQueryIfNotEmpty(req.QueryParam, "voucherDateFrom", filter.VoucherDateFrom)
	setQueryIfNotEmpty(req.QueryParam, "voucherDateTo", filter.VoucherDateTo)
	if err := setPaging(req.QueryParam, filter.Page, filter.Size); err != nil {
		return nil, err
	}

	result := &Page[VoucherListItem]{}
	resp, err := req.SetResult(result).Get("/v1/voucherlist")
	if apiErr := wrapAPIError("list vouchers", resp, err); apiErr != nil {
		return nil, apiErr
	}

	return result, nil
}

func (c *Client) ListPaymentConditions(ctx context.Context) ([]PaymentCondition, error) {
	var result []PaymentCondition
	resp, err := c.newRequest(ctx).
		SetResult(&result).
		Get("/v1/payment-conditions")
	if apiErr := wrapAPIError("list payment conditions", resp, err); apiErr != nil {
		return nil, apiErr
	}

	return result, nil
}

// GetSalesVoucher fetches a quotation, order confirmation, invoice, or down
// payment invoice by id.
func (c *Client) GetSalesVoucher(ctx context.Context, voucherType, id string) (*SalesVoucher, error) {
	path, ok := salesVoucherPaths[voucherType]
	if !ok {
		return nil, fmt.Errorf("unsupported voucher type %q", voucherType)
	}

	result := &SalesVoucher{}
	if err := c.getByID(ctx, "get "+voucherType, path, id, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) CreateQuotation(ctx context.Context, quotation Quotation, finalize bool) (*ResourceReference, error) {
	prepared, err := quotation.prepare()
	if err != nil {
		return nil, err
	}

	result := &ResourceReference{}
	resp, err := c.newRequest(ctx).
		SetBody(prepared).
		SetResult(result).
		SetQueryParam("finalize", strconv.FormatBool(finalize)).
		Post("/v1/quotations")
	if apiErr := wrapAPIError("create quotation", resp, err); apiErr != nil {
		return result, apiErr
	}

	return result, nil
}

// CreateOrderConfirmation creates an order confirmation. A non-empty
// precedingSalesVoucherID links it to the quotation it is based on.
func (c *Client) CreateOrderConfirmation(ctx context.Context, confirmation OrderConfirmation, precedingSalesVoucherID string) (*ResourceReference, error) {
	prepared, err := confirmation.prepare()
	if err != nil {
		return nil, err
	}

	req := c.newRequest(ctx)
	setQueryIfNotEmpty(req.QueryParam, "precedingSalesVoucherId", strings.TrimSpace(precedingSalesVoucherID))

	result := &ResourceReference{}
	resp, err := req.
		SetBody(prepared).
		SetResult(result).
		Post("/v1/order-confirmations")
	if apiErr := wrapAPIError("create order confirmation", resp, err); apiErr != nil {
		return result, apiErr
	}

	return result, nil
}

// CreateInvoice creates an invoice. A non-empty precedingSalesVoucherID links
// it to the quotation or order confirmation it is based on.
func (c *Client) CreateInvoice(ctx context.Context, invoice Invoice, precedingSalesVoucherID string, finalize *bool) (*ResourceReference, error) {
	prepared, err := invoice.prepare()
	if err != nil {
		return nil, err
	}

	req := c.newRequest(ctx)
	setQueryIfNotEmpty(req.QueryParam, "precedingSalesVoucherId", strings.TrimSpace(precedingSalesVoucherID))

	result := &ResourceReference{}
	resp, err := req.
		SetBody(prepared).
		SetResult(result).
		SetQueryParam("finalize", strconv.FormatBool(c.resolveFinalize(finalize))).
		Post("/v1/invoices")
	if apiErr := wrapAPIError("create invoice", resp, err); apiErr != nil {
		return result, apiErr
	}

	return result, nil
}

// FollowUpResult describes a voucher created from a preceding voucher.
type FollowUpResult struct {
	Created ResourceReference `json:"created"`
	Source  SourceVoucher     `json:"source"`
}

// SourceVoucher summarizes the preceding voucher so callers can cross-check
// totals against the created follow-up voucher.
type SourceVoucher struct {
	ID               string  `json:"id"`
	VoucherType      string  `json:"voucherType"`
	VoucherNumber    string  `json:"voucherNumber,omitempty"`
	VoucherStatus    string  `json:"voucherStatus"`
	LineItemCount    int     `json:"lineItemCount"`
	TotalNetAmount   float64 `json:"totalNetAmount"`
	TotalGrossAmount float64 `json:"totalGrossAmount"`
}

type OrderConfirmationFromQuotation struct {
	QuotationID   string
	VoucherDate   string
	ShippingTerms *ShippingTerm
	DeliveryTerms string
	Title         string
	Introduction  string
	Remark        string
}

// CreateOrderConfirmationFromQuotation copies address, line items, prices,
// and conditions of a quotation into a linked order confirmation.
func (c *Client) CreateOrderConfirmationFromQuotation(ctx context.Context, request OrderConfirmationFromQuotation) (*FollowUpResult, error) {
	source, err := c.GetSalesVoucher(ctx, VoucherTypeQuotation, request.QuotationID)
	if err != nil {
		return nil, err
	}
	base, err := followUpBaseFrom(source)
	if err != nil {
		return nil, err
	}

	voucherDate := request.VoucherDate
	if strings.TrimSpace(voucherDate) == "" {
		voucherDate = Today()
	}

	confirmation := OrderConfirmation{
		VoucherDate:   voucherDate,
		Address:       base.address,
		LineItems:     base.lineItems,
		TotalPrice:    base.totalPrice,
		TaxConditions: base.taxConditions,
		PaymentTerms:  base.paymentTerms,
		ShippingTerms: request.ShippingTerms,
		DeliveryTerms: request.DeliveryTerms,
		Title:         request.Title,
		Introduction:  request.Introduction,
		Remark:        request.Remark,
	}

	created, err := c.CreateOrderConfirmation(ctx, confirmation, source.ID)
	if err != nil {
		return nil, err
	}
	return &FollowUpResult{Created: *created, Source: summarizeSource(VoucherTypeQuotation, source)}, nil
}

type InvoiceFromVoucher struct {
	PrecedingVoucherType string
	PrecedingVoucherID   string
	VoucherDate          string
	ShippingTerms        ShippingTerm
	Title                string
	Introduction         string
	Remark               string
	Finalize             *bool
}

// CreateInvoiceFromVoucher copies a quotation or order confirmation into a
// linked invoice. Invoices are created as drafts unless finalize is set.
func (c *Client) CreateInvoiceFromVoucher(ctx context.Context, request InvoiceFromVoucher) (*FollowUpResult, error) {
	if request.PrecedingVoucherType != VoucherTypeQuotation && request.PrecedingVoucherType != VoucherTypeOrderConfirmation {
		return nil, fmt.Errorf("precedingVoucherType must be %s or %s, got %q", VoucherTypeQuotation, VoucherTypeOrderConfirmation, request.PrecedingVoucherType)
	}

	source, err := c.GetSalesVoucher(ctx, request.PrecedingVoucherType, request.PrecedingVoucherID)
	if err != nil {
		return nil, err
	}
	base, err := followUpBaseFrom(source)
	if err != nil {
		return nil, err
	}

	voucherDate := request.VoucherDate
	if strings.TrimSpace(voucherDate) == "" {
		voucherDate = Today()
	}

	invoice := Invoice{
		VoucherDate:   voucherDate,
		Address:       base.address,
		LineItems:     base.lineItems,
		TotalPrice:    base.totalPrice,
		TaxConditions: base.taxConditions,
		PaymentTerms:  base.paymentTerms,
		ShippingTerms: request.ShippingTerms,
		Title:         request.Title,
		Introduction:  request.Introduction,
		Remark:        request.Remark,
	}

	created, err := c.CreateInvoice(ctx, invoice, source.ID, request.Finalize)
	if err != nil {
		return nil, err
	}
	return &FollowUpResult{Created: *created, Source: summarizeSource(request.PrecedingVoucherType, source)}, nil
}

type followUpBase struct {
	address       Address
	lineItems     []LineItem
	totalPrice    TotalPrice
	taxConditions TaxCondition
	paymentTerms  *PaymentTerms
}

// followUpBaseFrom converts a fetched sales voucher into the writable parts
// of a follow-up voucher. Only the leading unit price is sent so Lexware
// recalculates the other one.
func followUpBaseFrom(source *SalesVoucher) (followUpBase, error) {
	base := followUpBase{
		address:       source.Address,
		taxConditions: source.TaxConditions,
		paymentTerms:  source.PaymentConditions,
		totalPrice: TotalPrice{
			Currency:                source.TotalPrice.Currency,
			TotalDiscountAbsolute:   source.TotalPrice.TotalDiscountAbsolute,
			TotalDiscountPercentage: source.TotalPrice.TotalDiscountPercentage,
		},
	}

	grossLeading := source.TaxConditions.TaxType == "gross"
	for index, item := range source.LineItems {
		if item.Optional || item.Alternative {
			return base, fmt.Errorf("lineItems[%d] %q is an optional or alternative position; Lexware rejects follow-up vouchers for such quotations, so resolve these positions in Lexware first", index, item.Name)
		}

		lineItem := LineItem{
			Type:        item.Type,
			Name:        item.Name,
			Description: item.Description,
		}
		if item.Type != "text" {
			if item.Type == "material" || item.Type == "service" {
				lineItem.ID = item.ID
			}
			lineItem.Quantity = item.Quantity
			lineItem.UnitName = item.UnitName
			lineItem.DiscountPercentage = item.DiscountPercentage
			if item.UnitPrice != nil {
				unitPrice := &UnitPrice{
					Currency:          item.UnitPrice.Currency,
					TaxRatePercentage: item.UnitPrice.TaxRatePercentage,
				}
				if grossLeading {
					unitPrice.GrossAmount = &item.UnitPrice.GrossAmount
				} else {
					unitPrice.NetAmount = &item.UnitPrice.NetAmount
				}
				lineItem.UnitPrice = unitPrice
			}
		}
		base.lineItems = append(base.lineItems, lineItem)
	}

	return base, nil
}

func summarizeSource(voucherType string, source *SalesVoucher) SourceVoucher {
	return SourceVoucher{
		ID:               source.ID,
		VoucherType:      voucherType,
		VoucherNumber:    source.VoucherNumber,
		VoucherStatus:    source.VoucherStatus,
		LineItemCount:    len(source.LineItems),
		TotalNetAmount:   source.TotalPrice.TotalNetAmount,
		TotalGrossAmount: source.TotalPrice.TotalGrossAmount,
	}
}

func (c *Client) resolveFinalize(value *bool) bool {
	if value != nil {
		return *value
	}
	return c.finalizeInvoices
}

func (c *Client) getByID(ctx context.Context, action, basePath, id string, result any) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return errors.New("id is required")
	}

	resp, err := c.newRequest(ctx).
		SetPathParam("id", id).
		SetResult(result).
		Get(basePath + "/{id}")
	return wrapAPIError(action, resp, err)
}

func setQueryIfNotEmpty(values interface{ Set(key, value string) }, key, value string) {
	if value != "" {
		values.Set(key, value)
	}
}

func setPaging(values interface{ Set(key, value string) }, page, size int) error {
	if page < 0 {
		return errors.New("page must not be negative")
	}
	if size < 0 || size > maxPageSize {
		return fmt.Errorf("size must be between 1 and %d", maxPageSize)
	}
	if size == 0 {
		size = defaultPageSize
	}
	values.Set("page", strconv.Itoa(page))
	values.Set("size", strconv.Itoa(size))
	return nil
}

func joinNonEmpty(values []string) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return strings.Join(parts, ",")
}
