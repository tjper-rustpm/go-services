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

	Status SubscriptionStatus

	Invoices []Invoice
}

type SubscriptionStatus string

const (
	SubscriptionStatusIncomplete        SubscriptionStatus = "incomplete"
	SubscriptionStatusIncompleteExpired SubscriptionStatus = "incomplete_expired"
	SubscriptionStatusTrialing          SubscriptionStatus = "trialing"
	SubscriptionStatusActive            SubscriptionStatus = "active"
	SubscriptionStatusPastDue           SubscriptionStatus = "past_due"
	SubscriptionStatusCanceled          SubscriptionStatus = "canceled"
	SubscriptionStatusUnpaid            SubscriptionStatus = "unpaid"
)

type Invoice struct {
	model.Model
	SubscriptionID uuid.UUID

	Status InvoiceStatus

	PaymentIntents []PaymentIntent
}

type InvoiceStatus string

const (
	InvoiceStatusDraft         InvoiceStatus = "draft"
	InvoiceStatusOpen          InvoiceStatus = "open"
	InvoiceStatusPaid          InvoiceStatus = "paid"
	InvoiceStatusUncollectible InvoiceStatus = "uncollectible"
	InvoiceStatusVoid          InvoiceStatus = "void"
)

type PaymentIntent struct {
	model.Model
	InvoiceID uuid.UUID

	Status PaymentIntentStatus
}

type PaymentIntentStatus string

const (
	PaymentIntentStatusRequiresPaymentMethod PaymentIntentStatus = "requires_payment_method"
	PaymentIntentStatusRequiresConfirmation  PaymentIntentStatus = "requires_confirmation"
	PaymentIntentStatusRequiresAction        PaymentIntentStatus = "requires_action"
	PaymentIntentStatusProcessing            PaymentIntentStatus = "processing"
	PaymentIntentStatusRequiresCapture       PaymentIntentStatus = "requires_capture"
	PaymentIntentStatusCanceled              PaymentIntentStatus = "canceled"
	PaymentIntentStatusSucceeded             PaymentIntentStatus = "succeeded"
)
