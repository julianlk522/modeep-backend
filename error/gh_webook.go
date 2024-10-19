package error

import (
	"errors"
)

var (
	ErrNoWebhookSecret         error = errors.New("webhook secret environment variable not found")
	ErrNoWebhookSignature      error = errors.New("gh webhook signature not found in headers")
	ErrInvalidWebhookSignature error = errors.New("invalid webhook signature: does not match expected")
)
