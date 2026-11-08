package main

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// playHaftgane uses only production lobby and table messages. The driver
// controls one human at a seven-seat table; every bot gets the chosen full
// preset. It reads the legal offer and never computes a move or cost.
func playHaftgane(ctx context.Context, wsURL string, seed uint64, level wire.BotLevel, index int) (gameSummary, error) {
	start := time.Now()
	c, err := dialVersion(ctx, wsURL, "capacity", fakeAddr(index, 0), wire.PlatformProtocolVersion)
	if err != nil {
		return gameSummary{}, err
	}
	defer c.close()
	seating, sides := "lobby", "choose"
	if err = c.send(ctx, wire.LobbyCreate{T: "lobby-create", SeatCount: 7, Options: wire.LobbyOptions{Seating: &seating, Sides: &sides}}); err != nil {
		return gameSummary{}, err
	}
	waitLobby := func(seq int) (wire.LobbyState, error) {
		for {
			f, e := c.next(ctx)
			if e != nil {
				return wire.LobbyState{}, e
			}
			switch v := f.(type) {
			case wire.LobbyState:
				if seq == 0 || v.Ack != nil && v.Ack.Seq == seq {
					return v, nil
				}
			case wire.LobbyReject:
				return wire.LobbyState{}, fmt.Errorf("lobby rejected: %s", v.Code)
			case wire.Error:
				return wire.LobbyState{}, fmt.Errorf("lobby error: %s", v.Code)
			}
		}
	}
	lobby, err := waitLobby(0)
	if err != nil {
		return gameSummary{}, err
	}
	c.code = lobby.Code
	bot := string(level)
	for seat := 1; seat < 7; seat++ {
		c.seq++
		if err = c.send(ctx, wire.LobbyEdit{T: "lobby-edit", Code: c.code, Seq: c.seq, Op: "add-bot", Seat: &seat, Bot: &bot}); err != nil {
			return gameSummary{}, err
		}
		if _, err = waitLobby(c.seq); err != nil {
			return gameSummary{}, err
		}
	}
	c.seq++
	if err = c.send(ctx, wire.LobbyStart{T: "lobby-start", Code: c.code, Seq: c.seq}); err != nil {
		return gameSummary{}, err
	}
	sentPoint := ""
	for {
		f, e := c.next(ctx)
		if e != nil {
			return gameSummary{}, e
		}
		switch v := f.(type) {
		case wire.TableReject:
			return gameSummary{}, fmt.Errorf("table rejected: %s", v.Code)
		case wire.LobbyReject:
			return gameSummary{}, fmt.Errorf("start rejected: %s", v.Code)
		case wire.Error:
			return gameSummary{}, fmt.Errorf("table error: %s", v.Code)
		case wire.TableState:
			if v.Status == "finished" {
				if v.View.Result == nil {
					return gameSummary{}, fmt.Errorf("finished without result")
				}
				return gameSummary{mode: "haftgane", code: c.code, seed: seed, plies: v.Batch, victory: "civilian", winner: "ranked", elapsed: time.Since(start)}, nil
			}
			if v.Point == nil || v.Point.Ref.ID == sentPoint || v.You.Seat == nil || v.Pending != nil || v.View.You == nil || len(v.View.You.Choices) == 0 || !slices.Contains(v.Point.Required, *v.You.Seat) {
				continue
			}
			c.seq++
			sentPoint = v.Point.Ref.ID
			if err = c.send(ctx, wire.TablePick{T: "table-pick", Code: c.code, Seq: c.seq, Point: v.Point.Ref, Key: v.View.You.Choices[0].Key, Confirm: true, Seat: v.You.Seat}); err != nil {
				return gameSummary{}, err
			}
		}
	}
}
