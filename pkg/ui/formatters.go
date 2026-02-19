package ui

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/hostodo/hostodo-cli/pkg/api"
)

// FormatInstancesJSON formats instances as JSON, omitting sensitive fields
func FormatInstancesJSON(instances []api.Instance) (string, error) {
	// Strip sensitive fields before marshaling
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
	sb.WriteString(fmt.Sprintf("  Updated:      %s\n", instance.UpdatedAt))

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
			if err == nil && time.Now().After(dueDate.AddDate(0, 0, 1)) {
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
