package ui

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/hostodo/odo-cli/v2/pkg/api"
)

// FormatInstancesJSON formats instances as JSON, redacting sensitive fields
func FormatInstancesJSON(instances []api.Instance) (string, error) {
	// Redact sensitive fields before serialization
	sanitized := make([]api.Instance, len(instances))
	copy(sanitized, instances)
	for i := range sanitized {
		sanitized[i].DefaultPassword = ""
	}
	data, err := json.MarshalIndent(sanitized, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return string(data), nil
}

// FormatInstancesSimpleTable formats instances as a simple ASCII table
func FormatInstancesSimpleTable(instances []api.Instance) string {
	if len(instances) == 0 {
		return "No instances found"
	}

	// Define column widths
	const (
		idWidth       = 12
		hostnameWidth = 25
		ipWidth       = 16
		statusWidth   = 14
		powerWidth    = 10
		ramWidth      = 8
		cpuWidth      = 6
		diskWidth     = 8
	)

	var sb strings.Builder

	// Header
	header := fmt.Sprintf(
		"%-*s  %-*s  %-*s  %-*s  %-*s  %*s  %*s  %*s",
		idWidth, "ID",
		hostnameWidth, "HOSTNAME",
		ipWidth, "IP ADDRESS",
		statusWidth, "STATUS",
		powerWidth, "POWER",
		ramWidth, "RAM (MB)",
		cpuWidth, "CPU",
		diskWidth, "DISK (GB)",
	)
	sb.WriteString(header + "\n")
	sb.WriteString(strings.Repeat("-", len(header)) + "\n")

	// Rows
	for _, instance := range instances {
		row := fmt.Sprintf(
			"%-*s  %-*s  %-*s  %-*s  %-*s  %*d  %*d  %*d",
			idWidth, truncate(instance.InstanceID, idWidth),
			hostnameWidth, truncate(instance.Hostname, hostnameWidth),
			ipWidth, truncate(instance.MainIP, ipWidth),
			statusWidth, truncate(instance.Status, statusWidth),
			powerWidth, truncate(instance.PowerStatus, powerWidth),
			ramWidth, instance.RAM,
			cpuWidth, instance.VCPU,
			diskWidth, instance.Disk,
		)
		sb.WriteString(row + "\n")
	}

	return sb.String()
}

// FormatInstancesDetailedTable formats instances with more details
func FormatInstancesDetailedTable(instances []api.Instance) string {
	if len(instances) == 0 {
		return "No instances found"
	}

	var sb strings.Builder

	for i, instance := range instances {
		if i > 0 {
			sb.WriteString("\n")
		}

		sb.WriteString(fmt.Sprintf("Instance: %s\n", instance.InstanceID))
		sb.WriteString(fmt.Sprintf("  Hostname:     %s\n", instance.Hostname))
		sb.WriteString(fmt.Sprintf("  IP Address:   %s\n", instance.MainIP))
		if len(instance.IPs) > 1 {
			sb.WriteString(fmt.Sprintf("  Additional:   %s\n", strings.Join(instance.IPs[1:], ", ")))
		}
		sb.WriteString(fmt.Sprintf("  Status:       %s\n", instance.Status))
		sb.WriteString(fmt.Sprintf("  Power:        %s\n", instance.PowerStatus))
		sb.WriteString(fmt.Sprintf("  Resources:    %d MB RAM, %d CPU, %d GB Disk\n",
			instance.RAM, instance.VCPU, instance.Disk))
		sb.WriteString(fmt.Sprintf("  Bandwidth:    %.2f / %d GB\n",
			instance.BandwidthUsage, instance.Bandwidth))
		sb.WriteString(fmt.Sprintf("  Plan:         %s\n", instance.Plan.Name))
		sb.WriteString(fmt.Sprintf("  Template:     %s\n", instance.Template.Name))
		sb.WriteString(fmt.Sprintf("  Region:       %s\n", instance.Node.Region))
		sb.WriteString(fmt.Sprintf("  Billing:      $%s / %s\n",
			instance.BillingAmount, instance.BillingCycle))
		sb.WriteString(fmt.Sprintf("  Next Due:     %s\n", instance.NextDueDate))
		if instance.IsSuspended {
			sb.WriteString(fmt.Sprintf("  Suspended:    Yes (%s)\n", instance.SuspensionReason))
		}
	}

	return sb.String()
}

// FormatInstanceDetail formats a single instance with full details
func FormatInstanceDetail(instance *api.Instance) string {
	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(TitleStyle.Render("Instance Details") + "\n\n")

	// Basic Info
	sb.WriteString(HeaderStyle.Render("Basic Information") + "\n")
	sb.WriteString(fmt.Sprintf("  ID:           %s\n", instance.InstanceID))
	sb.WriteString(fmt.Sprintf("  Hostname:     %s\n", instance.Hostname))
	sb.WriteString(fmt.Sprintf("  Status:       %s\n", GetPowerStatusBadge(instance.Status)))
	sb.WriteString(fmt.Sprintf("  Power:        %s\n", GetPowerStatusBadge(instance.PowerStatus)))
	sb.WriteString("\n")

	// Network
	sb.WriteString(HeaderStyle.Render("Network") + "\n")
	sb.WriteString(fmt.Sprintf("  Main IP:      %s\n", instance.MainIP))
	if len(instance.IPs) > 1 {
		sb.WriteString(fmt.Sprintf("  Additional:   %s\n", strings.Join(instance.IPs[1:], ", ")))
	}
	sb.WriteString(fmt.Sprintf("  MAC Address:  %s\n", instance.MAC))
	sb.WriteString("\n")

	// Resources
	sb.WriteString(HeaderStyle.Render("Resources") + "\n")
	sb.WriteString(fmt.Sprintf("  RAM:          %d MB\n", instance.RAM))
	sb.WriteString(fmt.Sprintf("  CPU:          %d cores\n", instance.VCPU))
	sb.WriteString(fmt.Sprintf("  Disk:         %d GB\n", instance.Disk))
	sb.WriteString(fmt.Sprintf("  Bandwidth:    %.2f / %d GB (%.1f%%)\n",
		instance.BandwidthUsage, instance.Bandwidth,
		(instance.BandwidthUsage/float64(instance.Bandwidth))*100))
	sb.WriteString("\n")

	// Plan & Template
	sb.WriteString(HeaderStyle.Render("Configuration") + "\n")
	sb.WriteString(fmt.Sprintf("  Plan:         %s\n", instance.Plan.Name))
	sb.WriteString(fmt.Sprintf("  Template:     %s\n", instance.Template.Name))
	sb.WriteString(fmt.Sprintf("  Region:       %s\n", instance.Node.Region))
	sb.WriteString(fmt.Sprintf("  Node:         %s\n", instance.Node.Name))
	sb.WriteString("\n")

	// Billing
	sb.WriteString(HeaderStyle.Render("Billing") + "\n")
	sb.WriteString(fmt.Sprintf("  Amount:       $%s / %s\n", instance.BillingAmount, instance.BillingCycle))
	sb.WriteString(fmt.Sprintf("  Next Due:     %s\n", instance.NextDueDate))
	sb.WriteString(fmt.Sprintf("  Auto-Renew:   %t\n", instance.AutorenewalEnabled))
	if instance.IsSuspended {
		sb.WriteString(fmt.Sprintf("  Suspended:    %s\n", ErrorStyle.Render("Yes - "+instance.SuspensionReason)))
	}
	sb.WriteString("\n")

	// Timestamps
	sb.WriteString(HeaderStyle.Render("Timeline") + "\n")
	sb.WriteString(fmt.Sprintf("  Created:      %s\n", instance.CreatedAt))

	return sb.String()
}

// truncate truncates a string to the specified length
func truncate(s string, length int) string {
	if len(s) <= length {
		return s
	}
	if length <= 3 {
		return s[:length]
	}
	return s[:length-3] + "..."
}

// FormatInvoicesTable formats invoices as an ASCII table
func FormatInvoicesTable(invoices []api.Invoice) string {
	if len(invoices) == 0 {
		return "No invoices found"
	}

	const (
		invoiceWidth  = 16
		amountWidth   = 12
		statusWidth   = 12
		dueWidth      = 12
		hostnameWidth = 20
		ipWidth       = 16
	)

	var sb strings.Builder

	// Header
	header := fmt.Sprintf(
		"%-*s  %*s  %-*s  %-*s  %-*s  %-*s",
		invoiceWidth, "INVOICE #",
		amountWidth, "AMOUNT",
		statusWidth, "STATUS",
		dueWidth, "DUE DATE",
		hostnameWidth, "HOSTNAME",
		ipWidth, "IP ADDRESS",
	)
	sb.WriteString(header + "\n")
	sb.WriteString(strings.Repeat("-", len(header)) + "\n")

	// Rows
	for _, invoice := range invoices {
		// Determine status display (check for overdue)
		statusDisplay := invoice.Status
		if invoice.Status == "unpaid" && invoice.DueDate != "" {
			// Parse due date and check if it's past
			dueDate, err := time.Parse("2006-01-02", invoice.DueDate)
			if err == nil && time.Now().After(dueDate) {
				statusDisplay = "Overdue"
			}
		}

		// Get hostname and IP from first instance
		hostname := "-"
		ipAddress := "-"
		if len(invoice.Instances) > 0 {
			if len(invoice.Instances) == 1 {
				hostname = invoice.Instances[0].Hostname
				ipAddress = invoice.Instances[0].MainIP
			} else {
				// Multiple instances - show comma-separated
				hostnames := make([]string, len(invoice.Instances))
				ips := make([]string, len(invoice.Instances))
				for i, inst := range invoice.Instances {
					hostnames[i] = inst.Hostname
					ips[i] = inst.MainIP
				}
				hostname = strings.Join(hostnames, ",")
				ipAddress = strings.Join(ips, ",")
			}
		}

		// Format amount with $ prefix
		amount := "$" + invoice.Subtotal

		row := fmt.Sprintf(
			"%-*s  %*s  %-*s  %-*s  %-*s  %-*s",
			invoiceWidth, truncate(invoice.InvoiceNumber, invoiceWidth),
			amountWidth, amount,
			statusWidth, truncate(statusDisplay, statusWidth),
			dueWidth, truncate(invoice.DueDate, dueWidth),
			hostnameWidth, truncate(hostname, hostnameWidth),
			ipWidth, truncate(ipAddress, ipWidth),
		)
		sb.WriteString(row + "\n")
	}

	return sb.String()
}

// FormatPaymentReceipt formats a payment receipt as a styled box
func FormatPaymentReceipt(invoiceNumber string, amount string, paymentMethod string, confirmationID string) string {
	now := time.Now().Format("2006-01-02 15:04:05 MST")

	content := fmt.Sprintf(`Payment Successful!

Invoice Number:    %s
Amount Paid:       $%s USD
Payment Method:    %s
Confirmation:      %s
Date:              %s

View details: https://console.hostodo.com/billing`,
		invoiceNumber,
		amount,
		paymentMethod,
		confirmationID,
		now,
	)

	receiptStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("42")).
		Padding(1, 2).
		Width(60)

	return "\n" + receiptStyle.Render(content) + "\n"
}

// SSHKeyDisplay holds formatted SSH key data for table display
type SSHKeyDisplay struct {
	Name        string
	Fingerprint string
	CreatedAt   string
}

// FormatSSHKeysTable formats SSH keys as an ASCII table
func FormatSSHKeysTable(displayKeys []SSHKeyDisplay) string {
	if len(displayKeys) == 0 {
		return "No SSH keys found"
	}

	const (
		nameWidth        = 20
		fingerprintWidth = 50
		dateWidth        = 12
	)

	var sb strings.Builder

	// Header
	header := fmt.Sprintf(
		"%-*s  %-*s  %-*s",
		nameWidth, "NAME",
		fingerprintWidth, "FINGERPRINT",
		dateWidth, "DATE ADDED",
	)
	sb.WriteString(header + "\n")
	sb.WriteString(strings.Repeat("-", len(header)) + "\n")

	// Rows
	for _, key := range displayKeys {
		row := fmt.Sprintf(
			"%-*s  %-*s  %-*s",
			nameWidth, truncate(key.Name, nameWidth),
			fingerprintWidth, truncate(key.Fingerprint, fingerprintWidth),
			dateWidth, truncate(key.CreatedAt, dateWidth),
		)
		sb.WriteString(row + "\n")
	}

	return sb.String()
}

func formatRAMGB(mb int) string {
	if mb <= 0 {
		return "0 GB"
	}
	if mb%1024 == 0 {
		return fmt.Sprintf("%d GB", mb/1024)
	}
	return fmt.Sprintf("%.1f GB", float64(mb)/1024.0)
}

func formatBandwidth(gb int) string {
	if gb <= 0 {
		return "0 GB"
	}
	if gb >= 1024 && gb%1024 == 0 {
		return fmt.Sprintf("%d TB", gb/1024)
	}
	if gb >= 1024 {
		return fmt.Sprintf("%.1f TB", float64(gb)/1024.0)
	}
	return fmt.Sprintf("%d GB", gb)
}

func usedOf(used, total int, unit string) string {
	if unit == "" {
		return fmt.Sprintf("%d/%d", used, total)
	}
	return fmt.Sprintf("%d/%d %s", used, total, unit)
}

// FormatPoolsTable formats Hostodo pools as an ASCII table.
func FormatPoolsTable(pools []api.ResourcePool) string {
	if len(pools) == 0 {
		return "No Hostodo pools found"
	}

	const (
		idWidth     = 16
		nameWidth   = 18
		statusWidth = 10
		ramWidth    = 14
		vcpuWidth   = 10
		diskWidth   = 12
		vmsWidth    = 8
		billWidth   = 16
	)

	var sb strings.Builder
	header := fmt.Sprintf(
		"%-*s  %-*s  %-*s  %-*s  %-*s  %-*s  %-*s  %-*s",
		idWidth, "POOL ID",
		nameWidth, "NAME",
		statusWidth, "STATUS",
		ramWidth, "RAM",
		vcpuWidth, "VCPU",
		diskWidth, "DISK",
		vmsWidth, "VMS",
		billWidth, "BILLING",
	)
	sb.WriteString(header + "\n")
	sb.WriteString(strings.Repeat("-", len(header)) + "\n")

	for _, pool := range pools {
		billing := "-"
		if pool.BillingAmount != "" {
			cycle := pool.BillingCycle
			if cycle == "" {
				cycle = "monthly"
			}
			billing = "$" + pool.BillingAmount + "/" + cycle
		}
		row := fmt.Sprintf(
			"%-*s  %-*s  %-*s  %-*s  %-*s  %-*s  %-*s  %-*s",
			idWidth, truncate(pool.PoolID, idWidth),
			nameWidth, truncate(pool.Label(), nameWidth),
			statusWidth, truncate(pool.Status, statusWidth),
			ramWidth, truncate(formatRAMGB(pool.Usage.RAMMB)+"/"+formatRAMGB(pool.Quota.RAMMB), ramWidth),
			vcpuWidth, truncate(usedOf(pool.Usage.VCPU, pool.Quota.VCPU, ""), vcpuWidth),
			diskWidth, truncate(usedOf(pool.Usage.DiskGB, pool.Quota.DiskGB, "GB"), diskWidth),
			vmsWidth, truncate(usedOf(pool.Usage.Instances, pool.Quota.Instances, ""), vmsWidth),
			billWidth, truncate(billing, billWidth),
		)
		sb.WriteString(row + "\n")
	}

	return sb.String()
}

// FormatPoolDetail formats a single Hostodo pool including members.
func FormatPoolDetail(pool *api.ResourcePoolDetail) string {
	var sb strings.Builder
	sb.WriteString(TitleStyle.Render("Pool: "+pool.Label()) + "\n")
	sb.WriteString(fmt.Sprintf("  ID:           %s\n", pool.PoolID))
	sb.WriteString(fmt.Sprintf("  Status:       %s\n", GetStatusStyle(pool.Status).Render(pool.Status)))
	if pool.BillingAmount != "" {
		cycle := pool.BillingCycle
		if cycle == "" {
			cycle = "monthly"
		}
		sb.WriteString(fmt.Sprintf("  Billing:      $%s / %s\n", pool.BillingAmount, cycle))
	}
	if pool.NextDueDate != "" {
		sb.WriteString(fmt.Sprintf("  Next due:     %s\n", pool.NextDueDate))
	}
	sb.WriteString(fmt.Sprintf("  Auto-renew:   %t\n", pool.AutorenewalEnabled))
	sb.WriteString("\n")

	sb.WriteString(HeaderStyle.Render("Quota") + "\n")
	sb.WriteString(fmt.Sprintf("  RAM:          %s / %s\n", formatRAMGB(pool.Usage.RAMMB), formatRAMGB(pool.Quota.RAMMB)))
	sb.WriteString(fmt.Sprintf("  vCPU:         %d / %d (max %d per VM)\n", pool.Usage.VCPU, pool.Quota.VCPU, pool.Quota.MaxVCPUPerInstance))
	sb.WriteString(fmt.Sprintf("  Disk:         %d / %d GB\n", pool.Usage.DiskGB, pool.Quota.DiskGB))
	sb.WriteString(fmt.Sprintf("  Bandwidth:    %s / %s\n", formatBandwidth(pool.Usage.BandwidthGB), formatBandwidth(pool.Quota.BandwidthGB)))
	sb.WriteString(fmt.Sprintf("  VMs:          %d / %d\n", pool.Usage.Instances, pool.Quota.Instances))
	sb.WriteString(fmt.Sprintf("  IPs:          %d / %d\n", pool.Usage.IPs, pool.Quota.IPs))
	sb.WriteString("\n")

	sb.WriteString(HeaderStyle.Render("Members") + "\n")
	if len(pool.Members) == 0 {
		sb.WriteString("  No VMs in this pool. Create one with: odo pools vm\n")
		return sb.String()
	}

	const (
		hostWidth   = 22
		ipWidth     = 16
		statusWidth = 12
		vcpuWidth   = 6
		ramWidth    = 8
		diskWidth   = 8
		regionWidth = 10
	)
	header := fmt.Sprintf(
		"  %-*s  %-*s  %-*s  %*s  %*s  %*s  %-*s",
		hostWidth, "HOSTNAME",
		ipWidth, "IP",
		statusWidth, "STATUS",
		vcpuWidth, "VCPU",
		ramWidth, "RAM",
		diskWidth, "DISK",
		regionWidth, "REGION",
	)
	sb.WriteString(header + "\n")
	sb.WriteString("  " + strings.Repeat("-", len(header)-2) + "\n")
	for _, m := range pool.Members {
		sb.WriteString(fmt.Sprintf(
			"  %-*s  %-*s  %-*s  %*d  %*s  %*s  %-*s\n",
			hostWidth, truncate(m.Hostname, hostWidth),
			ipWidth, truncate(m.MainIP, ipWidth),
			statusWidth, truncate(m.Status, statusWidth),
			vcpuWidth, m.VCPU,
			ramWidth, formatRAMGB(m.RAM),
			diskWidth, fmt.Sprintf("%d GB", m.Disk),
			regionWidth, truncate(m.Region, regionWidth),
		))
	}
	return sb.String()
}

// FormatPoolOptionsTable formats pool tiers as an ASCII table.
func FormatPoolOptionsTable(options *api.PoolOptionsResponse) string {
	if options == nil || len(options.Tiers) == 0 {
		return "No Hostodo pool tiers available"
	}

	var sb strings.Builder
	if options.CurrentPoolID != "" {
		sb.WriteString(fmt.Sprintf("Current pool: %s\n\n", options.CurrentPoolID))
	} else {
		sb.WriteString("No active Hostodo pool.\n\n")
	}

	const (
		idWidth    = 6
		nameWidth  = 18
		flagWidth  = 12
		ramWidth   = 8
		vcpuWidth  = 6
		diskWidth  = 8
		vmsWidth   = 6
		priceWidth = 10
	)
	header := fmt.Sprintf(
		"%-*s  %-*s  %-*s  %-*s  %-*s  %-*s  %-*s  %*s",
		idWidth, "ID",
		nameWidth, "NAME",
		flagWidth, "FLAG",
		ramWidth, "RAM",
		vcpuWidth, "VCPU",
		diskWidth, "DISK",
		vmsWidth, "VMS",
		priceWidth, "PRICE/MO",
	)
	sb.WriteString(header + "\n")
	sb.WriteString(strings.Repeat("-", len(header)) + "\n")
	for _, tier := range options.Tiers {
		flag := tier.Flag
		if flag == "" && tier.IsCurrent {
			flag = "current"
		}
		if flag == "" {
			flag = "available"
		}
		sb.WriteString(fmt.Sprintf(
			"%-*s  %-*s  %-*s  %-*s  %-*s  %-*s  %-*s  %*s\n",
			idWidth, fmt.Sprintf("%d", tier.ID),
			nameWidth, truncate(tier.Name, nameWidth),
			flagWidth, truncate(flag, flagWidth),
			ramWidth, truncate(formatRAMGB(tier.RAMMB), ramWidth),
			vcpuWidth, fmt.Sprintf("%d", tier.TotalVCPU),
			diskWidth, fmt.Sprintf("%d GB", tier.DiskGB),
			vmsWidth, fmt.Sprintf("%d", tier.MaxInstances),
			priceWidth, "$"+tier.PriceMonthly,
		))
	}
	return sb.String()
}
