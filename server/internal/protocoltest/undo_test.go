package protocoltest

import (
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// TestUndoHotseat checks that undo in a hot-seat room reverts exactly one
// human decision: the seat to move and the legal choices return to what
// they were right before it. The version itself never goes back down
// (room/doc.go: "the room version ... goes up by one with every accepted
// act and undo"), so it is not part of what "reverted" means here.
func TestUndoHotseat(t *testing.T) {
	srv := newServer(t, t.TempDir(), 9_600_000_000_001)
	c := dial(t, srv, "Hotseat")
	c.hello()
	c.create(wire.ModeHotseat, nil, wire.SeatNumber(0))

	if c.latest.CanUndo {
		t.Fatalf("a brand new room already allows undo")
	}
	before := c.latest
	toMove := *before.View.ToMove
	key := before.View.Choices[0].Key

	seq := c.act(key, seatInt(toMove))
	after := c.untilAck(seq)
	if !after.CanUndo {
		t.Fatalf("after one decision, CanUndo = false, want true")
	}
	if after.Version != before.Version+1 {
		t.Fatalf("after one decision, version = %d, want %d", after.Version, before.Version+1)
	}

	useq := c.undo()
	reverted := c.untilAck(useq)
	if reverted.Version != after.Version+1 {
		t.Fatalf("after undo, version = %d, want %d (undo also goes up by one; it never goes back down)", reverted.Version, after.Version+1)
	}
	if reverted.View.ToMove == nil || *reverted.View.ToMove != toMove {
		t.Fatalf("after undo, toMove = %v, want %d", reverted.View.ToMove, toMove)
	}
	if len(reverted.View.Choices) != len(before.View.Choices) {
		t.Fatalf("after undo, %d choices, want %d (the same position as before the decision)", len(reverted.View.Choices), len(before.View.Choices))
	}
}

// TestUndoAI checks that undo in a room with a bot removes the bot's
// automatic reply along with the human decision that provoked it: one
// undo returns all the way to before the human's move (room/doc.go). It
// looks for a turn where the bot genuinely replies (an extra turn can keep
// the decision with the human instead) rather than assuming the first one
// does.
func TestUndoAI(t *testing.T) {
	srv := newServer(t, t.TempDir(), 9_700_000_000_001)
	c := dial(t, srv, "AI-human")
	c.hello()
	bot := wire.BotLevelEasy
	c.create(wire.ModeAI, &bot, wire.SeatNumber(0))

	var before wire.State
	found := false
	for attempt := 0; attempt < 30 && !over(c); attempt++ {
		for !over(c) && !c.latest.You.CanAct {
			c.untilState(c.latest.Version + 1)
		}
		if over(c) {
			break
		}
		cand := c.latest
		seq := c.act(cand.View.Choices[0].Key, nil)
		c.untilAck(seq)
		for !over(c) && !c.latest.You.CanAct {
			c.untilState(c.latest.Version + 1)
		}
		if over(c) {
			break
		}
		if c.latest.Version > cand.Version+1 {
			// More than just the human's own move was applied: at least one
			// bot reply happened in between.
			before = cand
			found = true
			break
		}
	}
	if !found {
		t.Skip("could not find a human turn the bot actually replied to before the game ended")
	}
	if !c.latest.CanUndo {
		t.Fatalf("after a human decision and the bot's reply, CanUndo = false, want true")
	}

	useq := c.undo()
	reverted := c.untilAck(useq)
	if reverted.View.ToMove == nil || *reverted.View.ToMove != *before.View.ToMove {
		t.Fatalf("after undo, toMove = %v, want %d (back to the human's decision, not the bot's reply)", reverted.View.ToMove, *before.View.ToMove)
	}
	if len(reverted.View.Choices) != len(before.View.Choices) {
		t.Fatalf("after undo, %d choices, want %d (the same position as before the human's move)", len(reverted.View.Choices), len(before.View.Choices))
	}
}
