package room

import (
	"context"
	"errors"

	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// Inbound carries the authenticated connection beside a decoded request to a
// non-Duel actor. The actor still checks current membership and authorization.
// No request can supply or replace Client.
type Inbound struct {
	Client *Client
	Frame  wire.ClientMessage
}

func protocolGate(c *Client, kind string) error {
	if c != nil && c.ProtocolVersion != wire.PlatformProtocolVersion && kind != Duel {
		return ErrProtocol
	}
	return nil
}

// CreatePlatform has a closed creation table. A request cannot select a codec,
// constructor or package. Production registers the lobby kind, whose Create
// refuses until -haftgane-tables allows it.
func (m *Manager) CreatePlatform(ctx context.Context, c *Client, req wire.LobbyCreate) (string, error) {
	return m.createKind(ctx, c, "lobby", req)
}

func (m *Manager) createKind(ctx context.Context, c *Client, id string, req wire.ClientMessage) (string, error) {
	err := protocolGate(c, id)
	var code string
	if err == nil {
		kind, ok := m.cfg.Kinds.entries[id]
		if !ok {
			err = ErrInvalidSpec
		} else {
			code, err = kind.Create(m, c, req)
		}
	}
	if err == nil {
		c.Out.Push(wire.NewCreated(code))
		return code, m.Join(ctx, c, code)
	}
	c.Out.Push(wire.NewError(errorCode(err), errorMsg(err)))
	return "", err
}

// DispatchPlatform routes only the decoded family's requests to a joined actor
// of that actual kind. Duel never receives table/lobby messages.
func (m *Manager) DispatchPlatform(ctx context.Context, c *Client, msg wire.ClientMessage) error {
	family, code, seq, ok := wire.PlatformTarget(msg)
	if !ok {
		return m.requestFailed(c, 0, ErrInvalidSpec)
	}
	if c.ProtocolVersion != wire.PlatformProtocolVersion {
		c.Out.Push(wire.NewError(wire.ErrorCodeProtocol, errorMsg(ErrProtocol)))
		return ErrProtocol
	}
	h, err := m.lookup(code)
	if err == nil && (c.joined != code || h == nil) {
		err = ErrNotJoined
	}
	if err == nil && family != "" && h.kind != family {
		err = ErrInvalidSpec
	}
	if err == nil {
		// Common links use the actual room's decoder bounds. The historical Duel
		// actor indexes a two-seat array, so never pass a seven-seat value to it.
		if h.duel != nil {
			switch req := msg.(type) {
			case wire.PlatformLink:
				if req.Seat < 0 || req.Seat > 1 {
					err = ErrInvalidSpec
				} else {
					return m.Link(ctx, c, wire.NewLink(code, req.Seat))
				}
			case wire.PlatformRelink:
				if req.Seat < 0 || req.Seat > 1 {
					err = ErrInvalidSpec
				} else {
					return m.Relink(ctx, c, wire.NewRelink(code, req.Seat))
				}
			default:
				err = ErrInvalidSpec
			}
		} else {
			err = h.post(ctx, Inbound{Client: c, Frame: msg})
		}
	}
	switch {
	case err == nil, errors.Is(err, ErrClosed), isContextErr(err):
	case family == "haftgane":
		c.Out.Push(wire.NewTableReject(seq, "bad-request"))
	case family == "lobby":
		c.Out.Push(wire.NewLobbyReject(seq, "bad-request"))
	default:
		return m.requestFailed(c, seq, err)
	}
	return err
}

// foreignJoined recognizes common operations for a registered non-Duel actor.
// It never loads a room or bypasses Join's protocol and admission checks.
func (m *Manager) foreignJoined(c *Client, code string) *Handle {
	if code == "" || c.joined != code {
		return nil
	}
	h, _ := m.lookup(code)
	if h != nil && h.duel == nil {
		return h
	}
	return nil
}
func (m *Manager) commonPlatform(ctx context.Context, c *Client, h *Handle, msg wire.ClientMessage) error {
	err := protocolGate(c, h.kind)
	if err != nil {
		c.Out.Push(wire.NewError(wire.ErrorCodeProtocol, errorMsg(err)))
		return err
	}
	return m.requestFailed(c, 0, h.post(ctx, Inbound{Client: c, Frame: msg}))
}
