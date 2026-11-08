package tableroom_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"

	"github.com/NightMachinery/ajayeb/server/internal/ai"
	"github.com/NightMachinery/ajayeb/server/internal/room"
	"github.com/NightMachinery/ajayeb/server/internal/tableroom"
	"github.com/NightMachinery/ajayeb/server/internal/view"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

type capacitySamples struct {
	mu        sync.Mutex
	values    map[string][]int64
	fallbacks map[string]int
}

func (m *capacitySamples) add(k string, n int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.values[k] = append(m.values[k], n)
}
func (m *capacitySamples) report(r tableroom.BotReport) {
	if r.Cancelled {
		m.add("cancelled", 1)
		return
	}
	kind := "turn"
	for _, k := range []string{"sides", "discard-build", "last-card"} {
		if strings.Contains(r.Point, k) {
			kind = k
		}
	}
	m.add("active_ns."+kind, int64(r.Active))
	m.add("pool_wait_ns."+kind, int64(r.PoolWait))
	m.add("pool_hold_ns."+kind, int64(r.PoolHold))
	m.add("room_work_wait_ns."+kind, int64(r.RoomWorkWait))
	if r.Fallback != "" {
		m.mu.Lock()
		m.fallbacks[r.Fallback]++
		m.mu.Unlock()
	}
}
func (m *capacitySamples) log(t *testing.T, label string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := map[string]any{}
	for k, v := range m.values {
		slices.Sort(v)
		q := func(p int) int64 { return v[max(0, (len(v)*p+99)/100-1)] }
		out[k] = map[string]any{"n": len(v), "p50": q(50), "p95": q(95), "p99": q(99), "max": v[len(v)-1]}
	}
	b, err := json.Marshal(map[string]any{"case": label, "metrics": out, "fallbacks": m.fallbacks})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("capacity-ws %s", b)
	if len(m.fallbacks) != 0 {
		t.Fatal("ordinary fallback")
	}
}

// Actual loopback WebSockets, production codecs and full Scholar work. A cold
// mid-game restart preserves the pending pick. Burst spectators read normally;
// four extra TCP clients stop reading entirely after hello and join. The latter
// tests non-reading peers, without claiming to force TCP buffer exhaustion.
func TestCapacityWebSocketEvidence(t *testing.T) {
	if os.Getenv("HAFTGANE_CAPACITY") != "1" {
		t.Skip("opt-in host capacity")
	}
	for _, scenario := range []string{"solo-restore", "spectator-burst-slow", "all-bot-continuation"} {
		t.Run(scenario, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()
			m := &capacitySamples{values: map[string][]int64{}, fallbacks: map[string]int{}}
			cfg := tableroom.Config{Creation: true, Pool: ai.NewPool(2), RoomWork: ai.NewPool(1), BotFinished: m.report}
			dir := t.TempDir()
			s := newStackConfig(t, dir, cfg)
			mixed := os.Getenv("HAFTGANE_CAPACITY_MIXED") == "1"
			stopLoad := func() {}
			if mixed {
				stopLoad = capacityDuelLoad(t, s, m)
			}
			defer func() { stopLoad() }()
			token := newTokens(t, 1)[0]
			c := s.dial(ctx, token, "capacity-human", "Player")
			seed := uint64(942)
			sp := tableroom.Spec{Seed: &seed, Creator: c.user, Sides: "choose", Participants: []tableroom.Participant{{UserID: c.user, Name: "Player"}}}
			for range 6 {
				sp.Participants = append(sp.Participants, tableroom.Participant{Bot: tableroom.BotNormal})
			}
			code, err := s.tables.CreateTable(sp)
			if err != nil {
				t.Fatal(err)
			}
			fr := c.join(ctx, code)
			if scenario == "spectator-burst-slow" {
				for i, tok := range newTokens(t, 24) {
					started := time.Now()
					s.dial(ctx, tok, fmt.Sprintf("burst-%d", i), "Watcher").join(ctx, code)
					m.add("spectator_join_ns", time.Since(started).Nanoseconds())
				}
				for i, tok := range newTokens(t, 4) {
					conn, _, err := websocket.Dial(ctx, s.url, nil)
					if err != nil {
						t.Fatal(err)
					}
					t.Cleanup(func() { _ = conn.CloseNow() })
					hello, _ := json.Marshal(wire.NewHello(2, tok, "Slow", fmt.Sprintf("slow-%d", i)))
					if err = conn.Write(ctx, websocket.MessageText, hello); err != nil {
						t.Fatal(err)
					}
					if _, _, err = conn.Read(ctx); err != nil {
						t.Fatal(err)
					}
					join, _ := json.Marshal(wire.NewJoin(code))
					if err = conn.Write(ctx, websocket.MessageText, join); err != nil {
						t.Fatal(err)
					}
				}
			}
			if scenario == "all-bot-continuation" {
				c.send(ctx, wire.TableResign{T: "table-resign", Code: code, Seq: 1, Confirm: true})
			}
			seq, sentPoint, batch := 10, "", 0
			var sent time.Time
			restored := false
			for {
				if !sent.IsZero() && fr.Batch > batch {
					m.add("human_pick_to_reveal_ns", time.Since(sent).Nanoseconds())
					sent = time.Time{}
				}
				batch = fr.Batch
				var mem runtime.MemStats
				runtime.ReadMemStats(&mem)
				m.add("heap_bytes", int64(mem.HeapAlloc))
				if fr.Status == "finished" {
					if fr.View.Result == nil {
						t.Fatal("missing result")
					}
					data, err := json.Marshal(fr)
					if err != nil {
						t.Fatal(err)
					}
					m.add("end_frame_bytes", int64(len(data)))
					break
				}
				if scenario != "all-bot-continuation" && fr.Point != nil && fr.Point.Ref.ID != sentPoint && fr.You.Seat != nil && slices.Contains(fr.Point.Required, *fr.You.Seat) && fr.Pending == nil && fr.View.You != nil && len(fr.View.You.Choices) > 0 {
					seq++
					sentPoint = fr.Point.Ref.ID
					sent = time.Now()
					pick := wire.TablePick{T: "table-pick", Code: code, Seq: seq, Point: fr.Point.Ref, Key: fr.View.You.Choices[0].Key, Confirm: true}
					c.send(ctx, pick)
					if scenario == "solo-restore" && !restored && batch >= 2 {
						ack := c.untilState(ctx, "durable confirmed pick", func(st wire.TableState) bool { return st.Ack != nil && st.Ack.Seq == seq })
						started := time.Now()
						stopLoad()
						s.close()
						s = newStackConfig(t, dir, cfg)
						if mixed {
							stopLoad = capacityDuelLoad(t, s, m)
						}
						c = s.dial(ctx, token, "capacity-human", "Player")
						fr = c.join(ctx, code)
						m.add("cold_restart_join_ns", time.Since(started).Nanoseconds())
						if ack.Pending != nil && ack.Point != nil && fr.Point != nil && ack.Point.Ref == fr.Point.Ref && (fr.Pending == nil || fr.Pending.Key != ack.Pending.Key || fr.Pending.Confirm != ack.Pending.Confirm) {
							t.Fatal("restore lost accepted pick")
						}
						restored = true
						continue
					}
				}
				fr = c.untilState(ctx, "table progress", func(wire.TableState) bool { return true })
			}
			for _, data := range c.raw {
				m.add("wire_frame_bytes", int64(len(data)))
			}
			stopLoad()
			s.close()
			label := scenario
			if mixed {
				label = "mixed-" + label
			}
			m.log(t, label)
		})
	}
}

func capacityDuelLoad(t *testing.T, s *stack, m *capacitySamples) func() {
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	for i := range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c := room.NewClient(fmt.Sprintf("capacity-duel-%d", i), "duel-tab", "Player")
			c.Addr = fmt.Sprintf("capacity-address-%d", i)
			for ctx.Err() == nil {
				level := wire.BotLevelEasy
				req := wire.NewCreate(wire.ModeAI, &level, wire.SeatNumber(0), nil)
				req.Hints = &view.HintSettings{}
				code, err := s.m.Create(ctx, c, req)
				if err != nil {
					if ctx.Err() == nil {
						t.Error(err)
					}
					return
				}
				seq := 0
				pending := false
				finished := false
				var sent time.Time
				for !finished && ctx.Err() == nil {
					for _, msg := range c.Out.Take() {
						st, ok := msg.(wire.State)
						if !ok {
							continue
						}
						if pending && st.Ack != nil && st.Ack.Seq == seq {
							m.add("duel_action_ack_ns", time.Since(sent).Nanoseconds())
							pending = false
						}
						if st.Room.Status == wire.RoomStatusFinished {
							finished = true
							break
						}
						if !pending && st.You.CanAct && len(st.View.Choices) > 0 {
							seq++
							pending = true
							sent = time.Now()
							if err := s.m.Act(ctx, c, wire.NewAct(code, seq, st.Version, st.View.Choices[0].Key, nil)); err != nil {
								if ctx.Err() == nil {
									t.Error(err)
								}
								return
							}
						}
					}
					if finished {
						break
					}
					select {
					case <-c.Out.Ready():
					case <-ctx.Done():
					}
				}
			}
		}()
	}
	return func() { cancel(); wg.Wait() }
}
