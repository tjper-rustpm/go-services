package rcon

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"go.uber.org/zap"
)

// ErrRoutingIdentifier indicates that the router was unable to route a message
// because routing does not exist for the identifier passed.
var ErrRoutingIdentifier = errors.New("identifier routing DNE")

func NewRouter(logger *zap.Logger) *Router {
	return &Router{
		logger: logger,
		mutex:  new(sync.RWMutex),
		sendc:  make(chan Outbound, 1),
		routes: make(map[int]chan Inbound),
	}
}

// Router is responsible for routing Inbound and Outbound messages based on
// their Identifier fields.
type Router struct {
	logger *zap.Logger

	mutex  *sync.RWMutex
	sendc  chan Outbound
	routes map[int]chan Inbound
}

// Write sends the Outbound message and does expect a response.
func (r *Router) Write(ctx context.Context, out Outbound) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case r.sendc <- out:
	}
	return nil
}

// Request sends the Outbound message and waits for the corresponding response
// in the form of an Inbound message.
func (r *Router) Request(ctx context.Context, out Outbound) (chan Inbound, error) {
	var route chan Inbound
	sendRoute := func() {
		r.mutex.Lock()
		defer r.mutex.Unlock()

		route = make(chan Inbound, 1)
		r.routes[out.Identifier] = route
	}

	sendRoute()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case r.sendc <- out:
	}
	return route, nil
}

// CloseRoute closes the route associated with the specified identifier.
func (r *Router) CloseRoute(identifier int) {
	r.mutex.Lock()
	delete(r.routes, identifier)
	r.mutex.Unlock()
}

// Injest accepts a Inbound message and routes to the waiting Request process.
func (r *Router) Injest(close chan struct{}, in Inbound) error {
	var route chan Inbound
	fetchRoute := func() error {
		r.mutex.Lock()
		defer r.mutex.Unlock()

		routec, ok := r.routes[in.Identifier]
		if !ok {
			return fmt.Errorf("no routing for message with identifier %d; %w", in.Identifier, ErrRoutingIdentifier)
		}
		route = routec
		return nil
	}

	if err := fetchRoute(); err != nil {
		return err
	}
	select {
	case <-close:
		return nil
	case route <- in:
		return nil
	}
}

// Outboundc returns a channel responsible for all Outbound messages that are
// sent via Router.Request.
func (r *Router) Outboundc() chan Outbound {
	return r.sendc
}
