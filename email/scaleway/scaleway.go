package scaleway

import (
	"context"
	"fmt"
	"net/http"

	"github.com/bloom42/stdx/email"
	scwemail "github.com/scaleway/scaleway-sdk-go/api/tem/v1alpha1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

// Mailer implements the `email.Mailer` interface to send emails using scaleway's API
// https://developers.scaleway.com/en/products/transactional_email/api
// https://www.scaleway.com/en/docs/managed-services/transactional-email
type Mailer struct {
	scalewayClient *scwemail.API
}

type Config struct {
	AccessKeyID string
	SecretKey   string
	Region      string
	ProjectID   string
	HttpClient  *http.Client
}

// NewMailer returns a new smtp Mailer
func NewMailer(config Config) (mailer *Mailer, err error) {
	scwClientOptions := []scw.ClientOption{
		scw.WithAuth(config.AccessKeyID, config.SecretKey),
		scw.WithDefaultRegion(scw.Region(config.Region)),
		scw.WithDefaultProjectID(config.ProjectID),
	}
	if config.HttpClient != nil {
		scwClientOptions = append(scwClientOptions, scw.WithHTTPClient(*&config.HttpClient))
	}

	scwClient, err := scw.NewClient(scwClientOptions...)
	if err != nil {
		err = fmt.Errorf("scaleway.NewMailer: creating client: %w", err)
		return
	}

	scalewayApi := scwemail.NewAPI(scwClient)

	mailer = &Mailer{
		scalewayClient: scalewayApi,
	}

	return
}

func (mailer *Mailer) SendTransactionnal(ctx context.Context, email email.Email) error {
	// replyTo := ""
	// if len(email.ReplyTo) == 1 {
	// 	replyTo = email.ReplyTo[0].String()
	// }
	input := &scwemail.CreateEmailRequest{
		// Region:      "",
		From: &scwemail.CreateEmailRequestAddress{Email: email.From.Address, Name: &email.From.Name},
		To: []*scwemail.CreateEmailRequestAddress{
			{Email: email.To[0].Address, Name: &email.To[0].Name},
		},
		// Cc:          []*scwemail.CreateEmailRequestAddress{},
		// Bcc:         []*scwemail.CreateEmailRequestAddress{},
		Subject: email.Subject,
		Text:    string(email.Text),
		HTML:    string(email.HTML),
		// ProjectID:   "",
		// Attachments: []*scwemail.CreateEmailRequestAttachment{},
		// SendBefore:  &time.Time{},
	}

	_, err := mailer.scalewayClient.CreateEmail(input)

	return err
}

func (mailer *Mailer) SendBroadcast(ctx context.Context, email email.Email) error {
	return mailer.SendTransactionnal(ctx, email)
}
