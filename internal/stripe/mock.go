package stripe

import (
	"bytes"
	"encoding/json"

	"github.com/stripe/stripe-go/v72"
)

func NewMock() *Mock {
	return &Mock{
		checkoutSessions: make([]*stripe.CheckoutSessionParams, 0),
	}
}

type Mock struct {
	checkoutSessions      []*stripe.CheckoutSessionParams
	billingPortalSessions []*stripe.BillingPortalSessionParams
}

func (m *Mock) CheckoutSession(params *stripe.CheckoutSessionParams) (string, error) {
	m.checkoutSessions = append(m.checkoutSessions, params)
	return "", nil
}

func (m *Mock) PopCheckoutSession() *stripe.CheckoutSessionParams {
	session := m.checkoutSessions[0]
	m.checkoutSessions = m.checkoutSessions[1:]
	return session
}

func (m *Mock) BillingPortalSession(params *stripe.BillingPortalSessionParams) (string, error) {
	m.billingPortalSessions = append(m.billingPortalSessions, params)
	return "", nil
}

func (m *Mock) PopBillingPortalSession() *stripe.BillingPortalSessionParams {
	session := m.billingPortalSessions[0]
	m.billingPortalSessions = m.billingPortalSessions[1:]
	return session
}

func (m *Mock) ConstructEvent(b []byte, signature string) (stripe.Event, error) {
	var event stripe.Event
	if err := json.NewDecoder(bytes.NewReader(b)).Decode(&event); err != nil {
		return event, err
	}
	return event, nil
}
