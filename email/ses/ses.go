package ses

import (
	"context"
	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ses"
	"github.com/bloom42/stdx/email"
)

// Mailer implements the `email.Mailer` interface to send emails using SMTP
type Mailer struct {
	sesClient *ses.SES
}

type Config struct {
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	HttpClient      *http.Client
}

// NewMailer returns a new smtp Mailer
func NewMailer(config Config) (*Mailer, error) {
	awsSession, err := session.NewSession(&aws.Config{
		Region:      aws.String(config.Region),
		Credentials: credentials.NewStaticCredentials(config.AccessKeyID, config.SecretAccessKey, ""),
		HTTPClient:  config.HttpClient,
	})
	if err != nil {
		return nil, err
	}

	// Create SES service client
	sesClient := ses.New(awsSession)

	return &Mailer{
		sesClient,
	}, nil
}

// Send an email using the SES mailer
func (mailer *Mailer) SendTransactionnal(ctx context.Context, email email.Email) error {
	rawEmail, err := email.Bytes()
	if err != nil {
		return err
	}

	_, err = mailer.sesClient.SendRawEmail(&ses.SendRawEmailInput{
		RawMessage: &ses.RawMessage{
			Data: rawEmail,
		},
	})

	return err
}

// TODO
func (mailer *Mailer) SendBroadcast(ctx context.Context, email email.Email) error {
	rawEmail, err := email.Bytes()
	if err != nil {
		return err
	}

	_, err = mailer.sesClient.SendRawEmail(&ses.SendRawEmailInput{
		RawMessage: &ses.RawMessage{
			Data: rawEmail,
		},
	})

	return err
}
