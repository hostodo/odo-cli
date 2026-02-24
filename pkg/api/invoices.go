package api

import (
	"fmt"
	"net/url"
)

// ListInvoices retrieves all invoices with optional status filter
func (c *Client) ListInvoices(status string) ([]Invoice, error) {
	path := "/v1/billing/invoices/"
	if status != "" {
		path = fmt.Sprintf("%s?status=%s", path, url.QueryEscape(status))
	}

	resp, err := c.Get(path)
	if err != nil {
		return nil, err
	}

	var invoicesResp InvoicesResponse
	if err := parseResponse(resp, &invoicesResp); err != nil {
		return nil, err
	}

	return invoicesResp.Results, nil
}

// PayInvoice pays an invoice using the customer's default payment method
func (c *Client) PayInvoice(invoiceNumber string) (*PaymentResponse, error) {
	path := fmt.Sprintf("/v1/billing/invoices/%s/pay/", invoiceNumber)

	resp, err := c.Post(path, nil)
	if err != nil {
		return nil, err
	}

	var paymentResp PaymentResponse
	if err := parseResponse(resp, &paymentResp); err != nil {
		return nil, err
	}

	return &paymentResp, nil
}
