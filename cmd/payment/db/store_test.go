package db

import (
	"context"
	"fmt"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/tjper/rustcron/cmd/payment/model"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

var (
	dsn        = os.Getenv("PAYMENT_DSN")
	migrations = os.Getenv("PAYMENT_MIGRATIONS")
)

func TestCreateServer(t *testing.T) {
	skipIfMissingDatabaseEnvVars(t)

	type expected struct {
		server model.Server
	}
	tests := map[string]struct {
		server model.Server
		exp    expected
	}{
		"200 subscription limit": {
			server: model.Server{
				ID:                  alphaServerID,
				ActiveSubscriptions: 0,
				SubscriptionLimit:   200,
			},
			exp: expected{
				server: model.Server{
					ID:                  alphaServerID,
					ActiveSubscriptions: 0,
					SubscriptionLimit:   200,
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			store := newStore(t)

			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			err := store.CreateServer(ctx, &test.server)
			require.Nil(t, err)
			defer func() {
				err := store.db.WithContext(ctx).Unscoped().Delete(&test.server).Error
				require.Nil(t, err)
			}()

			actual := test.server.Clone()
			expected := test.exp.server

			actual.Scrub()
			expected.Scrub()

			require.Equal(t, expected, *actual)
		})
	}
}

func TestCreateVipCustomerDoesNotExist(t *testing.T) {
	testCreateVip(t, false)
}
func TestCreateVipCustomerDoesExist(t *testing.T) {
	testCreateVip(t, true)
}

func testCreateVip(t *testing.T, customerExists bool) {
	skipIfMissingDatabaseEnvVars(t)

	test := struct {
		server   *model.Server
		vip      *model.Vip
		customer *model.Customer
	}{
		server: &model.Server{
			ID:                  alphaServerID,
			ActiveSubscriptions: 0,
			SubscriptionLimit:   200,
		},
		vip: &model.Vip{
			StripeCheckoutID: "stripe-checkout-id",
			StripeEventID:    "stripe-event-id",
			ServerID:         alphaServerID,
			ExpiresAt:        time.Now().Add(time.Hour).UTC(),
		},
		customer: &model.Customer{
			StripeCustomerID: "stripe-customer-id",
			SteamID:          "steam-id",
		},
	}

	store := newStore(t)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := store.CreateServer(ctx, test.server)
	require.Nil(t, err)
	defer func() {
		err := store.db.WithContext(ctx).Unscoped().Delete(test.server).Error
		require.Nil(t, err)
	}()

	if customerExists {
		err = store.db.WithContext(ctx).Create(test.customer).Error
		require.Nil(t, err)
	}

	err = store.CreateVip(ctx, test.vip, test.customer)
	require.Nil(t, err)
	defer func() {
		err := store.db.WithContext(ctx).Unscoped().Delete(test.vip).Error
		require.Nil(t, err)

		err = store.db.WithContext(ctx).Unscoped().
			Delete(test.customer).Error
		require.Nil(t, err)
	}()
}

func TestCreateVipSubscriptionCustomerDoesNotExists(t *testing.T) {
	testCreateVipSubscription(t, false)
}
func TestCreateVipSubscriptionCustomerDoesExists(t *testing.T) {
	testCreateVipSubscription(t, true)
}

func testCreateVipSubscription(t *testing.T, customerExists bool) {
	skipIfMissingDatabaseEnvVars(t)

	test := struct {
		server       *model.Server
		vip          *model.Vip
		subscription *model.Subscription
		customer     *model.Customer
		user         *model.User
	}{
		server: &model.Server{
			ID:                  alphaServerID,
			ActiveSubscriptions: 0,
			SubscriptionLimit:   200,
		},
		vip: &model.Vip{
			StripeCheckoutID: "stripe-checkout-id",
			StripeEventID:    "stripe-event-id",
			ServerID:         alphaServerID,
			ExpiresAt:        time.Now().Add(time.Hour).UTC(),
		},
		subscription: &model.Subscription{
			StripeSubscriptionID: "stripe-subscription-id",
		},
		customer: &model.Customer{
			StripeCustomerID: "stripe-customer-id",
			SteamID:          "steam-id",
		},
		user: &model.User{
			ID: alphaUserID,
		},
	}

	store := newStore(t)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := store.CreateServer(ctx, test.server)
	require.Nil(t, err)
	defer func() {
		err := store.db.WithContext(ctx).Unscoped().Delete(test.server).Error
		require.Nil(t, err)
	}()

	if customerExists {
		err = store.db.WithContext(ctx).Create(test.customer).Error
		require.Nil(t, err)
	}

	err = store.CreateVipSubscription(
		ctx,
		test.vip,
		test.subscription,
		test.customer,
		test.user,
	)
	require.Nil(t, err)
	defer func() {
		err := store.db.WithContext(ctx).Unscoped().Delete(test.subscription).Error
		require.Nil(t, err)

		err = store.db.WithContext(ctx).Unscoped().Delete(test.user).Error
		require.Nil(t, err)

		err = store.db.WithContext(ctx).Unscoped().Delete(test.vip).Error
		require.Nil(t, err)

		err = store.db.WithContext(ctx).Unscoped().Delete(test.customer).Error
		require.Nil(t, err)
	}()
}

func TestAddInvoiceToVipSubscription(t *testing.T) {
	skipIfMissingDatabaseEnvVars(t)

	test := struct {
		server       *model.Server
		vip          *model.Vip
		subscription *model.Subscription
		customer     *model.Customer
		user         *model.User
		invoice      *model.Invoice
	}{
		server: &model.Server{
			ID:                  alphaServerID,
			ActiveSubscriptions: 0,
			SubscriptionLimit:   200,
		},
		vip: &model.Vip{
			StripeCheckoutID: "stripe-checkout-id",
			StripeEventID:    "stripe-checkout-event-id",
			ServerID:         alphaServerID,
			ExpiresAt:        time.Now().Add(time.Hour).UTC(),
		},
		subscription: &model.Subscription{
			StripeSubscriptionID: "stripe-subscription-id",
		},
		customer: &model.Customer{
			StripeCustomerID: "stripe-customer-id",
			SteamID:          "steam-id",
		},
		user: &model.User{
			ID: alphaUserID,
		},
		invoice: &model.Invoice{
			StripeEventID: "stripe-invoice-event-id",
			Status:        model.InvoiceStatusPaid,
		},
	}

	store := newStore(t)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := store.CreateServer(ctx, test.server)
	require.Nil(t, err)
	defer func() {
		err := store.db.WithContext(ctx).Unscoped().Delete(test.server).Error
		require.Nil(t, err)
	}()

	err = store.CreateVipSubscription(
		ctx,
		test.vip,
		test.subscription,
		test.customer,
		test.user,
	)
	require.Nil(t, err)
	defer func() {
		err := store.db.WithContext(ctx).Unscoped().Delete(test.subscription).Error
		require.Nil(t, err)

		err = store.db.WithContext(ctx).Unscoped().Delete(test.user).Error
		require.Nil(t, err)

		err = store.db.WithContext(ctx).Unscoped().Delete(test.vip).Error
		require.Nil(t, err)

		err = store.db.WithContext(ctx).Unscoped().Delete(test.customer).Error
		require.Nil(t, err)
	}()

	initialExpiredAt := test.vip.ExpiresAt
	require.WithinDuration(
		t,
		time.Now().Add(time.Hour).UTC(),
		initialExpiredAt,
		time.Second,
	)

	vip, err := store.AddInvoiceToVipSubscription(
		ctx,
		test.subscription.StripeSubscriptionID,
		test.invoice,
	)
	require.Nil(t, err)
	defer func() {
		err := store.db.WithContext(ctx).Unscoped().Delete(test.invoice).Error
		require.Nil(t, err)
	}()

	// Require that VIP ExpiredAt attribute has been updated.
	thirtyDays := 30 * 24 * time.Hour
	require.WithinDuration(
		t,
		time.Now().Add(thirtyDays).UTC(),
		vip.ExpiresAt,
		time.Second,
	)
	require.NotEqual(t, initialExpiredAt, vip.ExpiresAt)

	require.Equal(t, alphaServerID, vip.Server.ID)
	require.Equal(t, "steam-id", vip.Customer.SteamID)
}

func TestFindServers(t *testing.T) {
	skipIfMissingDatabaseEnvVars(t)

	future := time.Now().Add(time.Minute).UTC()
	past := time.Now().Add(-time.Minute).UTC()

	newVip := func(serverID uuid.UUID, expiresAt time.Time) model.Vip {
		id := uuid.New()
		return model.Vip{
			StripeCheckoutID: fmt.Sprintf("stripe-checkout-%s", id),
			StripeEventID:    fmt.Sprintf("stripe-event-%s", id),
			ServerID:         serverID,
			ExpiresAt:        expiresAt,
		}
	}

	type expected struct {
		servers model.Servers
	}
	tests := map[string]struct {
		servers model.Servers
		vips    map[model.Customer]model.Vips
		exp     expected
	}{
		"one server, zero vips": {
			servers: model.Servers{
				{
					ID:                alphaServerID,
					SubscriptionLimit: 200,
				},
			},
			vips: map[model.Customer]model.Vips{},
			exp: expected{
				servers: model.Servers{
					{
						ID:                  alphaServerID,
						ActiveSubscriptions: 0,
						SubscriptionLimit:   200,
					},
				},
			},
		},
		"one server, one vip": {
			servers: model.Servers{
				{
					ID:                alphaServerID,
					SubscriptionLimit: 200,
				},
			},
			vips: map[model.Customer]model.Vips{
				alphaCustomer: {
					newVip(alphaServerID, future),
				},
			},
			exp: expected{
				servers: model.Servers{
					{
						ID:                  alphaServerID,
						ActiveSubscriptions: 1,
						SubscriptionLimit:   200,
					},
				},
			},
		},
		"one server, two customers, two vips": {
			servers: model.Servers{
				{
					ID:                alphaServerID,
					SubscriptionLimit: 200,
				},
			},
			vips: map[model.Customer]model.Vips{
				alphaCustomer: {
					newVip(alphaServerID, future),
				},
				bravoCustomer: {
					newVip(alphaServerID, future),
				},
			},
			exp: expected{
				servers: model.Servers{
					{
						ID:                  alphaServerID,
						ActiveSubscriptions: 2,
						SubscriptionLimit:   200,
					},
				},
			},
		},
		"two servers, one customer, two vips": {
			servers: model.Servers{
				{
					ID:                alphaServerID,
					SubscriptionLimit: 200,
				},
				{
					ID:                bravoServerID,
					SubscriptionLimit: 100,
				},
			},
			vips: map[model.Customer]model.Vips{
				alphaCustomer: {
					newVip(alphaServerID, future),
					newVip(bravoServerID, future),
				},
			},
			exp: expected{
				servers: model.Servers{
					{
						ID:                  alphaServerID,
						ActiveSubscriptions: 1,
						SubscriptionLimit:   200,
					},
					{
						ID:                  bravoServerID,
						ActiveSubscriptions: 1,
						SubscriptionLimit:   100,
					},
				},
			},
		},
		"one server, two customers, one active vip, one expired vip": {
			servers: model.Servers{
				{
					ID:                alphaServerID,
					SubscriptionLimit: 200,
				},
			},
			vips: map[model.Customer]model.Vips{
				alphaCustomer: {
					newVip(alphaServerID, future),
				},
				bravoCustomer: {
					newVip(alphaServerID, past),
				},
			},
			exp: expected{
				servers: model.Servers{
					{
						ID:                  alphaServerID,
						ActiveSubscriptions: 1,
						SubscriptionLimit:   200,
					},
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			store := newStore(t)

			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			createServer := func(server *model.Server) func() {
				err := store.CreateServer(ctx, server)
				require.Nil(t, err)
				return func() {
					err := store.db.WithContext(ctx).Unscoped().Delete(server).Error
					require.Nil(t, err)
				}
			}

			createCustomer := func(customer *model.Customer) func() {
				err := store.db.WithContext(ctx).Create(customer).Error
				require.Nil(t, err)
				return func() {
					err := store.db.WithContext(ctx).Unscoped().Delete(customer).Error
					require.Nil(t, err)
				}
			}

			createVip := func(vip *model.Vip, customer *model.Customer) func() {
				err := store.CreateVip(ctx, vip, customer)
				require.Nil(t, err)
				return func() {
					err := store.db.WithContext(ctx).Unscoped().Delete(vip).Error
					require.Nil(t, err)
				}
			}

			for _, server := range test.servers {
				server := server
				cleanup := createServer(&server)
				defer cleanup()
			}

			for customer := range test.vips {
				customer := customer
				cleanup := createCustomer(&customer)
				defer cleanup()
			}

			for customer, vips := range test.vips {
				customer := customer
				for _, vip := range vips {
					vip := vip
					cleanup := createVip(&vip, &customer)
					defer cleanup()
				}
			}

			servers, err := store.FindServers(ctx)
			require.Nil(t, err)

			sort.Slice(
				servers,
				func(i, j int) bool {
					return servers[i].ID.String() < servers[j].ID.String()
				},
			)

			expected := test.exp.servers
			sort.Slice(
				expected,
				func(i, j int) bool {
					return expected[i].ID.String() < expected[j].ID.String()
				},
			)

			test.exp.servers.Scrub()
			servers.Scrub()

			require.Equal(t, test.exp.servers, servers)
		})
	}
}

var (
	// NOTE: It is likely that these identifiers and variables are used by
	// multiple tests, before modifiying them, please take this into
	// consideration.

	// alphaServerID is an idenitifer for the alpha server.
	alphaServerID = uuid.New()

	// bravoServerID is an idenitifer for the bravo server.
	bravoServerID = uuid.New()

	// alphaUserID is an identifier for the alpha user.
	alphaUserID = uuid.New()

	// alphaCustomer represents the alpha customer.
	alphaCustomer = model.Customer{
		StripeCustomerID: "stripe-alpha-customer-id",
		SteamID:          "alpha-steam-id",
	}

	// bravoCustomer represents the bravo customer.
	bravoCustomer = model.Customer{
		StripeCustomerID: "stripe-bravo-customer-id",
		SteamID:          "bravo-steam-id",
	}
)

// newStore is a helper for creating a new Store instance within a test.
func newStore(t *testing.T) *Store {
	db, err := Open(dsn)
	require.Nil(t, err)

	err = Migrate(db, migrations)
	require.Nil(t, err)

	return NewStore(db)
}

func skipIfMissingDatabaseEnvVars(t *testing.T) {
	switch {
	case dsn == "":
		t.Skip("PAYMENT_DSN must be set to execute this test.")
	case migrations == "":
		t.Skip("PAYMENT_MIGRATIONS must be set to execute this test.")
	}
}
