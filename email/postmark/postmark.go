package postmark

import (
	"context"
	"net/http"
	"strings"

	"github.com/bloom42/stdx/email"
	"github.com/bloom42/stdx/postmark"
)

// Mailer implements the `email.Mailer` interface to send emails using postmarkapp.com's API
// https://postmarkapp.com/developer/api/overview
type Mailer struct {
	accountApiToken             string
	serverApiToken              string
	postmarkClient              *postmark.Client
	messageStreamBroadcast      string
	messageStreamTransactionnal string
}

type Config struct {
	AccountApiToken string
	ServerApiToken  string
	HttpClient      *http.Client
}

// NewMailer returns a new smtp Mailer
func NewMailer(config Config) *Mailer {
	postmarkClient := postmark.NewClient(config.AccountApiToken, config.HttpClient)

	return &Mailer{
		accountApiToken:             config.AccountApiToken,
		serverApiToken:              config.ServerApiToken,
		postmarkClient:              postmarkClient,
		messageStreamBroadcast:      "broadcast",
		messageStreamTransactionnal: "outbound",
	}
}

func (mailer *Mailer) SendTransactionnal(ctx context.Context, email email.Email) error {
	replyTo := ""

	if len(email.ReplyTo) == 1 {
		replyTo = email.ReplyTo[0].String()
	}

	postmarkEmail := postmark.Email{
		From:          email.From.String(),
		To:            email.To[0].String(),
		ReplyTo:       replyTo,
		Subject:       email.Subject,
		HtmlBody:      string(email.HTML),
		TextBody:      string(email.Text),
		MessageStream: mailer.messageStreamTransactionnal,
	}

	_, err := mailer.postmarkClient.SendEmail(ctx, mailer.serverApiToken, postmarkEmail)

	return err
}

func (mailer *Mailer) SendBroadcast(ctx context.Context, email email.Email) error {
	replyTo := ""

	if len(email.ReplyTo) == 1 {
		replyTo = email.ReplyTo[0].String()
	}

	headers := make([]postmark.Header, 0, len(email.Headers))
	for key, values := range email.Headers {
		headers = append(headers, postmark.Header{
			Name:  key,
			Value: strings.Join(values, ","),
		})
	}

	postmarkEmail := postmark.Email{
		From:          email.From.String(),
		To:            email.To[0].String(),
		ReplyTo:       replyTo,
		Subject:       email.Subject,
		HtmlBody:      string(email.HTML),
		TextBody:      string(email.Text),
		MessageStream: mailer.messageStreamBroadcast,
		Headers:       headers,
	}

	_, err := mailer.postmarkClient.SendEmail(ctx, mailer.serverApiToken, postmarkEmail)

	return err
}
