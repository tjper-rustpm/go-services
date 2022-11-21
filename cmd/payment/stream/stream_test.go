package stream

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/tjper/rustcron/cmd/payment/db"
	"github.com/tjper/rustcron/cmd/payment/model"
	"github.com/tjper/rustcron/cmd/payment/staging"
	"github.com/tjper/rustcron/internal/event"
	igorm "github.com/tjper/rustcron/internal/gorm"
	imodel "github.com/tjper/rustcron/internal/model"
	"github.com/tjper/rustcron/internal/rand"
	"github.com/tjper/rustcron/internal/stream"
	istripe "github.com/tjper/rustcron/internal/stripe"
	"gorm.io/gorm"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/stripe/stripe-go/v72"
	"go.uber.org/zap"
)

func TestEventHandlerInputValidation(t *testing.T) {
	type expected struct {
		err error
	}
	tests := map[string]struct {
		event *event.StripeWebhookEvent
		exp   expected
	}{
		"event ID": {
			event: &event.StripeWebhookEvent{
				StripeEvent: stripe.Event{
					ID:   "",
					Type: "checkout.session.completed",
				},
			},
			exp: expected{
				err: fmt.Errorf("stripe event ID empty: %w", errNoRetry),
			},
		},
		"payment checkout ClientReferenceID": {
			event: &event.StripeWebhookEvent{
				StripeEvent: stripe.Event{
					ID:   uuid.NewString(),
					Type: "checkout.session.completed",
					Data: &stripe.EventData{
						Raw: json.RawMessage(`{
              "client_reference_id": "",
              "mode": "payment"
            }`),
					},
				},
			},
			exp: expected{
				err: fmt.Errorf("checkout ClientReferenceID empty: %w", errNoRetry),
			},
		},
		"payment checkout ID": {
			event: &event.StripeWebhookEvent{
				StripeEvent: stripe.Event{
					ID:   uuid.NewString(),
					Type: "checkout.session.completed",
					Data: &stripe.EventData{
						Raw: json.RawMessage(`{
              "id": "",
              "client_reference_id": "non-empty",
              "mode": "payment"
            }`),
					},
				},
			},
			exp: expected{
				err: fmt.Errorf("checkout ID empty: %w", errNoRetry),
			},
		},
		"payment checkout Customer": {
			event: &event.StripeWebhookEvent{
				StripeEvent: stripe.Event{
					ID:   uuid.NewString(),
					Type: "checkout.session.completed",
					Data: &stripe.EventData{
						Raw: json.RawMessage(`{
              "id": "non-empty",
              "client_reference_id": "non-empty",
              "mode": "payment"
            }`),
					},
				},
			},
			exp: expected{
				err: fmt.Errorf("checkout Customer nil: %w", errNoRetry),
			},
		},
		"payment checkout Customer ID": {
			event: &event.StripeWebhookEvent{
				StripeEvent: stripe.Event{
					ID:   uuid.NewString(),
					Type: "checkout.session.completed",
					Data: &stripe.EventData{
						Raw: json.RawMessage(`{
              "customer": {"id": ""},
              "id": "non-empty",
              "client_reference_id": "non-empty",
              "mode": "payment"
            }`),
					},
				},
			},
			exp: expected{
				err: fmt.Errorf("checkout Customer ID empty: %w", errNoRetry),
			},
		},
		"payment checkout PaymentStatus": {
			event: &event.StripeWebhookEvent{
				StripeEvent: stripe.Event{
					ID:   uuid.NewString(),
					Type: "checkout.session.completed",
					Data: &stripe.EventData{
						Raw: json.RawMessage(`{
              "payment_status": "",
              "customer": {"id": "non-empty"},
              "id": "non-empty",
              "client_reference_id": "non-empty",
              "mode": "payment"
            }`),
					},
				},
			},
			exp: expected{
				err: fmt.Errorf("checkout payment status is not \"paid\": %w", errNoRetry),
			},
		},
		"payment checkout LineItems nil": {
			event: &event.StripeWebhookEvent{
				StripeEvent: stripe.Event{
					ID:   uuid.NewString(),
					Type: "checkout.session.completed",
					Data: &stripe.EventData{
						Raw: json.RawMessage(`{
              "payment_status": "paid",
              "customer": {"id": "non-empty"},
              "id": "non-empty",
              "client_reference_id": "non-empty",
              "mode": "payment"
            }`),
					},
				},
			},
			exp: expected{
				err: fmt.Errorf("checkout LineItems nil: %w", errNoRetry),
			},
		},
		"payment checkout LineItems Data": {
			event: &event.StripeWebhookEvent{
				StripeEvent: stripe.Event{
					ID:   uuid.NewString(),
					Type: "checkout.session.completed",
					Data: &stripe.EventData{
						Raw: json.RawMessage(`{
              "line_items": {"data": []},
              "payment_status": "paid",
              "customer": {"id": "non-empty"},
              "id": "non-empty",
              "client_reference_id": "non-empty",
              "mode": "payment"
            }`),
					},
				},
			},
			exp: expected{
				err: fmt.Errorf("checkout not for a single item: %w", errNoRetry),
			},
		},
		"subscription checkout ClientReferenceID": {
			event: &event.StripeWebhookEvent{
				StripeEvent: stripe.Event{
					ID:   uuid.NewString(),
					Type: "checkout.session.completed",
					Data: &stripe.EventData{
						Raw: json.RawMessage(`{
              "client_reference_id": "",
              "mode": "subscription"
            }`),
					},
				},
			},
			exp: expected{
				err: fmt.Errorf("checkout ClientReferenceID empty: %w", errNoRetry),
			},
		},
		"subscription checkout ID": {
			event: &event.StripeWebhookEvent{
				StripeEvent: stripe.Event{
					ID:   uuid.NewString(),
					Type: "checkout.session.completed",
					Data: &stripe.EventData{
						Raw: json.RawMessage(`{
              "id": "",
              "client_reference_id": "non-empty",
              "mode": "subscription"
            }`),
					},
				},
			},
			exp: expected{
				err: fmt.Errorf("checkout ID empty: %w", errNoRetry),
			},
		},
		"subscription checkout Subscription": {
			event: &event.StripeWebhookEvent{
				StripeEvent: stripe.Event{
					ID:   uuid.NewString(),
					Type: "checkout.session.completed",
					Data: &stripe.EventData{
						Raw: json.RawMessage(`{
              "id": "non-empty",
              "client_reference_id": "non-empty",
              "mode": "subscription"
            }`),
					},
				},
			},
			exp: expected{
				err: fmt.Errorf("checkout Subscription nil: %w", errNoRetry),
			},
		},
		"subscription checkout Subscription ID": {
			event: &event.StripeWebhookEvent{
				StripeEvent: stripe.Event{
					ID:   uuid.NewString(),
					Type: "checkout.session.completed",
					Data: &stripe.EventData{
						Raw: json.RawMessage(`{
              "subscription": {"id": ""},
              "id": "non-empty",
              "client_reference_id": "non-empty",
              "mode": "subscription"
            }`),
					},
				},
			},
			exp: expected{
				err: fmt.Errorf("checkout Subscription ID empty: %w", errNoRetry),
			},
		},
		"subscription checkout Customer": {
			event: &event.StripeWebhookEvent{
				StripeEvent: stripe.Event{
					ID:   uuid.NewString(),
					Type: "checkout.session.completed",
					Data: &stripe.EventData{
						Raw: json.RawMessage(`{
              "subscription": {"id": "non-empty"},
              "id": "non-empty",
              "client_reference_id": "non-empty",
              "mode": "subscription"
            }`),
					},
				},
			},
			exp: expected{
				err: fmt.Errorf("checkout Customer nil: %w", errNoRetry),
			},
		},
		"subscription checkout Customer ID": {
			event: &event.StripeWebhookEvent{
				StripeEvent: stripe.Event{
					ID:   uuid.NewString(),
					Type: "checkout.session.completed",
					Data: &stripe.EventData{
						Raw: json.RawMessage(`{
              "customer": {"id": ""},
              "subscription": {"id": "non-empty"},
              "id": "non-empty",
              "client_reference_id": "non-empty",
              "mode": "subscription"
            }`),
					},
				},
			},
			exp: expected{
				err: fmt.Errorf("checkout Customer ID empty: %w", errNoRetry),
			},
		},
		"subscription checkout LineItems nil": {
			event: &event.StripeWebhookEvent{
				StripeEvent: stripe.Event{
					ID:   uuid.NewString(),
					Type: "checkout.session.completed",
					Data: &stripe.EventData{
						Raw: json.RawMessage(`{
              "customer": {"id": "non-empty"},
              "subscription": {"id": "non-empty"},
              "id": "non-empty",
              "client_reference_id": "non-empty",
              "mode": "subscription"
            }`),
					},
				},
			},
			exp: expected{
				err: fmt.Errorf("checkout LineItems nil: %w", errNoRetry),
			},
		},
		"subscription checkout LineItems Data": {
			event: &event.StripeWebhookEvent{
				StripeEvent: stripe.Event{
					ID:   uuid.NewString(),
					Type: "checkout.session.completed",
					Data: &stripe.EventData{
						Raw: json.RawMessage(`{
              "line_items": {"data": []},
              "customer": {"id": "non-empty"},
              "subscription": {"id": "non-empty"},
              "id": "non-empty",
              "client_reference_id": "non-empty",
              "mode": "subscription"
            }`),
					},
				},
			},
			exp: expected{
				err: fmt.Errorf("checkout not for a single item: %w", errNoRetry),
			},
		},
		"invoice Status": {
			event: &event.StripeWebhookEvent{
				StripeEvent: stripe.Event{
					ID:   uuid.NewString(),
					Type: "invoice.paid",
					Data: &stripe.EventData{
						Raw: json.RawMessage(`{
              "status": "",
              "id": "non-empty",
              "mode": "subscription"
            }`),
					},
				},
			},
			exp: expected{
				err: fmt.Errorf("invoice Status empty: %w", errNoRetry),
			},
		},
		"invoice Subscription": {
			event: &event.StripeWebhookEvent{
				StripeEvent: stripe.Event{
					ID:   uuid.NewString(),
					Type: "invoice.paid",
					Data: &stripe.EventData{
						Raw: json.RawMessage(`{
              "status": "paid",
              "id": "non-empty",
              "mode": "subscription"
            }`),
					},
				},
			},
			exp: expected{
				err: fmt.Errorf("invoice Subscription nil: %w", errNoRetry),
			},
		},
		"invoice Subscription ID": {
			event: &event.StripeWebhookEvent{
				StripeEvent: stripe.Event{
					ID:   uuid.NewString(),
					Type: "invoice.paid",
					Data: &stripe.EventData{
						Raw: json.RawMessage(`{
              "subscription": {"id": ""},
              "status": "paid",
              "id": "non-empty",
              "mode": "subscription"
            }`),
					},
				},
			},
			exp: expected{
				err: fmt.Errorf("invoice Subscription ID empty: %w", errNoRetry),
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			handler := NewHandler(
				zap.NewNop(),
				staging.NewClientMock(),
				db.NewStoreMock(),
				stream.NewClientMock(),
			)

			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			err := handler.handleStripeEvent(ctx, test.event)
			require.EqualError(t, err, test.exp.err.Error())
		})
	}
}

func TestHandlePaymentCheckoutSessionComplete(t *testing.T) {
	// shared encompasses state shared between stages of the test. These values
	// are often generated and are therefore unique for each test run. Therefore,
	// they are passed between stages of the test to check for equality.
	type shared struct {
		clientReferenceID string
		steamID           string
		serverID          uuid.UUID
		stripeEventID     string
		stripeCheckoutID  string
		stripeCustomerID  string
		expiresAt         time.Time
	}

	// expected encompasses expected values to test for.
	type expected struct {
		err error
	}

	tests := map[string]struct {
		event                        func(*testing.T, *shared) *event.StripeWebhookEvent
		stagingFetchCheckout         func(*testing.T, *shared) func(context.Context, string) (interface{}, error)
		storeFirstVipByStripeEventID func(*testing.T, *shared) func(context.Context, string) (*model.Vip, error)
		storeCreateVip               func(*testing.T, *shared) func(context.Context, *model.Vip, *model.Customer) error
		streamWrite                  func(*testing.T, *shared) func(context.Context, []byte) error
		exp                          expected
	}{
		"payment checkout session complete": {
			event: func(t *testing.T, shared *shared) *event.StripeWebhookEvent {
				shared.clientReferenceID = uuid.New().String()
				shared.stripeEventID = uuid.New().String()
				shared.stripeCheckoutID = uuid.New().String()
				shared.stripeCustomerID = uuid.New().String()

				checkout := map[string]interface{}{
					"id":                  shared.stripeCheckoutID,
					"mode":                "payment",
					"client_reference_id": shared.clientReferenceID,
					"customer": map[string]interface{}{
						"id": shared.stripeCustomerID,
					},
					"payment_status": "paid",
					"line_items": map[string]interface{}{
						"data": []map[string]interface{}{
							{"price": istripe.WeeklyVipOneTime},
						},
					},
				}
				raw, err := json.Marshal(&checkout)
				require.Nil(t, err)

				event := event.NewStripeWebhookEvent(
					stripe.Event{
						ID:      shared.stripeEventID,
						Type:    "checkout.session.completed",
						Created: time.Now().Unix(),
						Data: &stripe.EventData{
							Object: checkout,
							Raw:    raw,
						},
					},
				)
				return &event
			},
			stagingFetchCheckout: func(t *testing.T, shared *shared) func(context.Context, string) (interface{}, error) {
				return func(_ context.Context, id string) (interface{}, error) {
					require.Equal(t, shared.clientReferenceID, id)

					steamID, err := rand.GenerateString(11)
					require.Nil(t, err)

					shared.serverID = uuid.New()
					shared.steamID = steamID

					return &staging.Checkout{
						ServerID: shared.serverID,
						SteamID:  steamID,
					}, nil
				}
			},
			storeFirstVipByStripeEventID: func(t *testing.T, _ *shared) func(context.Context, string) (*model.Vip, error) {
				return func(_ context.Context, id string) (*model.Vip, error) {
					require.NotEmpty(t, id)

					return nil, igorm.ErrNotFound
				}
			},
			storeCreateVip: func(t *testing.T, shared *shared) func(context.Context, *model.Vip, *model.Customer) error {
				return func(_ context.Context, vip *model.Vip, customer *model.Customer) error {
					require.Equal(t, shared.stripeCheckoutID, vip.StripeCheckoutID)
					require.Equal(t, shared.stripeEventID, vip.StripeEventID)
					require.Equal(t, shared.serverID, vip.ServerID)

					fiveDays := 5 * 24 * time.Hour
					require.WithinDuration(t, time.Now().Add(fiveDays).UTC(), vip.ExpiresAt, time.Second)

					shared.expiresAt = vip.ExpiresAt

					require.Equal(t, shared.stripeCustomerID, customer.StripeCustomerID)
					require.Equal(t, shared.steamID, customer.SteamID)

					return nil
				}
			},
			streamWrite: func(t *testing.T, shared *shared) func(context.Context, []byte) error {
				return func(_ context.Context, b []byte) error {
					var event event.VipRefreshEvent
					err := json.Unmarshal(b, &event)
					require.Nil(t, err)

					require.Equal(t, shared.serverID, event.ServerID)
					require.Equal(t, shared.steamID, event.SteamID)
					require.Equal(t, shared.expiresAt, event.ExpiresAt)

					return nil
				}
			},
			exp: expected{
				err: nil,
			},
		},
		"payment checkout session complete duplicate": {
			event: func(t *testing.T, shared *shared) *event.StripeWebhookEvent {
				shared.clientReferenceID = uuid.New().String()
				shared.stripeEventID = uuid.New().String()
				shared.stripeCheckoutID = uuid.New().String()
				shared.stripeCustomerID = uuid.New().String()

				checkout := map[string]interface{}{
					"id":                  shared.stripeCheckoutID,
					"mode":                "payment",
					"client_reference_id": shared.clientReferenceID,
					"customer": map[string]interface{}{
						"id": shared.stripeCustomerID,
					},
					"payment_status": "paid",
					"line_items": map[string]interface{}{
						"data": []map[string]interface{}{
							{"price": istripe.WeeklyVipOneTime},
						},
					},
				}
				raw, err := json.Marshal(&checkout)
				require.Nil(t, err)

				event := event.NewStripeWebhookEvent(
					stripe.Event{
						ID:      shared.stripeEventID,
						Type:    "checkout.session.completed",
						Created: time.Now().Unix(),
						Data: &stripe.EventData{
							Object: checkout,
							Raw:    raw,
						},
					},
				)
				return &event
			},
			stagingFetchCheckout: func(t *testing.T, shared *shared) func(context.Context, string) (interface{}, error) {
				return func(_ context.Context, id string) (interface{}, error) {
					require.Equal(t, shared.clientReferenceID, id)

					steamID, err := rand.GenerateString(11)
					require.Nil(t, err)

					shared.serverID = uuid.New()
					shared.steamID = steamID

					return &staging.Checkout{
						ServerID: shared.serverID,
						SteamID:  steamID,
					}, nil
				}
			},
			storeFirstVipByStripeEventID: func(t *testing.T, shared *shared) func(context.Context, string) (*model.Vip, error) {
				return func(_ context.Context, id string) (*model.Vip, error) {
					require.NotEmpty(t, id)

					oneMinuteAgo := time.Now().Add(-time.Minute)
					fiveDays := 5 * 24 * time.Hour

					return &model.Vip{
						Model: imodel.Model{
							ID: uuid.New(),
							At: imodel.At{
								CreatedAt: oneMinuteAgo,
								UpdatedAt: oneMinuteAgo,
								DeletedAt: gorm.DeletedAt{Valid: false},
							},
						},
						StripeCheckoutID: shared.stripeCheckoutID,
						StripeEventID:    shared.stripeEventID,
						ServerID:         shared.serverID,
						CustomerID:       uuid.New(),
						ExpiresAt:        oneMinuteAgo.Add(fiveDays),
					}, nil
				}
			},
			storeCreateVip: func(t *testing.T, _ *shared) func(context.Context, *model.Vip, *model.Customer) error {
				return func(_ context.Context, _ *model.Vip, _ *model.Customer) error {
					require.FailNow(t, "store.CreateVip should not be called.")
					return nil
				}
			},
			streamWrite: func(t *testing.T, shared *shared) func(context.Context, []byte) error {
				return func(_ context.Context, b []byte) error {
					require.FailNow(t, "stream.Write should not be called.")
					return nil
				}
			},
			exp: expected{
				err: nil,
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			shared := &shared{}

			staging := staging.NewClientMock(
				staging.WithFetchCheckout(test.stagingFetchCheckout(t, shared)),
			)
			store := db.NewStoreMock(
				db.WithFirstVipByStripeEventID(test.storeFirstVipByStripeEventID(t, shared)),
				db.WithCreateVip(test.storeCreateVip(t, shared)),
			)
			stream := stream.NewClientMock(
				stream.WithWrite(test.streamWrite(t, shared)),
			)
			handler := NewHandler(
				zap.NewNop(),
				staging,
				store,
				stream,
			)

			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			err := handler.handleStripeEvent(ctx, test.event(t, shared))
			require.ErrorIs(t, err, test.exp.err)
		})
	}
}

func TestHandleSubscriptionCheckoutSessionComplete(t *testing.T) {
	// shared encompasses state shared between stages of the test. These values
	// are often generated and are therefore unique for each test run. Therefore,
	// they are passed between stages of the test to check for equality.
	type shared struct {
		clientReferenceID    string
		steamID              string
		serverID             uuid.UUID
		userID               uuid.UUID
		stripeEventID        string
		stripeCheckoutID     string
		stripeSubscriptionID string
		stripeCustomerID     string
		expiresAt            time.Time
	}

	// expected encompasses expected values to test for.
	type expected struct {
		err error
	}

	tests := map[string]struct {
		event                        func(*testing.T, *shared) *event.StripeWebhookEvent
		stagingFetchCheckout         func(*testing.T, *shared) func(context.Context, string) (interface{}, error)
		storeFirstVipByStripeEventID func(*testing.T, *shared) func(context.Context, string) (*model.Vip, error)
		storeCreateVipSubscription   func(*testing.T, *shared) func(context.Context, *model.Vip, *model.Subscription, *model.Customer, *model.User) error
		exp                          expected
	}{
		"subscription checkout session complete": {
			event: func(t *testing.T, shared *shared) *event.StripeWebhookEvent {
				shared.clientReferenceID = uuid.New().String()
				shared.stripeEventID = uuid.New().String()
				shared.stripeCheckoutID = uuid.New().String()
				shared.stripeSubscriptionID = uuid.New().String()
				shared.stripeCustomerID = uuid.New().String()

				checkout := map[string]interface{}{
					"id":                  shared.stripeCheckoutID,
					"mode":                "subscription",
					"client_reference_id": shared.clientReferenceID,
					"customer": map[string]interface{}{
						"id": shared.stripeCustomerID,
					},
					"subscription": map[string]interface{}{
						"id": shared.stripeSubscriptionID,
					},
					"payment_status": "paid",
					"line_items": map[string]interface{}{
						"data": []map[string]interface{}{
							{"price": istripe.MonthlyVipSubscription},
						},
					},
				}
				raw, err := json.Marshal(&checkout)
				require.Nil(t, err)

				event := event.NewStripeWebhookEvent(
					stripe.Event{
						ID:      shared.stripeEventID,
						Type:    "checkout.session.completed",
						Created: time.Now().Unix(),
						Data: &stripe.EventData{
							Object: checkout,
							Raw:    raw,
						},
					},
				)
				return &event
			},
			stagingFetchCheckout: func(t *testing.T, shared *shared) func(context.Context, string) (interface{}, error) {
				return func(_ context.Context, id string) (interface{}, error) {
					require.Equal(t, shared.clientReferenceID, id)

					steamID, err := rand.GenerateString(11)
					require.Nil(t, err)

					shared.serverID = uuid.New()
					shared.steamID = steamID
					shared.userID = uuid.New()

					return &staging.UserCheckout{
						Checkout: staging.Checkout{
							ServerID: shared.serverID,
							SteamID:  steamID,
						},
						UserID: shared.userID,
					}, nil
				}
			},
			storeFirstVipByStripeEventID: func(t *testing.T, _ *shared) func(context.Context, string) (*model.Vip, error) {
				return func(_ context.Context, id string) (*model.Vip, error) {
					require.NotEmpty(t, id)

					return nil, igorm.ErrNotFound
				}
			},
			storeCreateVipSubscription: func(t *testing.T, shared *shared) func(context.Context, *model.Vip, *model.Subscription, *model.Customer, *model.User) error {
				return func(
					_ context.Context,
					vip *model.Vip,
					subscription *model.Subscription,
					customer *model.Customer,
					user *model.User,
				) error {
					require.Equal(t, shared.stripeCheckoutID, vip.StripeCheckoutID)
					require.Equal(t, shared.stripeEventID, vip.StripeEventID)
					require.Equal(t, shared.serverID, vip.ServerID)

					thirtyDays := 30 * 24 * time.Hour
					require.WithinDuration(t, time.Now().Add(thirtyDays).UTC(), vip.ExpiresAt, time.Second)
					shared.expiresAt = vip.ExpiresAt

					require.Equal(t, shared.stripeSubscriptionID, subscription.StripeSubscriptionID)

					require.Equal(t, shared.stripeCustomerID, customer.StripeCustomerID)
					require.Equal(t, shared.steamID, customer.SteamID)

					require.Equal(t, shared.userID, user.ID)

					return nil
				}
			},
			exp: expected{
				err: nil,
			},
		},
		"payment subscription session complete duplicate": {
			event: func(t *testing.T, shared *shared) *event.StripeWebhookEvent {
				shared.clientReferenceID = uuid.New().String()
				shared.stripeEventID = uuid.New().String()
				shared.stripeCheckoutID = uuid.New().String()
				shared.stripeSubscriptionID = uuid.New().String()
				shared.stripeCustomerID = uuid.New().String()

				checkout := map[string]interface{}{
					"id":                  shared.stripeCheckoutID,
					"mode":                "subscription",
					"client_reference_id": shared.clientReferenceID,
					"customer": map[string]interface{}{
						"id": shared.stripeCustomerID,
					},
					"subscription": map[string]interface{}{
						"id": shared.stripeSubscriptionID,
					},
					"payment_status": "paid",
					"line_items": map[string]interface{}{
						"data": []map[string]interface{}{
							{"price": istripe.MonthlyVipSubscription},
						},
					},
				}
				raw, err := json.Marshal(&checkout)
				require.Nil(t, err)

				event := event.NewStripeWebhookEvent(
					stripe.Event{
						ID:      shared.stripeEventID,
						Type:    "checkout.session.completed",
						Created: time.Now().Unix(),
						Data: &stripe.EventData{
							Object: checkout,
							Raw:    raw,
						},
					},
				)
				return &event
			},
			stagingFetchCheckout: func(t *testing.T, shared *shared) func(context.Context, string) (interface{}, error) {
				return func(_ context.Context, id string) (interface{}, error) {
					require.Equal(t, shared.clientReferenceID, id)

					steamID, err := rand.GenerateString(11)
					require.Nil(t, err)

					shared.serverID = uuid.New()
					shared.steamID = steamID
					shared.userID = uuid.New()

					return &staging.UserCheckout{
						Checkout: staging.Checkout{
							ServerID: shared.serverID,
							SteamID:  steamID,
						},
						UserID: shared.userID,
					}, nil
				}
			},
			storeFirstVipByStripeEventID: func(t *testing.T, shared *shared) func(context.Context, string) (*model.Vip, error) {
				return func(_ context.Context, id string) (*model.Vip, error) {
					require.NotEmpty(t, id)

					oneMinuteAgo := time.Now().Add(-time.Minute)
					fiveDays := 5 * 24 * time.Hour

					return &model.Vip{
						Model: imodel.Model{
							ID: uuid.New(),
							At: imodel.At{
								CreatedAt: oneMinuteAgo,
								UpdatedAt: oneMinuteAgo,
								DeletedAt: gorm.DeletedAt{Valid: false},
							},
						},
						StripeCheckoutID: shared.stripeCheckoutID,
						StripeEventID:    shared.stripeEventID,
						ServerID:         shared.serverID,
						CustomerID:       uuid.New(),
						ExpiresAt:        oneMinuteAgo.Add(fiveDays),
					}, nil
				}
			},
			storeCreateVipSubscription: func(t *testing.T, shared *shared) func(context.Context, *model.Vip, *model.Subscription, *model.Customer, *model.User) error {
				return func(
					_ context.Context,
					vip *model.Vip,
					subscription *model.Subscription,
					customer *model.Customer,
					user *model.User,
				) error {
					require.FailNow(t, "store.CreateVipSubscription should not be called.")
					return nil
				}
			},
			exp: expected{
				err: nil,
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			shared := &shared{}

			staging := staging.NewClientMock(
				staging.WithFetchCheckout(test.stagingFetchCheckout(t, shared)),
			)
			store := db.NewStoreMock(
				db.WithFirstVipByStripeEventID(test.storeFirstVipByStripeEventID(t, shared)),
				db.WithCreateVipSubscription(test.storeCreateVipSubscription(t, shared)),
			)
			stream := stream.NewClientMock()
			handler := NewHandler(
				zap.NewNop(),
				staging,
				store,
				stream,
			)

			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			err := handler.handleStripeEvent(ctx, test.event(t, shared))
			require.ErrorIs(t, err, test.exp.err)
		})
	}
}

func TestHandleInvoice(t *testing.T) {
	// shared encompasses state shared between stages of the test. These values
	// are often generated and are therefore unique for each test run. Therefore,
	// they are passed between stages of the test to check for equality.
	type shared struct {
		steamID              string
		serverID             uuid.UUID
		expiresAt            time.Time
		stripeEventID        string
		stripeSubscriptionID string
		paymentStatus        model.InvoiceStatus
	}

	// expected encompasses expected values to test for.
	type expected struct {
		err error
	}

	tests := map[string]struct {
		event                            func(*testing.T, *shared) *event.StripeWebhookEvent
		storeFirstInvoiceByStripeEventID func(*testing.T, *shared) func(context.Context, string) (*model.Invoice, error)
		storeAddInvoiceToVipSubscription func(*testing.T, *shared) func(context.Context, string, *model.Invoice) (*model.Vip, error)
		streamWrite                      func(*testing.T, *shared) func(context.Context, []byte) error
		exp                              expected
	}{
		"paid invoice": {
			event: func(t *testing.T, shared *shared) *event.StripeWebhookEvent {
				shared.stripeEventID = uuid.New().String()
				shared.stripeSubscriptionID = uuid.New().String()
				shared.paymentStatus = "paid"

				invoice := map[string]interface{}{
					"id": uuid.New().String(),
					"subscription": map[string]interface{}{
						"id": shared.stripeSubscriptionID,
					},
					"status": shared.paymentStatus,
				}
				raw, err := json.Marshal(&invoice)
				require.Nil(t, err)

				event := event.NewStripeWebhookEvent(
					stripe.Event{
						ID:      shared.stripeEventID,
						Type:    "invoice.paid",
						Created: time.Now().Unix(),
						Data: &stripe.EventData{
							Object: invoice,
							Raw:    raw,
						},
					},
				)
				return &event
			},
			storeFirstInvoiceByStripeEventID: func(t *testing.T, shared *shared) func(context.Context, string) (*model.Invoice, error) {
				return func(_ context.Context, id string) (*model.Invoice, error) {
					require.Equal(t, shared.stripeEventID, id)

					return nil, igorm.ErrNotFound
				}
			},
			storeAddInvoiceToVipSubscription: func(t *testing.T, shared *shared) func(context.Context, string, *model.Invoice) (*model.Vip, error) {
				return func(
					ctx context.Context,
					subscriptionID string,
					invoice *model.Invoice,
				) (*model.Vip, error) {
					require.Equal(t, shared.stripeSubscriptionID, subscriptionID)
					require.Equal(t, shared.paymentStatus, invoice.Status)
					require.Equal(t, shared.stripeEventID, invoice.StripeEventID)

					steamID, err := rand.GenerateString(11)
					require.Nil(t, err)

					shared.steamID = steamID
					shared.serverID = uuid.New()

					thirtyDays := 30 * 24 * time.Hour
					shared.expiresAt = time.Now().Add(thirtyDays).UTC()

					return &model.Vip{
						Model: imodel.Model{
							ID: [16]byte{},
							At: imodel.At{
								CreatedAt: time.Now().Add(-thirtyDays).UTC(),
								UpdatedAt: time.Now().Add(-thirtyDays).UTC(),
								DeletedAt: gorm.DeletedAt{},
							},
						},
						Server:    model.Server{ID: shared.serverID},
						Customer:  model.Customer{SteamID: shared.steamID},
						ExpiresAt: shared.expiresAt,
					}, nil
				}
			},
			streamWrite: func(t *testing.T, shared *shared) func(context.Context, []byte) error {
				return func(_ context.Context, b []byte) error {
					var event event.VipRefreshEvent
					err := json.Unmarshal(b, &event)
					require.Nil(t, err)

					require.Equal(t, shared.serverID, event.ServerID)
					require.Equal(t, shared.steamID, event.SteamID)
					require.Equal(t, shared.expiresAt, event.ExpiresAt)

					return nil
				}
			},
			exp: expected{
				err: nil,
			},
		},
		"paid invoice duplicate": {
			event: func(t *testing.T, shared *shared) *event.StripeWebhookEvent {
				shared.stripeEventID = uuid.New().String()
				shared.stripeSubscriptionID = uuid.New().String()
				shared.paymentStatus = "paid"

				invoice := map[string]interface{}{
					"id": uuid.New().String(),
					"subscription": map[string]interface{}{
						"id": shared.stripeSubscriptionID,
					},
					"status": shared.paymentStatus,
				}
				raw, err := json.Marshal(&invoice)
				require.Nil(t, err)

				event := event.NewStripeWebhookEvent(
					stripe.Event{
						ID:      shared.stripeEventID,
						Type:    "invoice.paid",
						Created: time.Now().Unix(),
						Data: &stripe.EventData{
							Object: invoice,
							Raw:    raw,
						},
					},
				)
				return &event
			},
			storeFirstInvoiceByStripeEventID: func(t *testing.T, shared *shared) func(context.Context, string) (*model.Invoice, error) {
				return func(_ context.Context, id string) (*model.Invoice, error) {
					require.Equal(t, shared.stripeEventID, id)

					return &model.Invoice{
						SubscriptionID: uuid.New(),
						StripeEventID:  id,
						Status:         shared.paymentStatus,
					}, nil
				}
			},
			storeAddInvoiceToVipSubscription: func(t *testing.T, shared *shared) func(context.Context, string, *model.Invoice) (*model.Vip, error) {
				return func(
					ctx context.Context,
					subscriptionID string,
					invoice *model.Invoice,
				) (*model.Vip, error) {
					require.FailNow(t, "store.AddInvoiceToVipSubscription should not be called.")
					return nil, nil
				}
			},
			streamWrite: func(t *testing.T, shared *shared) func(context.Context, []byte) error {
				return func(_ context.Context, b []byte) error {
					var event event.VipRefreshEvent
					err := json.Unmarshal(b, &event)
					require.Nil(t, err)

					require.Equal(t, shared.serverID, event.ServerID)
					require.Equal(t, shared.steamID, event.SteamID)
					require.Equal(t, shared.expiresAt, event.ExpiresAt)

					return nil
				}
			},
			exp: expected{
				err: nil,
			},
		},
		"payment_failed invoice": {
			event: func(t *testing.T, shared *shared) *event.StripeWebhookEvent {
				shared.stripeEventID = uuid.New().String()
				shared.stripeSubscriptionID = uuid.New().String()
				shared.paymentStatus = "uncollectible"

				invoice := map[string]interface{}{
					"id": uuid.New().String(),
					"subscription": map[string]interface{}{
						"id": shared.stripeSubscriptionID,
					},
					"status": shared.paymentStatus,
				}
				raw, err := json.Marshal(&invoice)
				require.Nil(t, err)

				event := event.NewStripeWebhookEvent(
					stripe.Event{
						ID:      shared.stripeEventID,
						Type:    "invoice.payment_failed",
						Created: time.Now().Unix(),
						Data: &stripe.EventData{
							Object: invoice,
							Raw:    raw,
						},
					},
				)
				return &event
			},
			storeFirstInvoiceByStripeEventID: func(t *testing.T, shared *shared) func(context.Context, string) (*model.Invoice, error) {
				return func(_ context.Context, id string) (*model.Invoice, error) {
					require.Equal(t, shared.stripeEventID, id)

					return nil, igorm.ErrNotFound
				}
			},
			storeAddInvoiceToVipSubscription: func(t *testing.T, shared *shared) func(context.Context, string, *model.Invoice) (*model.Vip, error) {
				return func(
					ctx context.Context,
					subscriptionID string,
					invoice *model.Invoice,
				) (*model.Vip, error) {
					require.Equal(t, shared.stripeSubscriptionID, subscriptionID)
					require.Equal(t, shared.paymentStatus, invoice.Status)
					require.Equal(t, shared.stripeEventID, invoice.StripeEventID)

					steamID, err := rand.GenerateString(11)
					require.Nil(t, err)

					shared.steamID = steamID
					shared.serverID = uuid.New()

					thirtyDays := 30 * 24 * time.Hour
					shared.expiresAt = time.Now().Add(thirtyDays).UTC()

					return &model.Vip{
						Model: imodel.Model{
							ID: [16]byte{},
							At: imodel.At{
								CreatedAt: time.Now().Add(-thirtyDays).UTC(),
								UpdatedAt: time.Now().Add(-thirtyDays).UTC(),
								DeletedAt: gorm.DeletedAt{},
							},
						},
						Server:    model.Server{ID: shared.serverID},
						Customer:  model.Customer{SteamID: shared.steamID},
						ExpiresAt: shared.expiresAt,
					}, nil
				}
			},
			streamWrite: func(t *testing.T, _ *shared) func(context.Context, []byte) error {
				return func(_ context.Context, b []byte) error {
					require.FailNow(t, "stream.Write should not be called.")
					return nil
				}
			},
			exp: expected{
				err: nil,
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			shared := new(shared)

			staging := staging.NewClientMock()
			store := db.NewStoreMock(
				db.WithFirstInvoiceByStripeEventID(test.storeFirstInvoiceByStripeEventID(t, shared)),
				db.WithAddInvoiceToVipSubscription(test.storeAddInvoiceToVipSubscription(t, shared)),
			)
			stream := stream.NewClientMock(
				stream.WithWrite(test.streamWrite(t, shared)),
			)
			handler := NewHandler(
				zap.NewNop(),
				staging,
				store,
				stream,
			)

			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			err := handler.handleStripeEvent(ctx, test.event(t, shared))
			require.ErrorIs(t, err, test.exp.err)
		})
	}
}

func TestLaunch(t *testing.T) {
	claim := func(context.Context, time.Duration) (*stream.Message, error) {
		return nil, stream.ErrNoPending
	}

	readc := make(chan stream.Message)
	read := func(context.Context) (*stream.Message, error) {
		m := <-readc
		return &m, nil
	}

	ackc := make(chan struct{})
	ack := func(_ context.Context, _ *stream.Message) error {
		ackc <- struct{}{}
		return nil
	}

	streamClient := stream.NewClientMock(
		stream.WithClaim(claim),
		stream.WithRead(read),
		stream.WithAck(ack),
	)
	handler := NewHandler(
		zap.NewNop(),
		staging.NewClientMock(),
		db.NewStoreMock(),
		streamClient,
	)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	go func() {
		err := handler.Launch(ctx)
		require.ErrorIs(t, err, context.Canceled)
	}()

	// Build a stream.Message to for Launch to handle.
	steamID, err := rand.GenerateString(11)
	require.Nil(t, err)

	serverID := uuid.New()
	expiresAt := time.Now().Add(5 * 24 * time.Hour).UTC()

	event := event.NewVipRefreshEvent(serverID, steamID, expiresAt)
	b, err := json.Marshal(&event)
	require.Nil(t, err)

	readc <- stream.Message{
		ID:      uuid.NewString(),
		Payload: b,
	}

	select {
	case <-ctx.Done():
		require.FailNow(t, "Context should not be done before an ack occurs.")
	case <-ackc:
		break
	}
}
