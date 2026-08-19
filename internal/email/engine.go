package email

import (
	"context"
	"fmt"
	"log/slog"
	"net/smtp"
	"regexp"
	"strings"
	"time"

	"helpdesk/internal/config"
	"helpdesk/internal/db"
	"helpdesk/internal/types"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/microcosm-cc/bluemonday"
)

var (
	refTokenRegex = regexp.MustCompile(`\[TCK-(\d+)\]`)
	quoteRegexes  = []*regexp.Regexp{
		regexp.MustCompile(`(?s)(On\s+.+?wrote:.*)`),
		regexp.MustCompile(`(?s)(-----Original Message-----.*)`),
		regexp.MustCompile(`(?s)(_{5,}.*)`),
		regexp.MustCompile(`(?s)(From:\s+.+?Sent:\s+.+?To:\s+.*)`),
	}
)

type InboundParsedEmail struct {
	MessageID   string
	InReplyTo   string
	References  []string
	FromAddress string
	FromName    string
	ToAddresses []string
	Subject     string
	BodyHTML    string
	BodyText    string
	Attachments []EmailAttachmentData
}

type EmailAttachmentData struct {
	Filename  string
	MimeType  string
	Bytes     []byte
	IsInline  bool
	ContentID string
}

type Engine struct {
	cfg       *config.Config
	queries   db.Querier
	sanitizer *bluemonday.Policy
}

func NewEngine(cfg *config.Config, queries db.Querier) *Engine {
	p := bluemonday.UGCPolicy()
	p.AllowStyles()
	p.AllowDataURIImages()

	return &Engine{
		cfg:       cfg,
		queries:   queries,
		sanitizer: p,
	}
}

// StripQuotedReply removes previous quoted conversation thread from incoming email body
func StripQuotedReply(body string) string {
	cleaned := body
	for _, re := range quoteRegexes {
		cleaned = re.ReplaceAllString(cleaned, "")
	}
	return strings.TrimSpace(cleaned)
}

// SanitizeHTML sanitizes HTML content against XSS using safe allowlist
func (e *Engine) SanitizeHTML(rawHTML string) string {
	return e.sanitizer.Sanitize(rawHTML)
}

// ExtractReferenceToken checks for [TCK-xxxxx] in the subject
func ExtractReferenceToken(subject string) string {
	matches := refTokenRegex.FindStringSubmatch(subject)
	if len(matches) > 1 {
		return "TCK-" + matches[1]
	}
	return ""
}

// IngestInboundEmail processes an incoming email through the exact thread resolution order
func (e *Engine) IngestInboundEmail(
	ctx context.Context,
	mailAccount db.MailAccounts,
	email InboundParsedEmail,
) (*db.CreateTicketRow, *db.TicketEvents, error) {
	if email.MessageID != "" {
		if existing, err := e.queries.GetEventByEmailMessageID(ctx, pgtype.Text{String: email.MessageID, Valid: true}); err == nil && existing.ID.Valid {
			slog.Info("Duplicate inbound email detected, skipping", "message_id", email.MessageID)
			return nil, nil, nil
		}
	}

	bodyText := email.BodyText
	bodyHTML := email.BodyHTML
	if mailAccount.StripQuotedReply {
		bodyText = StripQuotedReply(bodyText)
		bodyHTML = e.SanitizeHTML(bodyHTML)
	} else {
		bodyHTML = e.SanitizeHTML(bodyHTML)
	}

	var matchedTicketID pgtype.UUID

	if email.InReplyTo != "" {
		if ev, err := e.queries.GetEventByEmailMessageID(ctx, pgtype.Text{String: email.InReplyTo, Valid: true}); err == nil && ev.TicketID.Valid {
			matchedTicketID = ev.TicketID
		}
	}

	if !matchedTicketID.Valid && len(email.References) > 0 {
		for _, ref := range email.References {
			if ev, err := e.queries.GetEventByEmailMessageID(ctx, pgtype.Text{String: ref, Valid: true}); err == nil && ev.TicketID.Valid {
				matchedTicketID = ev.TicketID
				break
			}
		}
	}

	if !matchedTicketID.Valid {
		token := ExtractReferenceToken(email.Subject)
		if token != "" {
			if t, err := e.queries.GetTicketByReferenceNo(ctx, token); err == nil && t.ID.Valid {
				matchedTicketID = t.ID
			}
		}
	}

	contact, err := e.queries.GetContactByEmail(ctx, email.FromAddress)
	if err != nil || !contact.ID.Valid {
		parts := strings.Split(email.FromAddress, "@")
		var orgID pgtype.UUID
		if len(parts) == 2 {
			domain := strings.ToLower(parts[1])
			if org, err := e.queries.GetOrCreateOrganizationByDomain(ctx, db.GetOrCreateOrganizationByDomainParams{
				ID:     types.NewUUIDv7(),
				Name:   domain,
				Slug:   domain,
				Domain: pgtype.Text{String: domain, Valid: true},
			}); err == nil {
				orgID = org.ID
			}
		}

		name := email.FromName
		if name == "" {
			name = email.FromAddress
		}

		contact, err = e.queries.CreateContact(ctx, db.CreateContactParams{
			ID:             types.NewUUIDv7(),
			OrganizationID: orgID,
			PrimaryEmail:   email.FromAddress,
			FullName:       name,
			Locale:         "en",
			Timezone:       "UTC",
			IsVerified:     true,
			CustomData:     []byte("{}"),
		})
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create contact from inbound email: %w", err)
		}
	}

	now := time.Now().UTC()
	nowTz := types.TimeToTimestamptz(now)

	if matchedTicketID.Valid {
		ticketDetail, err := e.queries.GetTicketDetail(ctx, matchedTicketID)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to fetch matched ticket: %w", err)
		}

		event, err := e.queries.CreateTicketEvent(ctx, db.CreateTicketEventParams{
			ID:              types.NewUUIDv7(),
			TicketID:        matchedTicketID,
			Kind:            "inbound_email",
			Visibility:      "public",
			AuthorType:      "contact",
			AuthorID:        contact.ID,
			BodyHtml:        bodyHTML,
			BodyText:        bodyText,
			Metadata:        []byte("{}"),
			EmailMessageID:  pgtype.Text{String: email.MessageID, Valid: email.MessageID != ""},
			EmailInReplyTo:  pgtype.Text{String: email.InReplyTo, Valid: email.InReplyTo != ""},
			EmailReferences: email.References,
			DeliveryStatus:  "delivered",
			OccurredAt:      nowTz,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create inbound event: %w", err)
		}

		var reopenCount int32 = ticketDetail.ReopenCount
		var newStatusID pgtype.UUID = ticketDetail.StatusID
		var newStatusCategory string = ticketDetail.StatusCategory

		if ticketDetail.StatusCategory == "resolved" || ticketDetail.StatusCategory == "closed" {
			reopenCount++
			newStatusCategory = "open"
		}

		_, _ = e.queries.UpdateTicketFields(ctx, db.UpdateTicketFieldsParams{
			ID:                     matchedTicketID,
			StatusID:               newStatusID,
			StatusCategory:         pgtype.Text{String: newStatusCategory, Valid: true},
			ReopenCount:            pgtype.Int4{Int32: reopenCount, Valid: true},
			LastCustomerActivityAt: nowTz,
		})

		return nil, &event, nil
	}

	newTicketID := types.NewUUIDv7()

	createdTicket, err := e.queries.CreateTicket(ctx, db.CreateTicketParams{
		ID:              newTicketID,
		Subject:         email.Subject,
		DescriptionHtml: bodyHTML,
		DescriptionText: bodyText,
		StatusID:        pgtype.UUID{Bytes: [16]byte{0x01, 0x8e, 0, 0, 0, 0, 0x70, 0, 0x80, 0, 0, 0, 0, 0, 0x01, 0x01}, Valid: true},
		StatusCategory:  "open",
		PriorityID:      pgtype.UUID{Bytes: [16]byte{0x01, 0x8e, 0, 0, 0, 0, 0x70, 0, 0x80, 0, 0, 0, 0, 0, 0x02, 0x02}, Valid: true},
		PriorityWeight:  2,
		TypeID:          mailAccount.DefaultTypeID,
		ContactID:       contact.ID,
		OrganizationID:  contact.OrganizationID,
		AssignedTeamID:  mailAccount.DefaultTeamID,
		Source:          "email",
		CustomData:      []byte("{}"),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create ticket from email: %w", err)
	}

	initialEvent, err := e.queries.CreateTicketEvent(ctx, db.CreateTicketEventParams{
		ID:              types.NewUUIDv7(),
		TicketID:        createdTicket.ID,
		Kind:            "inbound_email",
		Visibility:      "public",
		AuthorType:      "contact",
		AuthorID:        contact.ID,
		BodyHtml:        bodyHTML,
		BodyText:        bodyText,
		Metadata:        []byte("{}"),
		EmailMessageID:  pgtype.Text{String: email.MessageID, Valid: email.MessageID != ""},
		EmailInReplyTo:  pgtype.Text{String: email.InReplyTo, Valid: email.InReplyTo != ""},
		EmailReferences: email.References,
		DeliveryStatus:  "delivered",
		OccurredAt:      nowTz,
	})
	if err != nil {
		return &createdTicket, nil, fmt.Errorf("failed to create initial ticket event: %w", err)
	}

	return &createdTicket, &initialEvent, nil
}

// SendOutboundEmail sends an outbound email via SMTP with proper threading headers and suppression checks
func (e *Engine) SendOutboundEmail(
	ctx context.Context,
	toEmail string,
	referenceNo string,
	subject string,
	bodyHTML string,
	inReplyToMessageID string,
	references []string,
	agentSignature string,
) error {
	isSuppressed, err := e.queries.IsEmailSuppressed(ctx, toEmail)
	if err == nil && isSuppressed {
		slog.Warn("Recipient email is suppressed, omitting outbound send", "to", toEmail)
		return nil
	}

	msgID := fmt.Sprintf("<%s@%s>", types.UUIDToString(types.NewUUIDv7()), "helpdesk.local")
	formattedSubject := fmt.Sprintf("[%s] %s", referenceNo, subject)

	fullHTML := bodyHTML
	if agentSignature != "" {
		fullHTML = fmt.Sprintf("%s<br/><br/>%s", bodyHTML, agentSignature)
	}

	var headers strings.Builder
	headers.WriteString(fmt.Sprintf("From: %s\r\n", e.cfg.SMTPFrom))
	headers.WriteString(fmt.Sprintf("To: %s\r\n", toEmail))
	headers.WriteString(fmt.Sprintf("Subject: %s\r\n", formattedSubject))
	headers.WriteString(fmt.Sprintf("Message-ID: %s\r\n", msgID))
	if inReplyToMessageID != "" {
		headers.WriteString(fmt.Sprintf("In-Reply-To: %s\r\n", inReplyToMessageID))
	}
	if len(references) > 0 {
		headers.WriteString(fmt.Sprintf("References: %s\r\n", strings.Join(references, " ")))
	}
	headers.WriteString("MIME-Version: 1.0\r\n")
	headers.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	headers.WriteString("\r\n")
	headers.WriteString(fullHTML)

	addr := fmt.Sprintf("%s:%d", e.cfg.SMTPHost, e.cfg.SMTPPort)
	var auth smtp.Auth
	if e.cfg.SMTPUser != "" && e.cfg.SMTPPass != "" {
		auth = smtp.PlainAuth("", e.cfg.SMTPUser, e.cfg.SMTPPass, e.cfg.SMTPHost)
	}

	err = smtp.SendMail(addr, auth, e.cfg.SMTPFrom, []string{toEmail}, []byte(headers.String()))
	if err != nil {
		return fmt.Errorf("SMTP delivery failed to %s: %w", toEmail, err)
	}

	slog.Info("Outbound email sent successfully", "to", toEmail, "subject", formattedSubject)
	return nil
}
