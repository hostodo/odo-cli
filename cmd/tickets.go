package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/hostodo/odo-cli/v2/pkg/api"
	"github.com/hostodo/odo-cli/v2/pkg/auth"
	"github.com/hostodo/odo-cli/v2/pkg/config"
	"github.com/hostodo/odo-cli/v2/pkg/ui"
	"github.com/spf13/cobra"
)

var (
	ticketStatusFlag     string
	ticketMessageFlag    string
	ticketPriorityFlag   string
	ticketDepartmentFlag int
	ticketInternalFlag   bool
	ticketJSONFlag       bool
	ticketSimpleFlag     bool
)

var ticketsCmd = &cobra.Command{
	Use:     "tickets",
	Short:   "Open and reply to support tickets",
	Aliases: []string{"ticket", "support"},
	Long: `Open and manage Hostodo support tickets.

Examples:
  odo tickets list
  odo tickets open "Need rDNS set" --message "Please set rDNS on 1.2.3.4" --department-id 1
  odo tickets reply TCK-123456 --message "Thanks, that fixed it"
  echo "Reply body" | odo tickets reply TCK-123456`,
}

var ticketsListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List support tickets",
	Aliases: []string{"ls"},
	RunE:    runTicketsList,
}

var ticketsDepartmentsCmd = &cobra.Command{
	Use:     "departments",
	Short:   "List support departments",
	Aliases: []string{"depts"},
	RunE:    runTicketsDepartments,
}

var ticketsOpenCmd = &cobra.Command{
	Use:   "open [subject]",
	Short: "Open a support ticket",
	Args:  cobra.ExactArgs(1),
	RunE:  runTicketsOpen,
}

var ticketsReplyCmd = &cobra.Command{
	Use:   "reply [ticket-id]",
	Short: "Reply to a support ticket",
	Args:  cobra.ExactArgs(1),
	RunE:  runTicketsReply,
}

var ticketsShowCmd = &cobra.Command{
	Use:     "show [ticket-id]",
	Short:   "Show ticket details and replies",
	Aliases: []string{"view"},
	Args:    cobra.ExactArgs(1),
	RunE:    runTicketsShow,
}

func init() {
	ticketsCmd.AddCommand(ticketsListCmd, ticketsDepartmentsCmd, ticketsOpenCmd, ticketsReplyCmd, ticketsShowCmd)
	ticketsListCmd.Flags().StringVar(&ticketStatusFlag, "status", "", "Filter by status (open, waiting_on_staff, waiting_on_client, closed)")
	ticketsListCmd.Flags().BoolVar(&ticketJSONFlag, "json", false, "Output as JSON")
	ticketsListCmd.Flags().BoolVar(&ticketSimpleFlag, "simple", false, "Output as simple table")
	ticketsDepartmentsCmd.Flags().BoolVar(&ticketJSONFlag, "json", false, "Output as JSON")
	ticketsOpenCmd.Flags().StringVarP(&ticketMessageFlag, "message", "m", "", "Ticket message body (defaults to stdin or prompt)")
	ticketsOpenCmd.Flags().IntVar(&ticketDepartmentFlag, "department-id", 0, "Department ID (see 'odo tickets departments')")
	ticketsOpenCmd.Flags().StringVar(&ticketPriorityFlag, "priority", "medium", "Priority (low, medium, high, critical)")
	ticketsReplyCmd.Flags().StringVarP(&ticketMessageFlag, "message", "m", "", "Reply message body (defaults to stdin or prompt)")
	ticketsReplyCmd.Flags().BoolVar(&ticketInternalFlag, "internal", false, "Create an internal note (admin only)")
}

func newAuthenticatedClient() (*api.Client, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	if !auth.IsAuthenticated() {
		return nil, fmt.Errorf("not authenticated. Run 'odo login' first")
	}
	client, err := api.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}
	return client, nil
}

func readBody(message string) (string, error) {
	message = strings.TrimSpace(message)
	if message != "" {
		return message, nil
	}

	if stat, err := os.Stdin.Stat(); err == nil && (stat.Mode()&os.ModeCharDevice) == 0 {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", err
		}
		message = strings.TrimSpace(string(data))
		if message != "" {
			return message, nil
		}
	}

	fmt.Print("Message: ")
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	message = strings.TrimSpace(line)
	if message == "" {
		return "", fmt.Errorf("message is required")
	}
	return message, nil
}

func runTicketsList(cmd *cobra.Command, args []string) error {
	client, err := newAuthenticatedClient()
	if err != nil {
		return err
	}

	tickets, err := client.ListTickets(ticketStatusFlag)
	if err != nil {
		return fmt.Errorf("failed to list tickets: %w", err)
	}
	if len(tickets) == 0 {
		fmt.Println("No tickets found.")
		return nil
	}

	if ticketJSONFlag {
		data, err := json.MarshalIndent(tickets, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}

	if ticketSimpleFlag {
		fmt.Printf("%-14s  %-20s  %-10s  %-10s  %s\n", "ID", "DEPARTMENT", "STATUS", "PRIORITY", "SUBJECT")
		for _, t := range tickets {
			fmt.Printf("%-14s  %-20s  %-10s  %-10s  %s\n",
				displayTicketID(t), t.Department.Name, t.Status, t.Priority, t.Subject)
		}
		return nil
	}

	// Default: tab-separated
	for _, t := range tickets {
		fmt.Printf("%s\t%s\t%s\t%s\t%s\n", displayTicketID(t), t.Status, t.Priority, t.Department.Name, t.Subject)
	}
	return nil
}

func runTicketsDepartments(cmd *cobra.Command, args []string) error {
	client, err := newAuthenticatedClient()
	if err != nil {
		return err
	}

	departments, err := client.ListDepartments()
	if err != nil {
		return fmt.Errorf("failed to list departments: %w", err)
	}

	if ticketJSONFlag {
		data, err := json.MarshalIndent(departments, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}

	for _, d := range departments {
		fmt.Printf("%d\t%s\t%s\n", d.ID, d.Name, d.Description)
	}
	return nil
}

func runTicketsOpen(cmd *cobra.Command, args []string) error {
	if ticketDepartmentFlag == 0 {
		return fmt.Errorf("--department-id is required (run 'odo tickets departments')")
	}
	body, err := readBody(ticketMessageFlag)
	if err != nil {
		return err
	}
	client, err := newAuthenticatedClient()
	if err != nil {
		return err
	}

	ticket, err := client.CreateTicket(api.TicketCreateRequest{
		Subject:      args[0],
		Content:      body,
		DepartmentID: ticketDepartmentFlag,
		Priority:     ticketPriorityFlag,
	})
	if err != nil {
		return fmt.Errorf("failed to open ticket: %w", err)
	}
	fmt.Println(ui.SuccessStyle.Render("✓ Ticket opened: " + displayTicketID(*ticket)))
	fmt.Println(ticket.Subject)
	return nil
}

func runTicketsReply(cmd *cobra.Command, args []string) error {
	body, err := readBody(ticketMessageFlag)
	if err != nil {
		return err
	}
	client, err := newAuthenticatedClient()
	if err != nil {
		return err
	}

	reply, err := client.ReplyToTicket(args[0], api.TicketReplyRequest{Content: body, InternalNote: ticketInternalFlag})
	if err != nil {
		return fmt.Errorf("failed to reply to ticket: %w", err)
	}
	fmt.Println(ui.SuccessStyle.Render(fmt.Sprintf("✓ Reply added: #%d", reply.ID)))
	return nil
}

func runTicketsShow(cmd *cobra.Command, args []string) error {
	client, err := newAuthenticatedClient()
	if err != nil {
		return err
	}
	ticket, err := client.GetTicket(args[0])
	if err != nil {
		return fmt.Errorf("failed to get ticket: %w", err)
	}
	fmt.Printf("%s — %s\nStatus: %s\nPriority: %s\nDepartment: %s\n\n%s\n", displayTicketID(*ticket), ticket.Subject, ticket.Status, ticket.Priority, ticket.Department.Name, ticket.Content)
	for _, r := range ticket.Replies {
		label := "reply"
		if r.InternalNote {
			label = "internal note"
		}
		fmt.Printf("\n[%s #%d] %s\n", label, r.ID, r.CreatedAt)
		fmt.Println(r.Content)
	}
	return nil
}

func displayTicketID(ticket api.Ticket) string {
	if ticket.TicketID != "" {
		return ticket.TicketID
	}
	return fmt.Sprintf("%d", ticket.ID)
}
