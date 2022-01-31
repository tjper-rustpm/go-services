package model

import (
	"github.com/tjper/rustcron/internal/model"

	"github.com/google/uuid"
)

type Subscription struct {
	model.Model
	ServerID uuid.UUID
	UserID   uuid.UUID

	StripeCheckoutID     string
	StripeCustomerID     string
	StripeSubscriptionID string `gorm:"uniqueIndex"`

	Invoices []Invoice `gorm:"foreignKey:StripeSubscriptionID;references:StripeSubscriptionID"`
}

type Invoice struct {
	model.Model
	StripeSubscriptionID string

	Status InvoiceStatus
}

type InvoiceStatus string

const (
	InvoiceStatusPaid          InvoiceStatus = "paid"
	InvoiceStatusPaymentFailed InvoiceStatus = "payment_failed"
)
