package stripe

import (
	"bytes"
	"encoding/json"
	"sync"

	"github.com/stripe/stripe-go/v72"
)

func NewMock() *Mock {
	return &Mock{
		mutex:                 new(sync.Mutex),
		checkoutSessions:      make([]*stripe.CheckoutSessionParams, 0),
		billingPortalSessions: make([]*stripe.BillingPortalSessionParams, 0),
	}
}

type Mock struct {
	mutex *sync.Mutex

	checkoutSessions      []*stripe.CheckoutSessionParams
	billingPortalSessions []*stripe.BillingPortalSessionParams
}

func (m *Mock) CheckoutSession(params *stripe.CheckoutSessionParams) (string, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.checkoutSessions = append(m.checkoutSessions, params)
	return "", nil
}

func (m *Mock) PopCheckoutSession() *stripe.CheckoutSessionParams {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	session := m.checkoutSessions[0]
	m.checkoutSessions = m.checkoutSessions[1:]
	return session
}

func (m *Mock) BillingPortalSession(params *stripe.BillingPortalSessionParams) (string, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.billingPortalSessions = append(m.billingPortalSessions, params)
	return "", nil
}

func (m *Mock) PopBillingPortalSession() *stripe.BillingPortalSessionParams {
	m.mutex.Lock()
	defer m.mutex.Unlock()

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
