// Package email is the backend-side SES integration. Senders for guardian
// magic links (A0.4), DSR magic links (A1.1), and marketing campaigns plug
// in by composing a Sender; the infrastructure side (verified domain, DKIM,
// MAIL FROM, configuration set) lives in infra/email.
//
// This slice (A1.5) ships the interface and an SES implementation so future
// feature PRs don't have to redo the SDK wiring. No handler depends on this
// package yet.
package email

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	sestypes "github.com/aws/aws-sdk-go-v2/service/sesv2/types"
)

// Message is the minimal payload Sender accepts. Plain text only for now;
// HTML can be added when a feature actually needs it (guardian / DSR magic
// links are fine as text, marketing isn't built yet).
type Message struct {
	To         string
	Subject    string
	TextBody   string
	MessageTag string // optional CloudWatch dimension; defaults to "the-run"
}

// Sender abstracts "deliver this message somewhere". Production wires this
// to SES; unit tests pass NoOpSender or InMemorySender (latter to be added
// when handlers actually depend on this package).
type Sender interface {
	Send(ctx context.Context, msg Message) error
}

// Config controls Sender construction. Resolved from env vars by LoadConfig;
// production sets these from the Lambda env wired in infra/backend.
type Config struct {
	SenderAddress    string // SES_SENDER_ADDRESS, e.g. "noreply@running.rydback.net"
	ConfigurationSet string // SES_CONFIGURATION_SET, e.g. "the-run-events"
}

// LoadConfig reads the env vars set by Pulumi. Returns Config{} (zero) when
// nothing is configured — caller is expected to then use NoOpSender for
// dev / unit tests.
func LoadConfig() Config {
	return Config{
		SenderAddress:    os.Getenv("SES_SENDER_ADDRESS"),
		ConfigurationSet: os.Getenv("SES_CONFIGURATION_SET"),
	}
}

// NoOpSender records the call and reports success without contacting AWS.
// Suitable for unit tests and for local dev where SES isn't configured.
type NoOpSender struct{}

func (NoOpSender) Send(_ context.Context, msg Message) error {
	// Log the full body in NoOp mode so local devs can copy-paste any magic
	// links (guardian consent, future DSR) from the dev-backend stdout.
	// Production never hits this path — SES is wired via NewSenderFromEnv.
	log.Printf("email: NoOp send to=%s subject=%q\n--- body ---\n%s\n--- /body ---", msg.To, msg.Subject, msg.TextBody)
	return nil
}

// NewSenderFromEnv returns an SES-backed sender when SES env is configured,
// otherwise NoOpSender. The boolean reports which one was returned so the
// caller can log it once at startup.
func NewSenderFromEnv(ctx context.Context) (Sender, bool, error) {
	cfg := LoadConfig()
	if cfg.SenderAddress == "" || cfg.ConfigurationSet == "" {
		return NoOpSender{}, false, nil
	}
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("load aws config: %w", err)
	}
	return &SESSender{
		client: sesv2.NewFromConfig(awsCfg),
		cfg:    cfg,
	}, true, nil
}

// SESSender sends through Amazon SES v2.
type SESSender struct {
	client *sesv2.Client
	cfg    Config
}

func (s *SESSender) Send(ctx context.Context, msg Message) error {
	if msg.To == "" {
		return errors.New("email: To is required")
	}
	if msg.Subject == "" {
		return errors.New("email: Subject is required")
	}
	if msg.TextBody == "" {
		return errors.New("email: TextBody is required")
	}
	tag := msg.MessageTag
	if tag == "" {
		tag = "the-run"
	}
	_, err := s.client.SendEmail(ctx, &sesv2.SendEmailInput{
		FromEmailAddress:     aws.String(s.cfg.SenderAddress),
		ConfigurationSetName: aws.String(s.cfg.ConfigurationSet),
		Destination: &sestypes.Destination{
			ToAddresses: []string{msg.To},
		},
		Content: &sestypes.EmailContent{
			Simple: &sestypes.Message{
				Subject: &sestypes.Content{
					Data:    aws.String(msg.Subject),
					Charset: aws.String("UTF-8"),
				},
				Body: &sestypes.Body{
					Text: &sestypes.Content{
						Data:    aws.String(msg.TextBody),
						Charset: aws.String("UTF-8"),
					},
				},
			},
		},
		// MessageTag drives the CloudWatch dimension declared on the
		// configuration set's event destination — see infra/email.
		EmailTags: []sestypes.MessageTag{
			{Name: aws.String("MessageTag"), Value: aws.String(tag)},
		},
	})
	if err != nil {
		return fmt.Errorf("ses send: %w", err)
	}
	return nil
}
