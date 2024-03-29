package rcon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

var (
	// ErrModeratorExists indicates that the moderator being created via
	// Client.AddModerator already exists.
	ErrModeratorExists = errors.New("moderator already exists")

	// ErrModeratorDNE indicates that the moderator being removed via
	// Client.RemoveModerator does not exist.
	ErrModeratorDNE = errors.New("moderator does not exist")

	// ErrOwnerExists indicates that the owner being created via Client.AddOwner
	// already exists.
	ErrOwnerExists = errors.New("owner already exists")

	// ErrOwnerDNE indicates that the owner being removed via Client.RemoveOwner
	// does not exist.
	ErrOwnerDNE = errors.New("owner does not exist")

	// ErrPermissionAlreadyGranted indicates that the permission specified has
	// already been granted for the specified user.
	ErrPermissionAlreadyGranted = errors.New("permission has already been granted")

	// errRconClientUnexpectedClose may be returned when a process is interrupted
	// due to an unexpected Client.Close().
	errRconClientUnexpectedClose = errors.New("RCON client closing")
)

const (
	// DefaultRconPort is the default port used by a Rust server to RCON access.
	DefaultRconPort = 28016

	// BypassQueueAllow is the permission that allows a steam ID to bypass the
	// join queue.
	BypassQueueAllow = "bypassqueue.allow"

	// VipGroup is the Oxide group that Vips belong to.
	VipGroup = "vip"
)

var (
	errUnexpectedInboundMessage = errors.New("unexpected inbound message")

	errIdentifiersNotEqual = errors.New("identifiers not equal")

	errInboundTypeUnexpected = errors.New("inbound type not expected")
)

func Dial(
	ctx context.Context,
	logger *zap.Logger,
	url string,
) (*Client, error) {
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, url, http.Header{})
	if err != nil {
		return nil, err
	}

	closed := make(chan struct{}, 1)
	client := &Client{
		logger:    logger,
		conn:      conn,
		router:    NewRouter(logger),
		closed:    closed,
		closeOnce: new(sync.Once),
	}
	go func() {
		err := client.readPump()
		if err != nil {
			logger.Warn("error read pump", zap.Error(err))
		}
	}()
	go func() {
		err := client.writePump()
		if err != nil {
			logger.Warn("error write pump", zap.Error(err))
		}
	}()
	return client, nil
}

// Client represents a Rust server remote console client.
type Client struct {
	logger *zap.Logger

	conn   *websocket.Conn
	router *Router

	closed    chan struct{}
	closeOnce *sync.Once
}

// Close closes the RCON client, releasing its resources.
func (c *Client) Close() {
	c.closeOnce.Do(func() {
		close(c.router.Outboundc())
		<-c.closed
		c.conn.Close()
	})
}

// Say writes the specified message to the Rust server's chat.
func (c Client) Say(ctx context.Context, msg string) error {
	out := &Outbound{
		Identifier: -1,
		Message:    fmt.Sprintf("say %s", msg),
		Name:       "WebRcon",
	}
	if err := c.router.Write(ctx, *out); err != nil {
		return fmt.Errorf("error requesting say; %w", err)
	}
	defer c.router.CloseRoute(out.Identifier)

	return nil
}

type ServerInfo struct {
	Hostname        string
	MaxPlayers      int
	Players         int
	Queued          int
	Joining         int
	EntityCount     int
	GameTime        string
	Uptime          int
	Map             string
	Framerate       float32
	Memory          int
	Collections     int
	NetworkIn       float32
	NetworkOut      float32
	Restarting      bool
	SaveCreatedTime string
}

// ServerInfo requests the server info from the Rust server.
func (c Client) ServerInfo(ctx context.Context) (*ServerInfo, error) {
	out := NewOutbound("global.serverinfo")
	inboundc, err := c.router.Request(ctx, *out)
	if err != nil {
		return nil, fmt.Errorf("error requesting serverinfo; %w", err)
	}
	defer c.router.CloseRoute(out.Identifier)

	in, err := c.waitForInbound(ctx, inboundc)
	if err != nil {
		return nil, fmt.Errorf("error waiting for inbound; %w", err)
	}

	res := new(ServerInfo)
	if err := json.Unmarshal([]byte(in.Message), res); err != nil {
		return nil, err
	}
	return res, nil
}

// Quit saves and initiates the Rust server's shutdown process.
func (c Client) Quit(ctx context.Context) error {
	out := NewOutbound("global.quit")
	inboundc, err := c.router.Request(ctx, *out)
	if err != nil {
		return fmt.Errorf("error requesting quit; %w", err)
	}
	defer c.router.CloseRoute(out.Identifier)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.closed:
			return nil
		case _, ok := <-inboundc:
			if !ok {
				return nil
			}
		}
	}
}

// AddModerator adds the moderator specified by the id to the Rust server.
func (c Client) AddModerator(ctx context.Context, id string) error {
	out := NewOutbound(fmt.Sprintf("global.moderatorid \"%s\"", id))
	inboundc, err := c.router.Request(ctx, *out)
	if err != nil {
		return fmt.Errorf("error writing add-moderator; %w", err)
	}
	defer c.router.CloseRoute(out.Identifier)

	in, err := c.waitForInbound(ctx, inboundc)
	if err != nil {
		return fmt.Errorf("error waiting for inbound; %w", err)
	}

	if err := checkInbound(in, out.Identifier); err != nil {
		return err
	}
	if in.Message == fmt.Sprintf("User %s is already a Moderator", id) {
		return ErrModeratorExists
	}
	if in.Message != fmt.Sprintf("Added moderator unnamed, steamid %s", id) {
		return errUnexpectedInboundMessage
	}

	return nil
}

// RemoveModerator removes the moderator specified by the id from the Rust
// server.
func (c Client) RemoveModerator(ctx context.Context, id string) error {
	out := NewOutbound(fmt.Sprintf("global.removemoderator \"%s\"", id))
	inboundc, err := c.router.Request(ctx, *out)
	if err != nil {
		return fmt.Errorf("error writing remove-moderator; %w", err)
	}
	defer c.router.CloseRoute(out.Identifier)

	in, err := c.waitForInbound(ctx, inboundc)
	if err != nil {
		return fmt.Errorf("error waiting for inbound; %w", err)
	}
	if err := checkInbound(in, out.Identifier); err != nil {
		return err
	}
	if in.Message == fmt.Sprintf("User %s isn't a moderator", id) {
		return ErrModeratorDNE
	}
	if in.Message != fmt.Sprintf("Removed Moderator: %s", id) {
		return errUnexpectedInboundMessage
	}
	return nil
}

// AddOwner adds the owner specified by the id to the Rust server.
func (c Client) AddOwner(ctx context.Context, id string) error {
	out := NewOutbound(fmt.Sprintf("global.ownerid \"%s\"", id))
	inboundc, err := c.router.Request(ctx, *out)
	if err != nil {
		return fmt.Errorf("error writing add-owner; %w", err)
	}
	defer c.router.CloseRoute(out.Identifier)

	in, err := c.waitForInbound(ctx, inboundc)
	if err != nil {
		return fmt.Errorf("error waiting for inbound; %w", err)
	}

	if err := checkInbound(in, out.Identifier); err != nil {
		return err
	}
	if in.Message == fmt.Sprintf("User %s is already a Owner", id) {
		return ErrOwnerExists
	}
	if in.Message != fmt.Sprintf("Added owner unnamed, steamid %s", id) {
		return errUnexpectedInboundMessage
	}

	return nil
}

// RemoveOwner removes the owner specified by the id from the Rust
// server.
func (c Client) RemoveOwner(ctx context.Context, id string) error {
	out := NewOutbound(fmt.Sprintf("global.removeowner \"%s\"", id))
	inboundc, err := c.router.Request(ctx, *out)
	if err != nil {
		return fmt.Errorf("error writing remove-owner; %w", err)
	}
	defer c.router.CloseRoute(out.Identifier)

	in, err := c.waitForInbound(ctx, inboundc)
	if err != nil {
		return fmt.Errorf("error waiting for inbound; %w", err)
	}
	if err := checkInbound(in, out.Identifier); err != nil {
		return err
	}
	if in.Message == fmt.Sprintf("User %s isn't a owner", id) {
		return ErrOwnerDNE
	}
	if in.Message != fmt.Sprintf("Removed Owner: %s", id) {
		return errUnexpectedInboundMessage
	}
	return nil
}

// GrantPermission grants the passed permission to the specified steam ID.
func (c Client) GrantPermission(
	ctx context.Context,
	steamID, permission string,
) error {
	out := NewOutbound(fmt.Sprintf("oxide.grant user %s %s", steamID, permission))
	inboundc, err := c.router.Request(ctx, *out)
	if err != nil {
		return fmt.Errorf(
			"error granting permission \"%s\" to %s; %w",
			permission,
			steamID,
			err,
		)
	}
	defer c.router.CloseRoute(out.Identifier)

	in, err := c.waitForInbound(ctx, inboundc)
	if err != nil {
		return fmt.Errorf("error waiting for inbound; %w", err)
	}
	if err := checkInbound(in, out.Identifier); err != nil {
		return err
	}
	if in.Message == fmt.Sprintf(
		"Player '%s' already has permission '%s'",
		steamID,
		permission,
	) {
		return ErrPermissionAlreadyGranted
	}
	if in.Message != fmt.Sprintf(
		"Player '%s (%s)' granted permission '%s'",
		steamID,
		steamID,
		permission,
	) {
		return fmt.Errorf("%w: \"%s\"", errUnexpectedInboundMessage, in.Message)
	}

	return nil
}

// RevokePermission revokes the passed permission from the specified steam ID.
func (c Client) RevokePermission(
	ctx context.Context,
	steamID, permission string,
) error {
	out := NewOutbound(fmt.Sprintf("oxide.revoke user %s %s", steamID, permission))
	inboundc, err := c.router.Request(ctx, *out)
	if err != nil {
		return fmt.Errorf(
			"error revoking permission \"%s\" to %s; %w",
			permission,
			steamID,
			err,
		)
	}
	defer c.router.CloseRoute(out.Identifier)

	in, err := c.waitForInbound(ctx, inboundc)
	if err != nil {
		return fmt.Errorf("error waiting for inbound; %w", err)
	}
	if err := checkInbound(in, out.Identifier); err != nil {
		return err
	}

	if in.Message != fmt.Sprintf(
		"Player '%s (%s)' revoked permission '%s'",
		steamID,
		steamID,
		permission,
	) {
		return fmt.Errorf("%w: \"%s\"", errUnexpectedInboundMessage, in.Message)
	}
	return nil
}

// CreateGroup creates the passed Oxide group.
func (c Client) CreateGroup(ctx context.Context, group string) error {
	out := NewOutbound(fmt.Sprintf("oxide.group add %s", group))
	inboundc, err := c.router.Request(ctx, *out)
	if err != nil {
		return fmt.Errorf(
			"error creating group \"%s\"; %w",
			group,
			err,
		)
	}
	defer c.router.CloseRoute(out.Identifier)

	in, err := c.waitForInbound(ctx, inboundc)
	if err != nil {
		return fmt.Errorf("error waiting for inbound; %w", err)
	}
	if err := checkInbound(in, out.Identifier); err != nil {
		return err
	}

	if in.Message != fmt.Sprintf("Group '%s' created", group) {
		return fmt.Errorf("%w: \"%s\"", errUnexpectedInboundMessage, in.Message)
	}
	return nil
}

// AddToGroup adds the passed steamID to the passed Oxide group.
func (c Client) AddToGroup(ctx context.Context, steamID, group string) error {
	out := NewOutbound(fmt.Sprintf("oxide.usergroup add %s %s", steamID, group))
	inboundc, err := c.router.Request(ctx, *out)
	if err != nil {
		return fmt.Errorf(
			"error adding %s to group \"%s\"; %w",
			steamID,
			group,
			err,
		)
	}
	defer c.router.CloseRoute(out.Identifier)

	in, err := c.waitForInbound(ctx, inboundc)
	if err != nil {
		return fmt.Errorf("error waiting for inbound; %w", err)
	}
	if err := checkInbound(in, out.Identifier); err != nil {
		return err
	}

	if in.Message != fmt.Sprintf(
		"Player '%s' added to group: %s",
		steamID,
		group,
	) {
		return fmt.Errorf("%w: \"%s\"", errUnexpectedInboundMessage, in.Message)
	}

	return nil
}

// NewOutbound is a constructor for the Outbound type. Typically used to
// initialize the Outbound type with default values and a unique Message field.
func NewOutbound(msg string) *Outbound {
	return &Outbound{
		Identifier: rand.Intn(math.MaxInt32),
		Message:    msg,
		Name:       "WebRcon",
	}
}

// Outbound represents an outbound message destined for the Rcon server.
type Outbound struct {
	Identifier int
	Message    string
	Name       string
}

// Outbound represents an inbound message destined for the Rcon client.
type Inbound struct {
	Identifier int
	Message    string
	Type       string
	Stacktrace string
}

// --- private ---

const (
	// writeWait is the time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// pongWait is the time allowed between pong messages.
	pongWait = time.Minute

	// pingPeriod is the time allowed between ping messages.
	pingPeriod = (pongWait * 9) / 10

	// maxMessageSize allowed from peer.
	maxMessageSize = 4096
)

func (c Client) writePump() error {
	t := time.NewTicker(pingPeriod)
	defer func() {
		t.Stop()
		c.Close()
	}()
	for {
		select {
		case out, ok := <-c.router.Outboundc():
			if !ok {
				if err := c.write(websocket.CloseMessage, []byte{}); err != nil {
					return err
				}
				close(c.closed)
				return nil
			}

			b, err := json.Marshal(out)
			if err != nil {
				return err
			}

			if err := c.write(websocket.TextMessage, b); err != nil {
				return err
			}

		case <-t.C:
			if err := c.write(websocket.PingMessage, nil); err != nil {
				return err
			}
		}
	}
}

func (c Client) write(messageType int, b []byte) error {
	if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
		return err
	}
	return c.conn.WriteMessage(messageType, b)
}

func (c Client) readPump() error {
	defer func() {
		c.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	if err := c.conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		return fmt.Errorf("error setting websocket read deadline; %w", err)
	}
	for {
		_, b, err := c.conn.ReadMessage()
		select {
		case _, ok := <-c.closed:
			if !ok {
				return nil
			}
		default:
		}
		if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
			return fmt.Errorf("unexpected websocket connection closure; %w", err)
		}
		if err != nil {
			return fmt.Errorf("error reading websocket message; %w", err)
		}

		var inbound Inbound
		if err := json.Unmarshal(b, &inbound); err != nil {
			c.logger.Error("unable to unmarshal inbound websocket message", zap.Error(err))
		}

		err = c.router.Injest(c.closed, inbound)
		if err != nil && !errors.Is(err, ErrRoutingIdentifier) {
			c.logger.Error("error injesting inbound message", zap.Error(err))
		}
	}
}

var errRouteClosed = errors.New("rcon inbound route closed unexpectedly")

func (c Client) waitForInbound(ctx context.Context, inboundc chan Inbound) (*Inbound, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-c.closed:
		return nil, errRconClientUnexpectedClose
	case in, ok := <-inboundc:
		if !ok {
			return nil, errRouteClosed
		}
		return &in, nil
	}
}

// --- helpers ---

func checkInbound(in *Inbound, expid int) error {
	if in.Identifier != expid {
		return errIdentifiersNotEqual
	}
	if in.Type != "Generic" {
		return errInboundTypeUnexpected
	}
	return nil
}
