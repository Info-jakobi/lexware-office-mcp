package lexware

const TaxRatePercentage = 19

const defaultCurrency = "EUR"

// Contact is the Lexware contact resource. It is used for creating contacts
// and for decoding contact lookups.
type Contact struct {
	ID             string                 `json:"id,omitempty"`
	Version        int                    `json:"version"`
	Roles          Roles                  `json:"roles"`
	Person         *Person                `json:"person,omitempty"`
	Company        *Company               `json:"company,omitempty"`
	Addresses      *ContactAddresses      `json:"addresses,omitempty"`
	EmailAddresses *ContactEmailAddresses `json:"emailAddresses,omitempty"`
	PhoneNumbers   *ContactPhoneNumbers   `json:"phoneNumbers,omitempty"`
	Note           string                 `json:"note,omitempty"`
	Archived       bool                   `json:"archived,omitempty"`
}

type Profile struct {
	OrganizationID string `json:"organizationId"`
}

type Person struct {
	Salutation string `json:"salutation,omitempty"`
	FirstName  string `json:"firstName,omitempty"`
	LastName   string `json:"lastName"`
}

type Company struct {
	Name                 string          `json:"name"`
	TaxNumber            string          `json:"taxNumber,omitempty"`
	VatRegistrationID    string          `json:"vatRegistrationId,omitempty"`
	AllowTaxFreeInvoices bool            `json:"allowTaxFreeInvoices,omitempty"`
	ContactPersons       []ContactPerson `json:"contactPersons,omitempty"`
}

type ContactPerson struct {
	Salutation   string `json:"salutation,omitempty"`
	FirstName    string `json:"firstName,omitempty"`
	LastName     string `json:"lastName"`
	Primary      bool   `json:"primary,omitempty"`
	EmailAddress string `json:"emailAddress,omitempty"`
	PhoneNumber  string `json:"phoneNumber,omitempty"`
}

type Roles struct {
	Customer *ContactRole `json:"customer,omitempty"`
	Vendor   *ContactRole `json:"vendor,omitempty"`
}

type ContactRole struct {
	Number int `json:"number,omitempty"`
}

type ContactAddresses struct {
	Billing  []ContactAddress `json:"billing,omitempty"`
	Shipping []ContactAddress `json:"shipping,omitempty"`
}

type ContactAddress struct {
	Supplement  string `json:"supplement,omitempty"`
	Street      string `json:"street,omitempty"`
	Zip         string `json:"zip,omitempty"`
	City        string `json:"city,omitempty"`
	CountryCode string `json:"countryCode"`
}

type ContactEmailAddresses struct {
	Business []string `json:"business,omitempty"`
	Office   []string `json:"office,omitempty"`
	Private  []string `json:"private,omitempty"`
	Other    []string `json:"other,omitempty"`
}

type ContactPhoneNumbers struct {
	Business []string `json:"business,omitempty"`
	Office   []string `json:"office,omitempty"`
	Mobile   []string `json:"mobile,omitempty"`
	Private  []string `json:"private,omitempty"`
	Fax      []string `json:"fax,omitempty"`
	Other    []string `json:"other,omitempty"`
}

type Invoice struct {
	Archived      bool          `json:"archived,omitempty"`
	VoucherDate   string        `json:"voucherDate" jsonschema:"Invoice date as YYYY-MM-DD or RFC 3339 timestamp"`
	Address       Address       `json:"address"`
	LineItems     []LineItem    `json:"lineItems"`
	TotalPrice    TotalPrice    `json:"totalPrice,omitempty" jsonschema:"Currency and optional total discount; currency defaults to EUR"`
	TaxConditions TaxCondition  `json:"taxConditions"`
	PaymentTerms  *PaymentTerms `json:"paymentConditions,omitempty" jsonschema:"Optional payment conditions; see lexware_list_payment_conditions for the configured values"`
	ShippingTerms ShippingTerm  `json:"shippingConditions" jsonschema:"Date or period of performance (Leistungsdatum or Leistungszeitraum)"`
	Title         string        `json:"title,omitempty"`
	Introduction  string        `json:"introduction,omitempty"`
	Remark        string        `json:"remark,omitempty"`
}

type Quotation struct {
	VoucherDate    string        `json:"voucherDate" jsonschema:"Quotation date as YYYY-MM-DD or RFC 3339 timestamp"`
	ExpirationDate string        `json:"expirationDate" jsonschema:"Date until the quotation is valid, as YYYY-MM-DD or RFC 3339 timestamp"`
	Address        Address       `json:"address"`
	LineItems      []LineItem    `json:"lineItems"`
	TotalPrice     TotalPrice    `json:"totalPrice,omitempty" jsonschema:"Currency and optional total discount; currency defaults to EUR"`
	TaxConditions  TaxCondition  `json:"taxConditions"`
	PaymentTerms   *PaymentTerms `json:"paymentConditions,omitempty" jsonschema:"Optional payment conditions; see lexware_list_payment_conditions for the configured values"`
	Title          string        `json:"title,omitempty"`
	Introduction   string        `json:"introduction,omitempty"`
	Remark         string        `json:"remark,omitempty"`
}

type OrderConfirmation struct {
	VoucherDate   string        `json:"voucherDate" jsonschema:"Order confirmation date as YYYY-MM-DD or RFC 3339 timestamp"`
	Address       Address       `json:"address"`
	LineItems     []LineItem    `json:"lineItems"`
	TotalPrice    TotalPrice    `json:"totalPrice,omitempty" jsonschema:"Currency and optional total discount; currency defaults to EUR"`
	TaxConditions TaxCondition  `json:"taxConditions"`
	PaymentTerms  *PaymentTerms `json:"paymentConditions,omitempty" jsonschema:"Optional payment conditions; see lexware_list_payment_conditions for the configured values"`
	ShippingTerms *ShippingTerm `json:"shippingConditions,omitempty" jsonschema:"Planned date or period of performance"`
	DeliveryTerms string        `json:"deliveryTerms,omitempty" jsonschema:"Free text describing the delivery or execution terms"`
	Title         string        `json:"title,omitempty"`
	Introduction  string        `json:"introduction,omitempty"`
	Remark        string        `json:"remark,omitempty"`
}

type Address struct {
	ContactID   string `json:"contactId,omitempty" jsonschema:"Existing Lexware contact id; when set the other address fields are optional"`
	Supplement  string `json:"supplement,omitempty"`
	Name        string `json:"name,omitempty" jsonschema:"Recipient name; required when no contactId is given"`
	Street      string `json:"street,omitempty"`
	City        string `json:"city,omitempty"`
	Zip         string `json:"zip,omitempty"`
	CountryCode string `json:"countryCode,omitempty" jsonschema:"ISO 3166 alpha-2 country code such as DE; required when no contactId is given"`
}

type LineItem struct {
	ID                 string     `json:"id,omitempty" jsonschema:"Lexware article id; required for material and service line items"`
	Type               string     `json:"type" jsonschema:"One of custom, material, service or text"`
	Name               string     `json:"name"`
	Description        string     `json:"description,omitempty"`
	Quantity           float64    `json:"quantity,omitempty" jsonschema:"Quantity with up to 4 decimals, for example 12.35 for square meters; omit for text items"`
	UnitName           string     `json:"unitName,omitempty" jsonschema:"Unit such as m², lfm, Stk, Std or psch; omit for text items"`
	UnitPrice          *UnitPrice `json:"unitPrice,omitempty" jsonschema:"Unit price; omit for text items"`
	DiscountPercentage float64    `json:"discountPercentage,omitempty"`
	Optional           bool       `json:"optional,omitempty" jsonschema:"Quotations only: marks an optional position (Bedarfsposition) that is not part of the total"`
}

type UnitPrice struct {
	Currency          string   `json:"currency,omitempty" jsonschema:"Currency code; defaults to EUR"`
	NetAmount         *float64 `json:"netAmount,omitempty" jsonschema:"Net unit price; set this for every tax type except gross"`
	GrossAmount       *float64 `json:"grossAmount,omitempty" jsonschema:"Gross unit price; set this when taxType is gross"`
	TaxRatePercentage float64  `json:"taxRatePercentage" jsonschema:"Tax rate in percent, for example 19, 7 or 0"`
}

type TotalPrice struct {
	Currency                string   `json:"currency,omitempty" jsonschema:"Currency code; defaults to EUR"`
	TotalDiscountAbsolute   *float64 `json:"totalDiscountAbsolute,omitempty"`
	TotalDiscountPercentage *float64 `json:"totalDiscountPercentage,omitempty"`
}

type TaxCondition struct {
	TaxType     string `json:"taxType" jsonschema:"One of net, gross, vatfree, intraCommunitySupply, constructionService13b, externalService13b, thirdPartyCountryService, thirdPartyCountryDelivery, photovoltaicEquipment"`
	TaxTypeNote string `json:"taxTypeNote,omitempty"`
}

type PaymentTerms struct {
	PaymentTermLabel          string                     `json:"paymentTermLabel"`
	PaymentTermDuration       int                        `json:"paymentTermDuration" jsonschema:"Payment term in days"`
	PaymentDiscountConditions *PaymentDiscountConditions `json:"paymentDiscountConditions,omitempty" jsonschema:"Optional early payment discount (Skonto)"`
}

type PaymentDiscountConditions struct {
	DiscountPercentage float64 `json:"discountPercentage"`
	DiscountRange      int     `json:"discountRange" jsonschema:"Number of days in which the discount applies"`
}

type ShippingTerm struct {
	ShippingType    string `json:"shippingType" jsonschema:"One of service, serviceperiod, delivery, deliveryperiod or none"`
	ShippingDate    string `json:"shippingDate,omitempty" jsonschema:"Date of performance; required for service and delivery, start date for periods"`
	ShippingEndDate string `json:"shippingEndDate,omitempty" jsonschema:"End date; required for serviceperiod and deliveryperiod"`
}

func TaxConditionGross() TaxCondition {
	return TaxCondition{TaxType: "gross"}
}

func ShippingTermNone() ShippingTerm {
	return ShippingTerm{ShippingType: "none"}
}

// ResourceReference is returned by Lexware after creating or updating a resource.
type ResourceReference struct {
	ID          string `json:"id"`
	ResourceURI string `json:"resourceUri,omitempty"`
	CreatedDate string `json:"createdDate,omitempty"`
	UpdatedDate string `json:"updatedDate,omitempty"`
	Version     int    `json:"version,omitempty"`
}

// Page is the paging envelope Lexware uses for list endpoints.
type Page[T any] struct {
	Content          []T  `json:"content"`
	First            bool `json:"first"`
	Last             bool `json:"last"`
	TotalPages       int  `json:"totalPages"`
	TotalElements    int  `json:"totalElements"`
	NumberOfElements int  `json:"numberOfElements"`
	Size             int  `json:"size"`
	Number           int  `json:"number"`
}

// SalesVoucher is the read model shared by quotations, order confirmations,
// invoices, and down payment invoices.
type SalesVoucher struct {
	ID                 string           `json:"id"`
	Version            int              `json:"version"`
	Language           string           `json:"language,omitempty"`
	Archived           bool             `json:"archived"`
	VoucherStatus      string           `json:"voucherStatus"`
	VoucherNumber      string           `json:"voucherNumber,omitempty"`
	VoucherDate        string           `json:"voucherDate"`
	DueDate            string           `json:"dueDate,omitempty"`
	ExpirationDate     string           `json:"expirationDate,omitempty"`
	CreatedDate        string           `json:"createdDate,omitempty"`
	UpdatedDate        string           `json:"updatedDate,omitempty"`
	Address            Address          `json:"address"`
	LineItems          []LineItemResult `json:"lineItems"`
	TotalPrice         TotalPriceResult `json:"totalPrice"`
	TaxAmounts         []TaxAmount      `json:"taxAmounts,omitempty"`
	TaxConditions      TaxCondition     `json:"taxConditions"`
	PaymentConditions  *PaymentTerms    `json:"paymentConditions,omitempty"`
	ShippingConditions *ShippingTerm    `json:"shippingConditions,omitempty"`
	DeliveryTerms      string           `json:"deliveryTerms,omitempty"`
	ClosingInvoice     bool             `json:"closingInvoice,omitempty"`
	RelatedVouchers    []RelatedVoucher `json:"relatedVouchers,omitempty"`
	Title              string           `json:"title,omitempty"`
	Introduction       string           `json:"introduction,omitempty"`
	Remark             string           `json:"remark,omitempty"`
}

type LineItemResult struct {
	ID                 string           `json:"id,omitempty"`
	Type               string           `json:"type"`
	Name               string           `json:"name"`
	Description        string           `json:"description,omitempty"`
	Quantity           float64          `json:"quantity,omitempty"`
	UnitName           string           `json:"unitName,omitempty"`
	UnitPrice          *UnitPriceResult `json:"unitPrice,omitempty"`
	DiscountPercentage float64          `json:"discountPercentage,omitempty"`
	LineItemAmount     float64          `json:"lineItemAmount,omitempty"`
	Optional           bool             `json:"optional,omitempty"`
	Alternative        bool             `json:"alternative,omitempty"`
}

type UnitPriceResult struct {
	Currency          string  `json:"currency"`
	NetAmount         float64 `json:"netAmount"`
	GrossAmount       float64 `json:"grossAmount"`
	TaxRatePercentage float64 `json:"taxRatePercentage"`
}

type TotalPriceResult struct {
	Currency                string   `json:"currency"`
	TotalNetAmount          float64  `json:"totalNetAmount"`
	TotalGrossAmount        float64  `json:"totalGrossAmount"`
	TotalTaxAmount          float64  `json:"totalTaxAmount"`
	TotalDiscountAbsolute   *float64 `json:"totalDiscountAbsolute,omitempty"`
	TotalDiscountPercentage *float64 `json:"totalDiscountPercentage,omitempty"`
}

type TaxAmount struct {
	TaxRatePercentage float64 `json:"taxRatePercentage"`
	TaxAmount         float64 `json:"taxAmount"`
	Amount            float64 `json:"amount"`
}

type RelatedVoucher struct {
	ID            string `json:"id"`
	VoucherNumber string `json:"voucherNumber,omitempty"`
	VoucherType   string `json:"voucherType"`
}

// VoucherListItem is one entry of the /v1/voucherlist endpoint.
type VoucherListItem struct {
	ID            string  `json:"id"`
	VoucherType   string  `json:"voucherType"`
	VoucherStatus string  `json:"voucherStatus"`
	VoucherNumber string  `json:"voucherNumber,omitempty"`
	VoucherDate   string  `json:"voucherDate"`
	CreatedDate   string  `json:"createdDate,omitempty"`
	UpdatedDate   string  `json:"updatedDate,omitempty"`
	DueDate       string  `json:"dueDate,omitempty"`
	ContactID     string  `json:"contactId,omitempty"`
	ContactName   string  `json:"contactName,omitempty"`
	TotalAmount   float64 `json:"totalAmount"`
	OpenAmount    float64 `json:"openAmount"`
	Currency      string  `json:"currency"`
	Archived      bool    `json:"archived"`
}

type Article struct {
	ID            string       `json:"id"`
	Version       int          `json:"version"`
	Title         string       `json:"title"`
	Description   string       `json:"description,omitempty"`
	Type          string       `json:"type"`
	ArticleNumber string       `json:"articleNumber,omitempty"`
	GTIN          string       `json:"gtin,omitempty"`
	Note          string       `json:"note,omitempty"`
	UnitName      string       `json:"unitName"`
	Price         ArticlePrice `json:"price"`
	Archived      bool         `json:"archived,omitempty"`
}

type ArticlePrice struct {
	NetPrice     float64 `json:"netPrice"`
	GrossPrice   float64 `json:"grossPrice"`
	LeadingPrice string  `json:"leadingPrice"`
	TaxRate      float64 `json:"taxRate"`
}

type PaymentCondition struct {
	ID                        string                     `json:"id"`
	OrganizationDefault       bool                       `json:"organizationDefault"`
	PaymentTermLabelTemplate  string                     `json:"paymentTermLabelTemplate,omitempty"`
	PaymentTermDuration       int                        `json:"paymentTermDuration"`
	PaymentDiscountConditions *PaymentDiscountConditions `json:"paymentDiscountConditions,omitempty"`
}
