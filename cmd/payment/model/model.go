package model

import (
	"time"

	"github.com/tjper/rustcron/internal/model"
	"github.com/tjper/rustcron/internal/stripe"

	"github.com/google/uuid"
)

type Vip struct {
	model.Model

	StripeCheckoutID string
	StripeEventID    string

	ServerID uuid.UUID
	Server   Server

	CustomerID uuid.UUID
	Customer   Customer

	ExpiresAt time.Time
}

// ComputeVipExpiration determines a VIP expiration based on a Stripe price.
func ComputeVipExpiration(price stripe.Price) time.Time {
	var expiresAt time.Time
	switch price {
	case stripe.MonthlyVipSubscription:
		// Expires in 30 days.
		expiresAt = time.Now().Add(30 * 24 * time.Hour).UTC()
	case stripe.WeeklyVipOneTime:
		// Expires in 5 days.
		expiresAt = time.Now().Add(5 * 24 * time.Hour).UTC()
	}
	return expiresAt
}

type Vips []Vip

type Subscription struct {
	model.Model

	StripeSubscriptionID string

	VipID uuid.UUID
	Vip   Vip

	Invoices []Invoice
}

// LatestInvoiceStatus retrieves the latest invoice status of the subscription.
func (sub Subscription) LatestInvoiceStatus() InvoiceStatus {
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
	ID                  uuid.UUID `json:"id"`
	ActiveSubscriptions uint16    `gorm:"->" json:"activeSubscriptions"`
	SubscriptionLimit   uint16    `json:"subscriptionLimit"`

	model.At
}

func (s Server) Clone() *Server {
	clone := s
	return &clone
}

func (s *Server) Scrub() {
	s.At.Scrub()
}

type Servers []Server

func (s Servers) Scrub() {
	for i := range s {
		s[i].Scrub()
	}
}

type Customer struct {
	model.Model
	SteamID          string
	StripeCustomerID string
}

type Customers []Customer

type User struct {
	ID         uuid.UUID `json:"id"`
	CustomerID uuid.UUID

	model.At
}

type Users []User
