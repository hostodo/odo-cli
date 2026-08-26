package api

import (
	"encoding/json"
	"strings"
)

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

// Department represents a support department.
type Department struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
}

// TicketUser is the compact user payload embedded in ticket responses.
type TicketUser struct {
	ID        int    `json:"id"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Role      int    `json:"role"`
}

// TicketReply represents a helpdesk ticket reply.
type TicketReply struct {
	ID           int        `json:"id"`
	Content      string     `json:"content"`
	User         TicketUser `json:"user"`
	InternalNote bool       `json:"internal_note"`
	IsStaffReply bool       `json:"is_staff_reply"`
	CreatedAt    string     `json:"created_at"`
	UpdatedAt    string     `json:"updated_at"`
}

// Ticket represents a helpdesk ticket.
type Ticket struct {
	ID         int           `json:"id"`
	TicketID   string        `json:"ticket_id"`
	Subject    string        `json:"subject"`
	Content    string        `json:"content"`
	Status     string        `json:"status"`
	Priority   string        `json:"priority"`
	Department Department    `json:"department"`
	User       TicketUser    `json:"user"`
	Replies    []TicketReply `json:"replies"`
	CreatedAt  string        `json:"created_at"`
	UpdatedAt  string        `json:"updated_at"`
}

// TicketsResponse represents the paginated tickets response.
type TicketsResponse struct {
	Results []Ticket `json:"results"`
	Count   int      `json:"count"`
}

// DepartmentsResponse represents the paginated departments response.
type DepartmentsResponse struct {
	Results []Department `json:"results"`
	Count   int          `json:"count"`
}

// TicketCreateRequest represents the request to open a support ticket.
type TicketCreateRequest struct {
	Subject      string `json:"subject"`
	Content      string `json:"content"`
	DepartmentID int    `json:"department_id"`
	Priority     string `json:"priority,omitempty"`
}

// TicketReplyRequest represents the request to reply to a support ticket.
type TicketReplyRequest struct {
	Content      string `json:"content"`
	InternalNote bool   `json:"internal_note,omitempty"`
}

// PoolQuota is used/remaining/limit capacity for a Hostodo pool.
type PoolQuota struct {
	Instances          int `json:"instances"`
	VCPU               int `json:"vcpu"`
	RAMMB              int `json:"ram_mb"`
	DiskGB             int `json:"disk_gb"`
	BandwidthGB        int `json:"bandwidth_gb"`
	IPs                int `json:"ips"`
	MaxVCPUPerInstance int `json:"max_vcpu_per_instance,omitempty"`
}

// ResourcePool is a Hostodo capacity pool summary.
type ResourcePool struct {
	PoolID             string    `json:"pool_id"`
	DisplayName        string    `json:"display_name"`
	Status             string    `json:"status"`
	Enforcement        string    `json:"enforcement"`
	PlanID             int       `json:"plan_id"`
	BillingAmount      string    `json:"billing_amount"`
	BillingCycle       string    `json:"billing_cycle"`
	NextDueDate        string    `json:"next_due_date"`
	AutorenewalEnabled bool      `json:"autorenewal_enabled"`
	Quota              PoolQuota `json:"quota"`
	Usage              PoolQuota `json:"usage"`
	Remaining          PoolQuota `json:"remaining"`
	CreatedAt          string    `json:"created_at"`
	UpdatedAt          string    `json:"updated_at"`
}

// Label is the customer-facing pool name, falling back to pool_id.
func (p ResourcePool) Label() string {
	name := strings.TrimSpace(p.DisplayName)
	if name != "" {
		return name
	}
	return p.PoolID
}

// ResourcePoolMember is a VM billed against a pool.
type ResourcePoolMember struct {
	InstanceID string `json:"instance_id"`
	Hostname   string `json:"hostname"`
	Status     string `json:"status"`
	MainIP     string `json:"main_ip"`
	VCPU       int    `json:"vcpu"`
	RAM        int    `json:"ram"`
	Disk       int    `json:"disk"`
	Bandwidth  int    `json:"bandwidth"`
	Region     string `json:"region"`
	PlanName   string `json:"plan_name"`
}

// ResourcePoolDetail is a pool plus member VMs and region counts.
type ResourcePoolDetail struct {
	ResourcePool
	Members           []ResourcePoolMember `json:"members"`
	Regions           []PoolRegionCount    `json:"regions"`
	DowngradeBlockers []string             `json:"downgrade_blockers"`
}

// PoolRegionCount is how many pool VMs live in a region.
type PoolRegionCount struct {
	Region string `json:"region"`
	Count  int    `json:"count"`
}

// ResourcePoolsResponse is the paginated pool list.
type ResourcePoolsResponse struct {
	Results []ResourcePool `json:"results"`
	Count   int            `json:"count"`
}

// PoolTier is a buyable/upgradable Hostodo pool plan.
type PoolTier struct {
	ID                 int     `json:"id"`
	Name               string  `json:"name"`
	PriceMonthly       string  `json:"price_monthly"`
	PriceAnnually      string  `json:"price_annually"`
	PriceSemiannually  string  `json:"price_semiannually"`
	PriceBiennially    string  `json:"price_biennially"`
	PriceTriennially   string  `json:"price_triennially"`
	RAMMB              int     `json:"ram_mb"`
	TotalVCPU          int     `json:"total_vcpu"`
	MaxVCPUPerInstance int     `json:"max_vcpu_per_instance"`
	DiskGB             int     `json:"disk_gb"`
	BandwidthGB        int     `json:"bandwidth_gb"`
	MaxInstances       int     `json:"max_instances"`
	MaxIPs             int     `json:"max_ips"`
	DollarsPerGB       float64 `json:"dollars_per_gb"`
	SelfServe          bool    `json:"self_serve"`
	Flag               string  `json:"flag"`
	IsCurrent          bool    `json:"is_current"`
}

// PoolOptionsResponse is the pool catalog plus the caller's current pool.
type PoolOptionsResponse struct {
	BillingCycles []string   `json:"billing_cycles"`
	CurrentPoolID string     `json:"current_pool_id"`
	Tiers         []PoolTier `json:"tiers"`
}

// PoolCheckoutRequest buys or upgrades a Hostodo pool.
type PoolCheckoutRequest struct {
	PlanID          int    `json:"plan_id"`
	TargetPlanID    int    `json:"target_plan_id,omitempty"`
	BillingCycle    string `json:"billing_cycle,omitempty"`
	PaymentMethod   string `json:"payment_method,omitempty"`
	PaymentMethodID string `json:"payment_method_id,omitempty"`
	Promocode       string `json:"promocode,omitempty"`
	IdempotencyKey  string `json:"idempotency_key,omitempty"`
	QuoteOnly       bool   `json:"quote_only,omitempty"`
	Confirm         bool   `json:"confirm,omitempty"`
}

// PoolQuote is a purchase/upgrade price quote.
type PoolQuote struct {
	PlanID               int         `json:"plan_id"`
	PlanName             string      `json:"plan_name"`
	BillingCycle         string      `json:"billing_cycle"`
	Mode                 string      `json:"mode"`
	ExistingPoolID       string      `json:"existing_pool_id"`
	UnitPrice            json.Number `json:"unit_price"`
	Subtotal             json.Number `json:"subtotal"`
	RecurringAmount      json.Number `json:"recurring_amount"`
	CreditsAvailable     json.Number `json:"credits_available"`
	CreditsApplied       json.Number `json:"credits_applied_if_created"`
	AmountDueAfterCredit json.Number `json:"amount_due_after_credit"`
	PromocodeApplied     bool        `json:"promocode_applied"`
	InvoiceDate          string      `json:"invoice_date"`
	NextDueDate          string      `json:"next_due_date"`
	Quota                PoolQuota   `json:"quota"`
}

// PoolCheckoutResponse is returned after buying or upgrading a pool.
type PoolCheckoutResponse struct {
	Mode           string                 `json:"mode"`
	OrderNumber    string                 `json:"order_number"`
	InvoiceNumber  string                 `json:"invoice_number"`
	AmountDue      string                 `json:"amount_due"`
	UnitPrice      string                 `json:"unit_price"`
	PlanID         int                    `json:"plan_id"`
	PlanName       string                 `json:"plan_name"`
	PoolID         string                 `json:"pool_id"`
	ExistingPoolID string                 `json:"existing_pool_id"`
	CheckoutURL    string                 `json:"checkout_url"`
	Checkout       map[string]interface{} `json:"checkout"`
	PaymentMethod  string                 `json:"payment_method"`
}

// CreatePoolVMRequest creates a $0 gen2 VM inside a pool.
type CreatePoolVMRequest struct {
	PoolID      string `json:"pool_id"`
	Hostname    string `json:"hostname"`
	RegionID    int    `json:"region_id,omitempty"`
	Region      string `json:"region,omitempty"`
	TemplateID  int    `json:"template_id"`
	VCPU        int    `json:"vcpu,omitempty"`
	RAMMB       int    `json:"ram_mb,omitempty"`
	DiskGB      int    `json:"disk_gb,omitempty"`
	BandwidthGB int    `json:"bandwidth_gb,omitempty"`
	PlanID      int    `json:"plan_id,omitempty"`
	SSHKeyID    int    `json:"ssh_key_id,omitempty"`
}

// CreatePoolVMResponse is returned after creating a pool VM.
type CreatePoolVMResponse struct {
	Instance struct {
		InstanceID string `json:"instance_id"`
		PoolID     string `json:"pool_id"`
		Status     string `json:"status"`
		Hostname   string `json:"hostname"`
		MainIP     string `json:"main_ip"`
		Bandwidth  int    `json:"bandwidth"`
	} `json:"instance"`
	Quota struct {
		Used      PoolQuota `json:"used"`
		Remaining PoolQuota `json:"remaining"`
	} `json:"quota"`
}
