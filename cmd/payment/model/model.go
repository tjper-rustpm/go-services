package model

import (
	"github.com/tjper/rustcron/internal/model"

	"github.com/google/uuid"
)

type Subscription struct {
	model.Model
	ServerID uuid.UUID
	UserID   uuid.UUID

	StripeCustomerID     string
	StripeSubscriptionID string
	StripePriceID        string

	Invoices []Invoice
}

type Invoice struct {
	model.Model
	SubscriptionID uuid.UUID

	Status InvoiceStatus
}

type InvoiceStatus string

const (
	InvoiceStatusPaid          InvoiceStatus = "paid"
	InvoiceStatusPaymentFailed InvoiceStatus = "payment_failed"
)
