package lexware

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
)

var validTaxTypes = []string{
	"net",
	"gross",
	"vatfree",
	"intraCommunitySupply",
	"constructionService13b",
	"externalService13b",
	"thirdPartyCountryService",
	"thirdPartyCountryDelivery",
	"photovoltaicEquipment",
}

var validShippingTypes = []string{"service", "serviceperiod", "delivery", "deliveryperiod", "none"}

var validLineItemTypes = []string{"custom", "material", "service", "text"}

func (q Quotation) prepare() (Quotation, error) {
	var err error
	if q.VoucherDate, err = requireDate("voucherDate", q.VoucherDate); err != nil {
		return q, err
	}
	if q.ExpirationDate, err = requireDate("expirationDate", q.ExpirationDate); err != nil {
		return q, err
	}
	if isBefore(q.ExpirationDate, q.VoucherDate) {
		return q, errors.New("expirationDate must not be before voucherDate")
	}
	if err = validateAddress(q.Address); err != nil {
		return q, err
	}
	if err = validateTaxConditions(q.TaxConditions); err != nil {
		return q, err
	}
	if q.LineItems, err = prepareLineItems(q.LineItems, true); err != nil {
		return q, err
	}
	q.TotalPrice = prepareTotalPrice(q.TotalPrice)
	return q, nil
}

func (o OrderConfirmation) prepare() (OrderConfirmation, error) {
	var err error
	if o.VoucherDate, err = requireDate("voucherDate", o.VoucherDate); err != nil {
		return o, err
	}
	if err = validateAddress(o.Address); err != nil {
		return o, err
	}
	if err = validateTaxConditions(o.TaxConditions); err != nil {
		return o, err
	}
	if o.LineItems, err = prepareLineItems(o.LineItems, false); err != nil {
		return o, err
	}
	if o.ShippingTerms != nil {
		shipping, err := prepareShippingTerm(*o.ShippingTerms)
		if err != nil {
			return o, err
		}
		o.ShippingTerms = &shipping
	}
	o.TotalPrice = prepareTotalPrice(o.TotalPrice)
	return o, nil
}

func (i Invoice) prepare() (Invoice, error) {
	var err error
	if i.VoucherDate, err = requireDate("voucherDate", i.VoucherDate); err != nil {
		return i, err
	}
	if err = validateAddress(i.Address); err != nil {
		return i, err
	}
	if err = validateTaxConditions(i.TaxConditions); err != nil {
		return i, err
	}
	if i.LineItems, err = prepareLineItems(i.LineItems, false); err != nil {
		return i, err
	}
	if i.ShippingTerms, err = prepareShippingTerm(i.ShippingTerms); err != nil {
		return i, err
	}
	i.TotalPrice = prepareTotalPrice(i.TotalPrice)
	return i, nil
}

func requireDate(field, value string) (string, error) {
	if strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("%s is required", field)
	}
	normalized, err := NormalizeDate(value)
	if err != nil {
		return "", fmt.Errorf("%s: %w", field, err)
	}
	return normalized, nil
}

func validateAddress(address Address) error {
	if strings.TrimSpace(address.ContactID) != "" {
		return nil
	}
	if strings.TrimSpace(address.Name) == "" {
		return errors.New("address requires either contactId or name")
	}
	if strings.TrimSpace(address.CountryCode) == "" {
		return errors.New("address.countryCode is required when no contactId is given")
	}
	return nil
}

func validateTaxConditions(conditions TaxCondition) error {
	if !slices.Contains(validTaxTypes, conditions.TaxType) {
		return fmt.Errorf("taxConditions.taxType must be one of %s, got %q", strings.Join(validTaxTypes, ", "), conditions.TaxType)
	}
	return nil
}

// prepareLineItems validates line items and returns a copy with the default
// currency applied, leaving the caller's slice untouched.
func prepareLineItems(items []LineItem, allowOptional bool) ([]LineItem, error) {
	if len(items) == 0 {
		return nil, errors.New("at least one line item is required")
	}

	prepared := make([]LineItem, len(items))
	for index, item := range items {
		position := fmt.Sprintf("lineItems[%d]", index)
		if !slices.Contains(validLineItemTypes, item.Type) {
			return nil, fmt.Errorf("%s.type must be one of %s, got %q", position, strings.Join(validLineItemTypes, ", "), item.Type)
		}
		if strings.TrimSpace(item.Name) == "" {
			return nil, fmt.Errorf("%s.name is required", position)
		}
		if item.Optional && !allowOptional {
			return nil, fmt.Errorf("%s.optional is only supported for quotations", position)
		}

		if item.Type != "text" {
			if (item.Type == "material" || item.Type == "service") && strings.TrimSpace(item.ID) == "" {
				return nil, fmt.Errorf("%s.id must reference a Lexware article for type %s", position, item.Type)
			}
			if item.Quantity == 0 {
				return nil, fmt.Errorf("%s.quantity is required and must not be 0", position)
			}
			if strings.TrimSpace(item.UnitName) == "" {
				return nil, fmt.Errorf("%s.unitName is required", position)
			}
			if item.UnitPrice == nil || (item.UnitPrice.NetAmount == nil && item.UnitPrice.GrossAmount == nil) {
				return nil, fmt.Errorf("%s.unitPrice requires netAmount or grossAmount", position)
			}
			unitPrice := *item.UnitPrice
			if unitPrice.Currency == "" {
				unitPrice.Currency = defaultCurrency
			}
			item.UnitPrice = &unitPrice
		}

		prepared[index] = item
	}

	return prepared, nil
}

func prepareTotalPrice(total TotalPrice) TotalPrice {
	if total.Currency == "" {
		total.Currency = defaultCurrency
	}
	return total
}

func prepareShippingTerm(term ShippingTerm) (ShippingTerm, error) {
	if !slices.Contains(validShippingTypes, term.ShippingType) {
		return term, fmt.Errorf("shippingConditions.shippingType must be one of %s, got %q", strings.Join(validShippingTypes, ", "), term.ShippingType)
	}

	var err error
	if term.ShippingDate, err = NormalizeDate(term.ShippingDate); err != nil {
		return term, fmt.Errorf("shippingConditions.shippingDate: %w", err)
	}
	if term.ShippingEndDate, err = NormalizeDate(term.ShippingEndDate); err != nil {
		return term, fmt.Errorf("shippingConditions.shippingEndDate: %w", err)
	}

	switch term.ShippingType {
	case "service", "delivery":
		if term.ShippingDate == "" {
			return term, fmt.Errorf("shippingConditions.shippingDate is required for shippingType %s", term.ShippingType)
		}
	case "serviceperiod", "deliveryperiod":
		if term.ShippingDate == "" || term.ShippingEndDate == "" {
			return term, fmt.Errorf("shippingConditions.shippingDate and shippingEndDate are required for shippingType %s", term.ShippingType)
		}
		if isBefore(term.ShippingEndDate, term.ShippingDate) {
			return term, errors.New("shippingConditions.shippingEndDate must not be before shippingDate")
		}
	}

	return term, nil
}

// isBefore compares two timestamps produced by NormalizeDate.
func isBefore(value, reference string) bool {
	parsedValue, errValue := time.Parse(lexwareTimestampLayout, value)
	parsedReference, errReference := time.Parse(lexwareTimestampLayout, reference)
	if errValue != nil || errReference != nil {
		return false
	}
	return parsedValue.Before(parsedReference)
}
