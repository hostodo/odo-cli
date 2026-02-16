package api

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
	ID           int     `json:"id"`
	Name         string  `json:"name"`
	RAM          int     `json:"ram"`
	VCPU         int     `json:"vcpu"`
	Disk         int     `json:"disk"`
	Bandwidth    int     `json:"bandwidth"`
	PriceMonthly string  `json:"price_monthly"`
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
	InstanceID                 string    `json:"instance_id"`
	Hostname                   string    `json:"hostname"`
	MainIP                     string    `json:"main_ip"`
	Status                     string    `json:"status"`
	PowerStatus                string    `json:"power_status"`
	AutorenewalEnabled         bool      `json:"autorenewal_enabled"`
	AutoRenewalPaymentMethod   string    `json:"auto_renewal_payment_method"`
	IPs                        []string  `json:"ips"`
	BandwidthUsage             float64   `json:"bandwidth_usage"`
	RAM                        int       `json:"ram"`
	VCPU                       int       `json:"vcpu"`
	Disk                       int       `json:"disk"`
	Bandwidth                  int       `json:"bandwidth"`
	IsSuspended                bool      `json:"is_suspended"`
	SuspensionReason           string    `json:"suspension_reason"`
	MAC                        string    `json:"mac"`
	BillingCycle               string    `json:"billing_cycle"`
	BillingAmount              string    `json:"billing_amount"`
	NextDueDate                string    `json:"next_due_date"`
	CreatedAt                  string `json:"created_at"`
	UpdatedAt                  string `json:"updated_at"`
	ProxID                     int       `json:"prox_id"`
	Plan                       Plan      `json:"plan"`
	Template                   Template  `json:"template"`
	Node                       Node      `json:"node"`
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
