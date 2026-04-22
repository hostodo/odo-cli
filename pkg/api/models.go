package api

import "encoding/json"

// LoginRequest represents the login credentials
type LoginRequest struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	RememberMe bool   `json:"remember_me"`
}

// LoginResponse represents the authentication response
type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	UserID       int    `json:"user_id"`
	Email        string `json:"email"`
}

// User represents the authenticated user
type User struct {
	ID        int    `json:"id"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

// Plan represents a VPS plan
type Plan struct {
	ID                int    `json:"id"`
	Name              string `json:"name"`
	RAM               int    `json:"ram"`
	VCPU              int    `json:"vcpu"`
	Disk              int    `json:"disk"`
	Bandwidth         int    `json:"bandwidth"`
	PriceMonthly      string `json:"price_monthly"`
	PriceAnnually     string `json:"price_annually"`
	PriceSemiannually string `json:"price_semiannually"`
	PriceBiennially   string `json:"price_biennially"`
	PriceTriennially  string `json:"price_triennially"`
	Enabled           bool   `json:"show_on_frontend"`
	OutOfStock        bool   `json:"out_of_stock"`
	PlanCategoryID    int    `json:"plan_category_id"`
}

// Template represents an OS template
type Template struct {
	ID              int    `json:"id"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	DefaultUsername string `json:"default_username"`
	LogoSVGURL      string `json:"logo_svg_url"`
}

// Node represents a Proxmox node
type Node struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Region string `json:"region"`
}

// Instance represents a VPS instance
type Instance struct {
	InstanceID               string   `json:"instance_id"`
	Hostname                 string   `json:"hostname"`
	MainIP                   string   `json:"main_ip"`
	Status                   string   `json:"status"`
	PowerStatus              string   `json:"power_status"`
	AutorenewalEnabled       bool     `json:"autorenewal_enabled"`
	AutoRenewalPaymentMethod string   `json:"auto_renewal_payment_method"`
	IPs                      []string `json:"ips"`
	BandwidthUsage           float64  `json:"bandwidth_usage"`
	RAM                      int      `json:"ram"`
	VCPU                     int      `json:"vcpu"`
	Disk                     int      `json:"disk"`
	Bandwidth                int      `json:"bandwidth"`
	IsSuspended              bool     `json:"is_suspended"`
	SuspensionReason         string   `json:"suspension_reason"`
	MAC                      string   `json:"mac"`
	BillingCycle             string   `json:"billing_cycle"`
	BillingAmount            string   `json:"billing_amount"`
	NextDueDate              string   `json:"next_due_date"`
	CreatedAt                string   `json:"created_at"`
	UpdatedAt                string   `json:"updated_at"`
	DefaultPassword          string   `json:"default_password,omitempty"`
	ProxID                   int      `json:"prox_id"`
	Plan                     Plan     `json:"plan"`
	Template                 Template `json:"template"`
	Node                     Node     `json:"node"`
}

// InstancesResponse represents the paginated instances response
type InstancesResponse struct {
	Count    int        `json:"count"`
	Next     *string    `json:"next"`
	Previous *string    `json:"previous"`
	Results  []Instance `json:"results"`
}

// InstanceDetailResponse represents a single instance response
type InstanceDetailResponse struct {
	Instance Instance `json:"instance"`
}

// PowerStatusResponse represents the power status response
type PowerStatusResponse struct {
	PowerStatus string `json:"power_status"`
}

// PowerControlRequest represents a power control action
type PowerControlRequest struct {
	Action string `json:"action"` // "start", "stop", "reboot"
}

// CLISession represents an active CLI session
type CLISession struct {
	ID         int     `json:"id"`
	DeviceName string  `json:"device_name"`
	CreatedAt  string  `json:"created_at"`
	LastUsedAt string  `json:"last_used_at"`
	LoginIP    string  `json:"login_ip"`
	UserAgent  string  `json:"user_agent"`
	RevokedAt  *string `json:"revoked_at"`
}

// CLISessionsResponse is the paginated response for sessions
type CLISessionsResponse struct {
	Results []CLISession `json:"results"`
	Count   int          `json:"count"`
}

// ErrorResponse represents an API error
type ErrorResponse struct {
	Detail  string `json:"detail"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// PlanCategory represents a plan category associated with a region
type PlanCategory struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// Region represents a VPS region/location
type Region struct {
	ID             int            `json:"id"`
	Name           string         `json:"name"`
	Description    string         `json:"description"`
	OutOfStock     bool           `json:"out_of_stock"`
	PlanCategories []PlanCategory `json:"plan_categories"`
}

// PaymentMethod represents a saved payment method
type PaymentMethod struct {
	PaymentMethodID string `json:"payment_method_id"`
	LastFour        string `json:"last_four"`
	CardType        string `json:"card_type"`
	CustomerDefault bool   `json:"customer_default"`
	ExpiryMonth     int    `json:"expiry_month"`
	ExpiryYear      int    `json:"expiry_year"`
}

// PaymentMethodsResponse represents the paginated payment methods response
type PaymentMethodsResponse struct {
	Results []PaymentMethod `json:"results"`
	Count   int             `json:"count"`
}

// QuoteRequest represents a price quote request
type QuoteRequest struct {
	PlanID       int    `json:"plan_id"`
	BillingCycle string `json:"billing_cycle"`
	Quantity     int    `json:"quantity"`
	Promocode    string `json:"promocode,omitempty"`
}

// QuoteResponse represents a price quote response
type QuoteResponse struct {
	AmountDue json.Number `json:"amount_due"`
	UnitPrice json.Number `json:"unit_price"`
	Quantity  int         `json:"quantity"`
}

// DeployRequest represents an instance deployment request
type DeployRequest struct {
	Hostname        string `json:"hostname"`
	Region          string `json:"region"`
	Template        string `json:"template"`
	Plan            string `json:"plan"`
	BillingCycle    string `json:"billing_cycle"`
	SSHKey          string `json:"ssh_key,omitempty"`
	PaymentMethodID string `json:"payment_method_id,omitempty"`
	Promocode       string `json:"promocode,omitempty"`
	Quantity        int    `json:"quantity"`
}

// DeployResponse represents the response after creating a deployment order
type DeployResponse struct {
	Order struct {
		OrderNumber   string `json:"order_number"`
		Status        string `json:"status"`
		BillingAmount string `json:"billing_amount"`
		Hostname      string `json:"hostname"`
	} `json:"order"`
	Invoice struct {
		InvoiceNumber string `json:"invoice_number"`
		Status        string `json:"status"`
		Subtotal      string `json:"subtotal"`
	} `json:"invoice"`
	CheckoutURL string `json:"checkout_url"`
}

// PlansResponse represents the paginated plans response
type PlansResponse struct {
	Results []Plan `json:"results"`
}

// RegionsResponse represents the paginated regions response
type RegionsResponse struct {
	Results []Region `json:"results"`
}

// TemplatesResponse represents the paginated templates response
type TemplatesResponse struct {
	Results []Template `json:"results"`
}

// Invoice represents a billing invoice
type Invoice struct {
	InvoiceNumber string `json:"invoice_number"`
	Status        string `json:"status"`
	DueDate       string `json:"due_date"`
	Subtotal      string `json:"subtotal"`
	CreatedAt     string `json:"created_at"`
	Instances     []struct {
		Hostname string `json:"hostname"`
		MainIP   string `json:"main_ip"`
	} `json:"instances"`
}

// InvoicesResponse represents the paginated invoices response
type InvoicesResponse struct {
	Results []Invoice `json:"results"`
	Count   int       `json:"count"`
}

// PaymentResponse represents the response after paying an invoice
type PaymentResponse struct {
	TransactionID      string `json:"transaction_id"`
	Amount             string `json:"amount"`
	BillingIntegration string `json:"billing_integration"`
	StripeCheckoutURL  string `json:"stripe_checkout_url,omitempty"`
	Status             string `json:"status"`
}

// EventLog represents a provisioning event
type EventLog struct {
	ID                 int    `json:"id"`
	InstanceID         int    `json:"instance_id"`
	ClientEventMessage string `json:"client_event_message"`
	Status             string `json:"status"`
	CreatedAt          string `json:"created_at"`
}

// EventsResponse represents the events endpoint response
type EventsResponse struct {
	Events []EventLog `json:"events"`
}

// SSHKey represents an SSH public key
type SSHKey struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	PublicKey   string `json:"public_key"`
	Fingerprint string `json:"fingerprint"`
	CreatedAt   string `json:"created_at"`
}
