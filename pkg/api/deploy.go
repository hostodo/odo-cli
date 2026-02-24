package api

// ListPlans retrieves all available VPS plans
func (c *Client) ListPlans() ([]Plan, error) {
	path := "/client/plans/"

	resp, err := c.Get(path)
	if err != nil {
		return nil, err
	}

	var plansResp PlansResponse
	if err := parseResponse(resp, &plansResp); err != nil {
		return nil, err
	}

	// Filter out disabled and out-of-stock plans
	var availablePlans []Plan
	for _, plan := range plansResp.Results {
		if plan.Enabled && !plan.OutOfStock {
			availablePlans = append(availablePlans, plan)
		}
	}

	return availablePlans, nil
}

// ListRegions retrieves all available regions
func (c *Client) ListRegions() ([]Region, error) {
	path := "/client/regions/"

	resp, err := c.Get(path)
	if err != nil {
		return nil, err
	}

	var regionsResp RegionsResponse
	if err := parseResponse(resp, &regionsResp); err != nil {
		return nil, err
	}

	// Filter out out-of-stock regions
	var availableRegions []Region
	for _, region := range regionsResp.Results {
		if !region.OutOfStock {
			availableRegions = append(availableRegions, region)
		}
	}

	return availableRegions, nil
}

// ListTemplates retrieves all available OS templates
func (c *Client) ListTemplates() ([]Template, error) {
	path := "/client/templates/"

	resp, err := c.Get(path)
	if err != nil {
		return nil, err
	}

	var templatesResp TemplatesResponse
	if err := parseResponse(resp, &templatesResp); err != nil {
		return nil, err
	}

	return templatesResp.Results, nil
}

// ListPaymentMethods retrieves all saved payment methods
func (c *Client) ListPaymentMethods() ([]PaymentMethod, error) {
	path := "/v1/billing/payment-methods/"

	resp, err := c.Get(path)
	if err != nil {
		return nil, err
	}

	var pmResp PaymentMethodsResponse
	if err := parseResponse(resp, &pmResp); err != nil {
		return nil, err
	}

	return pmResp.Results, nil
}

// GetDefaultPaymentMethod retrieves the default payment method
func (c *Client) GetDefaultPaymentMethod() (*PaymentMethod, error) {
	paymentMethods, err := c.ListPaymentMethods()
	if err != nil {
		return nil, err
	}

	for _, pm := range paymentMethods {
		if pm.CustomerDefault {
			return &pm, nil
		}
	}

	return nil, nil
}

// GetQuote retrieves a price quote for a deployment
func (c *Client) GetQuote(req QuoteRequest) (*QuoteResponse, error) {
	path := "/client/orders/price/"

	resp, err := c.Post(path, req)
	if err != nil {
		return nil, err
	}

	var quoteResp QuoteResponse
	if err := parseResponse(resp, &quoteResp); err != nil {
		return nil, err
	}

	return &quoteResp, nil
}

// CreateDeployOrder creates a new instance deployment order
func (c *Client) CreateDeployOrder(req DeployRequest) (*DeployResponse, error) {
	path := "/client/orders/deploy_instance/"

	resp, err := c.Post(path, req)
	if err != nil {
		return nil, err
	}

	var deployResp DeployResponse
	if err := parseResponse(resp, &deployResp); err != nil {
		return nil, err
	}

	return &deployResp, nil
}

// CheckHostnameExists checks if a hostname is already in use
func (c *Client) CheckHostnameExists(hostname string) (bool, error) {
	// Use ListInstances to get all instances and check for hostname collision
	instancesResp, err := c.ListInstances(1000, 0)
	if err != nil {
		return false, err
	}

	for _, instance := range instancesResp.Results {
		if instance.Hostname == hostname {
			return true, nil
		}
	}

	return false, nil
}
