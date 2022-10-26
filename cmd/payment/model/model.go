package model

import (
	"time"

	"github.com/tjper/rustcron/internal/model"

	"github.com/google/uuid"
)

type Subscription struct {
	model.Model

	StripeCheckoutID     string
	StripeSubscriptionID string
	StripeEventID        string

	ServerID uuid.UUID
	Server   Server

	CustomerID uuid.UUID
	Customer   Customer `gorm:"foreignKey:UserID;references:CustomerID"`

	Invoices []Invoice
}

// Status retrieves the status of the subscription.
func (sub Subscription) Status() InvoiceStatus {
	if len(sub.Invoices) == 0 {
		return InvoiceStatusUnknown
	}

	latest := sub.Invoices[0]
	for _, invoice := range sub.Invoices {
		if invoice.CreatedAt.After(latest.CreatedAt) {
			latest = invoice
		}
	}

	duration := time.Hour * 24 * 30 // 30 days
	if latest.CreatedAt.Before(time.Now().Add(-duration)) {
		return InvoiceStatusInactive
	}

	return latest.Status
}

// Subscription many Subscription entities.
type Subscriptions []Subscription

// Invoice is the record of a payment transaction.
type Invoice struct {
	model.Model

	SubscriptionID uuid.UUID
	StripeEventID  string

	Status InvoiceStatus
}

type InvoiceStatus string

const (
	InvoiceStatusUnknown       InvoiceStatus = "unknown"
	InvoiceStatusPaid          InvoiceStatus = "paid"
	InvoiceStatusPaymentFailed InvoiceStatus = "payment_failed"
	InvoiceStatusInactive      InvoiceStatus = "inactive"
)

type Server struct {
	ID                  uuid.UUID      `json:"id"`
	ActiveSubscriptions uint16         `gorm:"->" json:"activeSubscriptions"`
	SubscriptionLimit   uint16         `json:"subscriptionLimit"`
	Subscriptions       []Subscription `json:"-"`

	model.At
}

type Servers []Server

type Customer struct {
	UserID           uuid.UUID
	StripeCustomerID string
	SteamID          string
	Subscriptions    []Subscription `gorm:"foreignKey:CustomerID;references:UserID"`

	model.At
}
