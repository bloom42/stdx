package email

import "context"

// Mailer is the interface defining a service which sends emails
type Mailer interface {
	SendTransactionnal(ctx context.Context, email Email) error
	SendBroadcast(ctx context.Context, email Email) error
}
