package channels

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/smtp"
	"net/textproto"
	"strings"
	"time"

	"github.com/alexandrmotologa/notiflux/internal/config"
	"github.com/alexandrmotologa/notiflux/internal/engine"
	"github.com/alexandrmotologa/notiflux/internal/store"
)

// EmailDispatcher handles sending transactional emails via SMTP or Resend API.
type EmailDispatcher struct {
	cfg        *config.Config
	httpClient *http.Client
}

func NewEmailDispatcher(cfg *config.Config) *EmailDispatcher {
	return &EmailDispatcher{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (d *EmailDispatcher) Channel() string {
	return store.ChannelEmail
}

func (d *EmailDispatcher) Dispatch(ctx context.Context, msg *engine.CompiledMessage) error {
	if msg.Subscriber == nil || msg.Subscriber.Email == "" {
		return fmt.Errorf("subscriber email address is required")
	}

	// 1. Try Resend if configured
	if d.cfg.ResendAPIKey != "" {
		return d.sendViaResend(ctx, msg)
	}

	// 2. Try SMTP if configured
	if d.cfg.SMTPHost != "" {
		return d.sendViaSMTP(ctx, msg)
	}

	return fmt.Errorf("no email provider configured: set SMTP or Resend API credentials")
}

func (d *EmailDispatcher) sendViaResend(ctx context.Context, msg *engine.CompiledMessage) error {
	from := d.cfg.SMTPFrom
	if from == "" {
		from = "Notiflux <notifications@notiflux.dev>"
	}

	payload := map[string]any{
		"from":    from,
		"to":      []string{msg.Subscriber.Email},
		"subject": msg.Subject,
	}

	if msg.HTMLBody != "" {
		payload["html"] = msg.HTMLBody
	}
	if msg.Body != "" {
		payload["text"] = msg.Body
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.resend.com/emails", bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+d.cfg.ResendAPIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("resend request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("resend returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func (d *EmailDispatcher) sendViaSMTP(ctx context.Context, msg *engine.CompiledMessage) error {
	from := d.cfg.SMTPFrom
	if from == "" {
		from = "no-reply@notiflux.dev"
	}
	to := msg.Subscriber.Email

	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("From: %s\r\n", from))
	buf.WriteString(fmt.Sprintf("To: %s\r\n", to))
	buf.WriteString(fmt.Sprintf("Subject: %s\r\n", msg.Subject))
	buf.WriteString("MIME-Version: 1.0\r\n")

	writer := multipart.NewWriter(&buf)
	buf.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=%s\r\n\r\n", writer.Boundary()))

	// Plain text part
	if msg.Body != "" {
		partHeader := make(textproto.MIMEHeader)
		partHeader.Set("Content-Type", "text/plain; charset=UTF-8")
		partHeader.Set("Content-Transfer-Encoding", "quoted-printable")
		part, err := writer.CreatePart(partHeader)
		if err == nil {
			part.Write([]byte(msg.Body))
		}
	}

	// HTML part
	if msg.HTMLBody != "" {
		partHeader := make(textproto.MIMEHeader)
		partHeader.Set("Content-Type", "text/html; charset=UTF-8")
		partHeader.Set("Content-Transfer-Encoding", "quoted-printable")
		part, err := writer.CreatePart(partHeader)
		if err == nil {
			part.Write([]byte(msg.HTMLBody))
		}
	}

	_ = writer.Close()

	addr := fmt.Sprintf("%s:%d", d.cfg.SMTPHost, d.cfg.SMTPPort)

	var auth smtp.Auth
	if d.cfg.SMTPUser != "" {
		auth = smtp.PlainAuth("", d.cfg.SMTPUser, d.cfg.SMTPPass, d.cfg.SMTPHost)
	}

	// Port 465 typically uses implicit TLS, ports 587 / 25 use STARTTLS
	if d.cfg.SMTPPort == 465 {
		tlsConfig := &tls.Config{
			ServerName: d.cfg.SMTPHost,
		}
		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return fmt.Errorf("tls connection failed: %w", err)
		}
		defer conn.Close()

		client, err := smtp.NewClient(conn, d.cfg.SMTPHost)
		if err != nil {
			return err
		}
		defer client.Quit()

		if auth != nil {
			if err := client.Auth(auth); err != nil {
				return fmt.Errorf("smtp auth failed: %w", err)
			}
		}

		if err := client.Mail(from); err != nil {
			return err
		}
		if err := client.Rcpt(to); err != nil {
			return err
		}

		w, err := client.Data()
		if err != nil {
			return err
		}
		_, err = w.Write(buf.Bytes())
		if err != nil {
			return err
		}
		return w.Close()
	}

	// Standard STARTTLS / Plain SMTP
	return smtp.SendMail(addr, auth, extractEmailAddress(from), []string{to}, buf.Bytes())
}

func extractEmailAddress(from string) string {
	if strings.Contains(from, "<") && strings.Contains(from, ">") {
		start := strings.Index(from, "<") + 1
		end := strings.Index(from, ">")
		if start < end {
			return from[start:end]
		}
	}
	return from
}
