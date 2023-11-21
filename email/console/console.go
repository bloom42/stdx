package console

import (
	"context"
	"fmt"

	"github.com/bloom42/stdx/email"
)

// Mailer implements the `email.Mailer` interface to print emails to console
type Mailer struct {
}

// NewMailer returns a new console Mailer
func NewMailer() *Mailer {
	return &Mailer{}
}

// Send an email using the console mailer
func (mailer *Mailer) SendTransactionnal(ctx context.Context, email email.Email) error {
	data, err := email.Bytes()
	if err != nil {
		return err
	}
	fmt.Println(string(data))

	return nil
}

func (mailer *Mailer) SendBroadcast(ctx context.Context, email email.Email) error {
	data, err := email.Bytes()
	if err != nil {
		return err
	}
	fmt.Println(string(data))

	return nil
}
