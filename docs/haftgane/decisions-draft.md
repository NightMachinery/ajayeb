# Draft decisions for Haftgane

@Opus5.5 [2026-10-21 Wed]. Proposed entries for `docs/dev/decisions.md`, collected from the M12 design documents, merged where two documents proposed the same thing, and numbered from D30 in the file's format. None is accepted yet: each becomes a real entry only when the owner approves the design. Every entry applies to Haftgane only unless it says otherwise, and its last paragraph names the existing decisions it extends or supersedes. Duel keeps D01 to D29 exactly as written.

The owner answered the review questions on 2026-10-21 and named the game Haftgane; the entries record those answers. Two points stay open, and [README.md](README.md) restates both: the shared lobby with Duel (D60, awaiting the owner) and two details of the admin powers in D42 (provisional).

## D30. Haftgane is 7 Wonders, second edition, on the same site in one kind of room

**Decision (draft, 2026-10-21).** Ajayeb's second game, **Haftgane** (Persian هفتگانه, "the seven"; next to Ajayeb, عجایب, the two words make the Persian name for the Seven Wonders of the World), is the base game of 7 Wonders, second edition (2020), for 3 to 7 players. Its game id and room kind are `haftgane`. The owner decided that it lives on the same site and in the same binary as Duel, with a game picker on Home, and that it has one kind of room: 3 to 7 seats, in which the creator may add any number of bots, so a table can be the creator alone with bots or any mix of humans and bots. Spectators work as in Duel. Pass-the-phone (hot-seat) play is deferred. The base game only, decided on 2026-10-21: no expansion is built, and the design keeps only cheap extension points.

**Rationale.** One site keeps one identity, one deployment and one set of room links. A single room kind, with bots added seat by seat, covers playing alone and playing with friends without a mode choice.

**Rejected.** A separate site or binary (two deployments to maintain for one owner). Separate bot and online modes, as Duel has (they contradict "any mix of humans and bots"). Hot-seat for 3 to 7 players in the first version (a pass-the-device flow for up to seven people, deferred by the owner).

**Relation.** Extends D05 (one static SPA served by one binary, now for two games). D01 holds unchanged. D15's bot and hot-seat modes do not exist in this game; D43 replaces them here. Whether its online lobby also starts Duel at two seats awaits the owner (D60).

Source: the owner's answers of 2026-10-21; [platform-architecture.md](platform-architecture.md), summary.

## D31. Room kinds are registered side by side, and Duel's room stays as it is

**Decision (draft, 2026-10-21).** The server registers **room kinds**, each a complete game implementation: a file codec, a room constructor, an inbound frame decoder and a lazy client entry. Duel registers its existing actor, schema-1 codec and client unchanged. Haftgane gets its own actor (`server/internal/tableroom`) typed directly to its engine. The two share the Manager's lifetime bookkeeping, connection queues, claims, clocks, atomic file operations and the bot permit pool through narrow seams. Some orchestration is duplicated at first and is extracted only once both actors have tests that prove it identical.

**Rationale.** Duel is finished and verified against stored games. Generalizing its actor, Session, review, tutorial, hot-seat, undo and bot scheduling before a second caller exists would put all of it at risk for code that Haftgane does not need.

**Rejected.** A universal engine interface or factory with a capability matrix. Two transaction modes inside one actor. Rewriting Duel's orchestration in advance.

**Relation.** New. It constrains how D13, D16 and D28 are implemented for two games (see D41) but changes none of them for Duel.

Source: [platform-architecture.md](platform-architecture.md), "Room registration and package boundaries".

## D32. Haftgane's rules live in their own pure packages; only infrastructure is shared

**Decision (draft, 2026-10-21).** The rules live in `server/internal/haftgane/` (`catalog`, `rules`, `view`, test helpers in `rulestest`, bots in `bot`), pure under AGENTS.md's engine rules, with one stricter rule: no file of the new engine imports `math/rand/v2`. Code that means exactly the same thing in both games moves out of Duel into `server/internal/enginekit/`, each piece just before its first new caller and with Duel's goldens, replays, stored rooms and rules version proven unchanged: the seeded generator and shuffle, the seed mixer, the digest helper, the import and fixed-value shape checkers (test helpers) and the three-way catalog diff (a test helper). The bot permit pool is shared by use, not moved. State, catalog types, effects, payment, military, science, scoring, views, replay helpers, test harnesses and bot search are written per game.

**Rationale.** The same words (production, chains, science, military) name different rules in the two games, and Duel's engine is tied to stored rooms through its hashes. Infrastructure with identical meaning should exist once, so that a fix reaches both games and copies cannot drift. This answers the owner's question about sharing code between the engines.

**Rejected.** Generalizing `server/internal/engine`. A shared rules core with per-game plug-ins. Copying the infrastructure. One package for catalog, rules and DTOs.

**Relation.** Extends D01: Haftgane's rules also run only on the server. The new engine follows D11's determinism through D36.

Source: [engine-architecture.md](engine-architecture.md), sections 1 and 15.

## D33. Card designs are the rules identity; physical copies exist only while decks are built

**Decision (draft, 2026-10-21).** The catalog has one entry per **design** (a card as printed for one Age) and a separate name identity for the one-building-per-name rule. Hands, cities, tucked cards, the discard pile, keys, events, views and observations hold design ids. Physical copies are counted only when the decks for N players are built, so no id, order or key can encode a deal position. Every list of cards a viewer receives is sorted by design id.

**Rationale.** Copies of a design are indistinguishable in play. Copy ids would carry no information, would need a canonical-copy rule for keys, would enlarge what bots sample, and could leak the shuffle.

**Rejected.** Copy ids in the state or on the wire. One design per name (it would blur which Age a card belongs to).

**Relation.** New.

Source: [engine-architecture.md](engine-architecture.md), section 2 and its draft entry; [platform-architecture.md](platform-architecture.md), "Views, events and history".

## D34. A construction key names its payment outcome

**Decision (draft, 2026-10-21).** A construction key says how many coins go to each neighbour: `build:<design>:<left>:<right>` or `stage:<design>:<left>:<right>`. The engine offers exactly the affordable outcomes that no other outcome beats (paying no more to either neighbour and less to one), at most 15 per construction, cheapest first, and a free construction only as free. Buying a unit one's own production covers is never offered. The server accepts only an offered key and applies the engine's own action.

**Rationale.** In Duel the cheapest payment is never worse, so D02 keeps payment out of moves. Here trade coins go to one of two rival neighbours and the rules let the buyer choose, so which rival gets paid is a real decision. Leaving out beaten outcomes is a deliberate restriction (rules.md ruling 10): it keeps the list short and its size proven.

**Rejected.** The engine choosing the cheapest split with a fixed tie-break (removes a real choice). A payment field validated by the server (D02's reasons). One key per way of covering a cost (identical effects). Every outcome, beaten ones included (up to 225 per construction).

**Relation.** Supersedes D02's payment clause for Haftgane only. D02's canonical keys and set membership stay. The owner accepted unbeaten payments only on 2026-10-21.

Source: [engine-architecture.md](engine-architecture.md), sections 7 and 8; [rules.md](rules.md), ruling 10.

## D35. A simultaneous decision resolves as one complete batch from a frozen position

**Decision (draft, 2026-10-21).** The engine defines **decision points**: a set of required seats choosing against one frozen position. Each turn is a point for all seats, checked and priced against the position at its opening and applied in seven fixed steps (choose, check, pay, place, when-placed effects, end-of-turn abilities, pass). The Halikarnassos discard build and Babylon's last card follow as one-seat points, in that order. The room holds one pick per required seat and persists every accepted change privately. A seat may replace its pick or withdraw its confirmation until the last required seat confirms; that confirmation produces one seat-sorted batch, which the engine resolves before one durable write and one grouped reveal, and a change arriving after it is refused. Reload and reconnect restore a seat's pick and its confirmed state, and others see only who has confirmed. Complete batches are the canonical history; arrival order never becomes rules order. The owner chose changeable picks on 2026-10-21.

**Rationale.** This makes simultaneous play exact: a key legal at the opening stays legal at resolution, so there is no conflict resolution, and neither arrival order nor seat order decides anything. The last confirmation closes the point at once, so a late change has nothing to race with: it is refused, and a delayed request never reverts a newer pick because it carries an older seq.

**Rejected.** Applying picks in seat order or as they arrive. Appending every arrival to the rules history. Mutating the engine with partial picks. Playing Babylon's last card inside the sixth turn. Picks locked once confirmed (the provisional default, which the owner declined, because a player could not correct a pick while others were still thinking). A visible count of pick changes (it would show hesitation).

**Relation.** Extends D02 (membership is checked per seat against that seat's frozen legal set) and D11 (a game is its setup plus its complete batches).

Source: [engine-architecture.md](engine-architecture.md), section 6; [platform-architecture.md](platform-architecture.md), "Decision points and acceptance" and "Owner decisions".

## D36. Setup draws from one seeded stream per random element

**Decision (draft, 2026-10-21).** All randomness is still resolved when the game is created. Seating, boards, sides, the guild draw and each Age's deck draw from their own stream, derived from the seed and a fixed tag pinned by a golden, with the shared generator.

**Rationale.** With one stream, an option that skips a draw (chosen seating, chosen sides) would shift every later shuffle, so options would change the deals. Separate streams also let a later element be added without moving the others.

**Rejected.** One stream consumed in order, as in Duel.

**Relation.** Extends D11 for Haftgane. Duel's single stream is unchanged.

Source: [engine-architecture.md](engine-architecture.md), section 5.

## D37. Each viewer's knowledge is derived from the game record and sent by whitelist

**Decision (draft, 2026-10-21).** The engine value is the state plus a fixed-size **record** of every dealt packet, every pick, every public play and every discard build. Views, events, history and bot observations are projections of it, built by whitelist for each seat and for one spectator role, never by deleting fields from a full object. A seat's hand, offered keys, pending pick, tucked and sold cards and remembered knowledge reach only that seat; who has confirmed, and when, is public, and a seat's pick changes are not. Late joiners and resyncs get the same whitelist with a rebuilt history. Decided by the owner on 2026-10-21: each seat's view carries a private **notebook** (the hands it held, what it did with each card, and the pile at its own discard opportunities) with summary statistics drawn only from that seat's knowledge, each certain or an upper bound, so humans know what bots know.

**Rationale.** What a seat legitimately knows is a function of the record, so deriving it keeps views, bots and history consistent, and the whole game stays one comparable, hashable value.

**Rejected.** Knowledge stored per seat (derivable, and about three times the size). The record kept by the room. A shared private hint object. Full-state redaction by deletion. No notebook (less UI, but human and bot memory unequal). Statistics that guess (advice built on assumptions).

**Relation.** Extends D03: still a full per-viewer snapshot after every change, now for up to seven seats and a spectator.

Source: [engine-architecture.md](engine-architecture.md), sections 4 and 10; [platform-architecture.md](platform-architecture.md), "Views, events and history".

## D38. Everything is revealed at the end unless the room hides it, and the engine has no resignation

**Decision (draft, 2026-10-21).** By default every hidden card is shown when the game ends and the log can be downloaded as `ajayeb-log/2` (setup, complete batches and the final fingerprint). A creation option keeps hidden cards hidden after the end, and then no download is offered, because the log's keys name sold and tucked cards. There is no victory before the end of Age III. Resigning hands the seat to a bot; it is a room act, so the engine has no resign key.

**Rationale.** It matches Duel's end policy. The score counts only public things, so a hidden end still shows the full result.

**Rejected.** Always revealing. A resignation or surrender victory in the engine.

**Relation.** Follows D03 and D28 (downloaded logs keep their strict final-hash check). Duel's `ajayeb-log/1` is unchanged.

Source: [engine-architecture.md](engine-architecture.md), sections 10 and 12; [platform-architecture.md](platform-architecture.md), "Persistence, replay and compatibility".

## D39. Wonder sides are a room option

**Decision (draft, 2026-10-21).** By default every player chooses a side after seeing every board, in one simultaneous hidden point before Age I is dealt, and the choices are revealed together. Room options give every seat its day side, or draw sides at random.

**Rationale.** rules.md rulings 16 and 26. A simultaneous hidden choice fits the decision model and removes any advantage from choosing last.

**Rejected.** Open choice in seat order (slower online, and the first seat always chooses blind). Dealing Age I before the choice.

**Relation.** New; Duel's wonder draft is a different mechanism. The owner accepted the simultaneous hidden choice on 2026-10-21.

Source: [engine-architecture.md](engine-architecture.md), section 5; [rules.md](rules.md), rulings 16 and 26.

## D40. The coin reserve is unlimited

**Decision (draft, 2026-10-21).** As in Duel: the reserve never runs out, no payment exceeds the treasure in the frozen position, and the base game takes no coins away.

**Rationale.** rules.md ruling 14: the rulebook sets no limit, and no rule covers an empty reserve.

**Rejected.** A cap at the coins in the box.

**Relation.** Confirms D09 for Haftgane, as a rule of this game rather than an inheritance.

Source: [engine-architecture.md](engine-architecture.md), draft entries.

## D41. Compatibility lives in per-game file codecs and frame families

**Decision (draft, 2026-10-21).** Duel's room files stay on schema 1 with today's codec; Haftgane files use schema 2 with their own codec, an explicit game id (`haftgane`) and their own rules version. A bounded header decides the codec before any decoding or replay, and an unknown schema, codec or game is left untouched and unsupported, never archived as a broken Duel room. Hello accepts protocol versions 1 and 2. Duel's game frames keep one unchanged contract on both; version 2 only adds Haftgane frames (`table-state` and its requests), and a version-1 client that tries to join a Haftgane room gets the existing reload path. New DTOs are generated separately, leaving Duel's generated files byte-identical.

**Rationale.** No Duel room, frame or fixture changes, so the risk to Duel is confined to the dispatch seams, and old binaries already refuse schema 2 through their error count.

**Rejected.** Migrating or normalizing Duel records. Discovery messages before join. A second Duel wire contract. A game field added to Duel's state frame.

**Relation.** Extends D13 and D16 to two games, extends D28's compatibility tuple to (schema and codec, game, game rules version), and keeps D29's archive layout and retention for both games.

Source: [platform-architecture.md](platform-architecture.md), "Persistence, replay and compatibility" and "Protocol and lazy client entry".

## D42. One mixed-seat lobby; Start freezes the table; the admin switches who controls a seat, never the table

**Decision (draft, 2026-10-21).** Creation names the game, the seat count and the options. The creator holds one lobby seat and is the admin; humans who open the link take free seats or spectate; the admin adds and removes bots, edits options and can move a lobby human to the spectators. There are no ready flags: Start is the consent, every seat must be filled, and an explicit Start option fills open seats with bots. Start applies the seating rule and freezes the seat assignment, the starting human count, the undo flag and the bot configuration. A disconnect never changes neighbours or ends the game. The admin may switch any seat between human and bot control at any time: a human replaced by a bot becomes a spectator, and the bot plays the seat's pick, confirmed or not; a spectator seated in a bot's seat takes over its pick, still changeable, and its knowledge. Resignation hands one's own seat to a bot. No switch declares a surrender winner, and every switch shows in the game log for everyone. A rematch is one fresh lobby with seats reserved for the original humans. Seating is random, with the admin able to use the lobby order instead. Decided by the owner on 2026-10-21. Provisional details for the owner to confirm: a former player returns only to their own former seat, and if the admin has been away for two minutes the role passes to the connected human seated longest.

**Rationale.** One lobby serves a solo table and a table of friends alike. Freezing at Start keeps undo, confirmation and bot work from depending on later events. Keeping the table fixed keeps neighbours, and so the rules, stable. The admin, not a timer, decides when an absent player's seat goes to a bot, and can give a seat a bot was keeping to a returning or new human.

**Rejected.** Separate bot and online modes. Readiness revisions. A connected-but-idle deadline. Permanent takeover by any seated human after two minutes (the provisional default, replaced by the owner's admin powers). Seating a former player in another seat (they know their old seat's hands).

**Relation.** New. Reuses D14's identity and claims.

Source: [platform-architecture.md](platform-architecture.md), "Lobby, admin powers and rematch" and "Owner decisions".

## D43. Undo exists only in a game that started with exactly one human seat

**Decision (draft, 2026-10-21; decided by the owner the same day).** A game that started with exactly one human seat allows undo unless its creator turned it off. Undo turns off for the rest of the game as soon as a second human takes a seat, and no later resignation or switch turns it back on. Undo replays from setup to before the latest completed batch with the human's decision, dropping later batches and the pending block; it never changes a seat's controller or rerolls the deal. Undo after a reveal shows the bots' simultaneous picks, and deterministic bots repeat them.

**Rationale.** A solo player can correct a mistake without any other human's information being rolled back. Peeking at bot replies through undo is the same accepted trade as D15's in Duel's bot rooms.

**Rejected.** No undo in Haftgane (prevents peeking but removes solo correction). Undo in games with several humans.

**Relation.** Extends D15 to Haftgane, where "bot room" means "started with one human seat".

Source: [platform-architecture.md](platform-architecture.md), "Owner decisions"; [bot-plan.md](bot-plan.md).

## D44. Every bot level gets the same honest information, and samples hidden cards only within proven bounds

**Decision (draft, 2026-10-21).** Bots never receive the game. Each gets the engine's `Observation`: the public position, its own hand and its legitimately remembered hands and pile looks, with a canonical encoding. Difficulty changes computation and noise, never access. The stronger levels sample hidden cards with the engine's `Determinize`, which fills every hidden place consistently with the whole observed history (including skipped Halikarnassos abilities), counts its work against a declared bound, and returns no world rather than an inconsistent one; the bot then falls back to Greedy. Policy randomness comes from the game id, policy version, logical point, seat and the observation's fingerprint, under a domain label separate from setup; never from the full state, pending keys, wall clock or arrival order.

**Rationale.** One engine-owned contract keeps what bots infer consistent with views and replay, and a bounded failure keeps play honest. Observation-derived seeds make the same knowledge give the same choice, which is the fairness test.

**Rejected.** Real hidden state with random mistakes. A second sampler in the bot package. Unbounded retries. Seeds from the full-state hash.

**Relation.** Applies D01's no-cheating principle to bots and carries M3's fairness test (same public information, same choice) to seven seats.

Source: [bot-plan.md](bot-plan.md), "Honest information and bounded sampling"; [engine-architecture.md](engine-architecture.md), section 11; [platform-architecture.md](platform-architecture.md), "Bots: information, work and lifecycle".

## D45. Each bot gets fixed work per decision at its difficulty, through the existing permit pool

**Decision (draft, 2026-10-21).** Each difficulty has a fixed, deterministic iteration allowance per bot decision under a versioned preset. Table size, bot count, queue pressure and the order of other picks never reduce it. Before a bot searches, the room persists that seat's work row (policy version, preset, allowance); rows and accepted picks survive restart and are never redistributed, and a bot switched into a seat gets a full row. At most one bot job per room is queued or active, through `ai.Pool.AcquireFor` and `ReleaseFor`, released after each pick. Decided by the owner on 2026-10-21: equal work per bot at every table size, so a full table is slower rather than weaker.

**Rationale.** Fixed work keeps a chosen difficulty meaning the same thing when the creator adds bots. The existing pool already gives fair admission between rooms, and durable rows stop a restart or a seat switch from silently changing a pending choice.

**Rejected.** Dividing one room allowance among its bots (faster turns, weaker bots at full tables). Time-limited strength as normal policy. Simultaneous searches within a room. A new global CPU and memory scheduler. Cheaper work for a bot switched into a seat.

**Relation.** New. Duel's budgets, presets, pool settings and review lane are unchanged.

Source: [bot-plan.md](bot-plan.md), "Fixed work, existing permits and frozen rows"; [platform-architecture.md](platform-architecture.md), "Owner decisions".

## D46. Bot levels compare root moves by simulating whole simultaneous rounds

**Decision (draft, 2026-10-21).** The levels are Apprentice (Greedy with softmax noise), Scholar (paired root comparisons that simulate every seat's simultaneous picks to the end of the current Age) and Master (the same with full-game rollouts). Random and deterministic Greedy are simulator baselines, and Greedy is also the legal fallback. A simulated round gathers every required choice from one frozen position before resolving it; opponents are independently self-interested policies drawn from a small mixture of strategies. Rewards blend rank and win credit. The owner chose to wait for Scholar: Greedy and Apprentice first complete an internal milestone, the public release offers Apprentice and Scholar, and Master may follow later. Persona names per level are decided.

**Rationale.** Paired root comparison has bounded storage and a clear information boundary, and the Age versus full-game horizon gives the levels a real difference.

**Rejected.** Duel's alternating minimax and two-player backup. A coalition of opponents. Enumerating joint actions. A simultaneous information-set tree before it proves itself at equal work.

**Relation.** New. Duel's bots and their names are unchanged; Haftgane reuses the persona names with its own presets.

Source: [bot-plan.md](bot-plan.md), "Families and simultaneous search".

## D47. The bots' evaluation is written for this game's scoring and economy

**Decision (draft, 2026-10-21).** Rules, legal payments and exact scoring stay in the engine. The bot adds transparent future-value estimates in victory-point equivalents: per-neighbour military forecasts, science squares and sets, production and trade with both neighbours, chains, wonder progress, guild scopes, and capped passing and denial terms. Weights, temperature, mixture, horizon and work bounds freeze together as one bot-policy version, separate from the rules version.

**Rationale.** Two neighbours, nonlinear science and directional drafting need different incentives from Duel's.

**Rejected.** Duel's weights. Linear shield rewards. Treating all production as tradeable. Tuning on the release seeds.

**Relation.** New.

Source: [bot-plan.md](bot-plan.md), "Evaluate this game's scoring and economy".

## D48. Bot strength is measured in seat-balanced deal blocks at the live presets

**Decision (draft, 2026-10-21).** `ajayeb-sim` gains game-selected N-player tournaments, with Duel's commands and output unchanged. A **deal block** fixes the decks, guilds, sides and seats; the focal policy rotates through every seat, results are reported by player count, seat and side, and 95% intervals keep a block's rotations together. Release targets are declared in advance (for Apprentice and Greedy against Random, Scholar against Greedy, Master against Scholar) and are measured at the same fixed per-bot presets as live play.

**Rationale.** Neighbours and passing direction can reverse rankings, so paired deal blocks reduce noise without pretending rotated games are independent.

**Rejected.** Two-seat swaps as the only pairing. Raw score alone. Reduced-budget proxies standing in for live settings.

**Relation.** Follows M3's practice of predeclared targets; no existing decision changes.

Source: [bot-plan.md](bot-plan.md), "Seat-balanced strength and privacy evaluation".

## D49. Haftgane capacity limits come from measurements on the host

**Decision (draft, 2026-10-21).** Existing room, client and queue limits stay. The only new limit is one configurable maximum of Haftgane rooms doing bot work at once, set from host measurements of the full seven-seat, six-bot case, mixed with Duel load. No room cap, memory reservation, payload ceiling or merge timer is invented before measurement.

**Rationale.** Under the owner's decision a small fixed room cap would refuse the third person who opens a room on the whole site; numbers chosen before measurement are guesses.

**Rejected.** A two-room cap, per-room memory reservations and fixed frame or file ceilings.

**Relation.** New. Duel's limits are unchanged.

Source: [platform-architecture.md](platform-architecture.md), "Capacity and acceptance".

## D50. Haftgane ships without hints or an advisor

**Decision (draft, 2026-10-21).** The first release of Haftgane has no hint families and no post-move advisor. If they come later, the admin owns their settings as in D23, their absence never blocks play, and anything that depends on a hand is computed inside that seat's projection from its permitted information. A future advisor reviews a decision only after its batch resolves, privately to that human.

**Rationale.** Honest playable bots and measured full-table performance come first. Duel's shared hint object cannot carry private hands.

**Rejected.** Enabling Duel's hint ids for Haftgane. Pre-pick move suggestions. Advice that uses the true deal.

**Relation.** Extends D23 (deferred for this game; ownership by the admin kept for later). Duel's hints and advisor are unchanged.

Source: [bot-plan.md](bot-plan.md), "Defer hints and private post-round advice"; [platform-architecture.md](platform-architecture.md), "Views, events and history".

## D51. Duel's client and the service worker stay as they are; Haftgane loads lazily

**Decision (draft, 2026-10-21).** The client picks a lazy game entry from the authoritative snapshot. A generic `/play/<code>` entry runs its hello and join only after the existing name prompt, and hands a Duel room to the unchanged Duel Session through its ordinary version-1 connection. The service worker keeps precaching the whole build, so a Duel player downloads changed Haftgane code once in the background per build; art stays out of the precache (D24). Cold Home stays within 200,000 gzip bytes, and cold Home and Duel entry stay within their baseline plus 5,000 gzip bytes, measured with service workers blocked; import checks prove that Home and Duel never execute Haftgane catalogs or renderers.

**Rationale.** It keeps Duel's Session and cache behaviour untouched and protects an open page's lazy chunks across a deploy. The cost is one extra round trip on generic Duel entry and some background bytes.

**Rejected.** Generalizing Duel's Session. Selective precache before install-transfer evidence exists.

**Relation.** Keeps D05 and D24 as they are; adds the two cold-entry budgets.

Source: [platform-architecture.md](platform-architecture.md), "Protocol and lazy client entry" and "Capacity and acceptance".

## D52. The seat ring is drawn left to right in every language

**Decision (draft, 2026-10-21).** The left neighbour is always on the screen's left, and the other seats run across the top from left to right, also in right-to-left languages. The page chrome, panel contents and sheets follow the page direction; signed numbers are isolated and formatted with `Intl`.

**Rationale.** Left and right are rules terms that name real table directions (passing, the trading posts). Mirroring would put the left neighbour on the right.

**Rejected.** Mirroring the ring. Relabelling neighbours "first" and "second".

**Relation.** Extends D10 and the existing convention that Duel's board region is left to right.

Source: [phone-layout.md](phone-layout.md), draft entries.

## D53. The hand never scrolls

**Decision (draft, 2026-10-21).** The whole hand is always visible as a grid sized by its container: four columns by two rows on phones, one row of seven in short landscape.

**Rationale.** Every card's cost and affordability must be comparable at a glance, and the board never scrolls sideways.

**Rejected.** A scrolling row. A fan. List rows (kept only as a fallback for very small screens).

**Relation.** New.

Source: [phone-layout.md](phone-layout.md), draft entries.

## D54. Each turn ends with one grouped reveal and a recap

**Decision (draft, 2026-10-21).** When the last confirmation arrives, every seat is tagged with what it did at the same moment, and a recap of the turn opens by itself and closes on any tap; a button reopens it during the next turn. There are no per-seat or per-bot pauses. With Animation off the tags and the new hand appear at once and the recap still opens.

**Rationale.** Simultaneous turns should feel simultaneous; with up to six bots a sequence of pauses would add seconds to every turn; a recap makes six actions readable on a phone.

**Rejected.** Sequential reveals per seat. A recap only on request (kept as a setting, as the owner accepted).

**Relation.** Follows D21: the recap carries information, so it survives Animation off.

Source: [phone-layout.md](phone-layout.md), draft entries; [platform-architecture.md](platform-architecture.md), "Bots: information, work and lifecycle".

## D55. Neighbour panels show server-computed trade prices and the military outlook

**Decision (draft, 2026-10-21).** Each seat's view includes, for each neighbour, the units that seat may buy and its price per unit after its discounts, with the discount's source, and the conflict token it would take against that neighbour if the Age ended now. Running scores for every seat join them, as the owner chose. The client shows them and computes none of them.

**Rationale.** D01 keeps the client thin: prices depend on discounts and the outlook on the Age, which is rules arithmetic.

**Rejected.** Deriving them in the client from public city data.

**Relation.** Applies D01 to the new screens. The engine's view slice must add these fields (see [roadmap-proposal.md](roadmap-proposal.md), M15).

Source: [phone-layout.md](phone-layout.md), draft entries and "Server fields the screens rely on".

## D56. Seat crests are fixed by table position

**Decision (draft, 2026-10-21; accepted by the owner the same day).** Seven crests and seven seat colours are assigned by seat order at Start, each crest a distinct shape.

**Rationale.** A stable identity for the whole game without a lobby picker, and shapes that differ for colour vision.

**Rejected.** Crests chosen in the lobby (more personal, but needs a picker and a rule for duplicates).

**Relation.** New. The crest set is drawn in the vector loop under D18.

Source: [phone-layout.md](phone-layout.md), draft entries and owner answers.

## D57. Approved paintings are shared through game-specific bindings

**Decision (draft, 2026-10-21).** Haftgane reuses approved Duel paintings and their shipped, content-hashed URLs through explicit game-specific aliases (58 of the 80 designs from 55 paintings), and 21 new paintings cover the other 22 designs. New assets live in a game-qualified namespace. Rules ids and display ids stay independent, and Duel's art ids, files and hashes are unchanged byte for byte.

**Rationale.** Reuse saves 55 generations and keeps the two games visually related; new mixed-resource and profession scenes avoid misleading reuse.

**Rejected.** Regenerating all 80 designs. Treating every matching name as a fit. Copying Duel's art into a second directory. Changing Duel's art ids or hashes.

**Relation.** Extends D07, D18 and D24 to two games without changing them.

Source: [art-reuse.md](art-reuse.md), draft decisions.

## D58. Wonder paintings stay intact inside a wide live board

**Decision (draft, 2026-10-21).** The seven existing wonder paintings serve all fourteen sides, shown whole inside a wider responsive panel whose stage strip (two to four cells, drawn in CSS from server data) sits below the painting.

**Rationale.** The originals and their three sizes suffice, and the stages stay readable without covering the monuments.

**Rejected.** Cropping or stretching every painting into a wide backdrop. Baking costs and effects into pixels. Fourteen new paintings.

**Relation.** New, under D07 and D24.

Source: [art-reuse.md](art-reuse.md), draft decisions.

## D59. Concealed card backs follow what the viewer may know

**Decision (draft, 2026-10-21).** In Haftgane every concealed Age III card, guilds included, uses the ordinary Age III back; mixed-Age piles and tucked stages are drawn procedurally with a neutral mark. The violet guild back appears only where the category is public, such as a rules preview. The client never chooses a back from a hidden engine id.

**Rationale.** Which guilds are in the game is hidden, so a distinct guild back would leak it.

**Rejected.** Applying Duel's public guild back to every game. Painting new backs to solve a redaction problem.

**Relation.** Supersedes D12 for Haftgane only; Duel keeps D12.

Source: [art-reuse.md](art-reuse.md), draft decisions.

## D60. Duel and Haftgane share one online lobby, and its seat count picks the game

**Decision (draft, 2026-10-21, awaiting the owner).** Home keeps Duel's one-tap starts unchanged: play the bot, the tutorial, and pass and play. The online start opens one shared lobby of 2 to 7 seats, whose admin sets the seat count until Start. At Start, two filled seats create a Duel game with those two players, human or bot, under the same code and link; three to seven create a Haftgane game, and Haftgane's solo games use the same lobby. The lobby shows the settings of the game its seat count selects. It is its own small room kind, whose Start writes the chosen game's room file under the same code and swaps the actor. Duel gains one create path, a room built from a started lobby with both seats assigned, which then plays exactly like today's Duel game with the same seats; Duel's current online sheet and its start-when-the-friend-joins flow retire. The admin powers of D42 and the undo of D43 apply to Haftgane games only.

**Rationale.** A group then only has to know its size, as the owner suggested, while bot games and the tutorial stay one tap away.

**Rejected for now.** (A) Two separate Home entries with Duel untouched (a group of two must know that it wants Duel). (B) Everything through the lobby, with Duel's one-tap starts removed (it costs one-tap bot games and changes more of Duel).

**Relation.** Extends D31: the lobby is one more registered room kind. Changes Duel's online creation through one new create path, under a FULL review; D14's identity and claims carry through the swap.

Source: [platform-architecture.md](platform-architecture.md), "Shared lobby, awaiting the owner"; the owner's question of 2026-10-21.

## Owner answers (2026-10-21)

- The name Haftgane, with its id `haftgane`, in the title, the intro and D30, and in place of "the new game" or "the second game" throughout.
- Decided as recommended: D30 (base game only), D34 (unbeaten payments), D39 (hidden simultaneous sides), D45 (equal work), D54 (the recap opens by itself), D55 (running scores shown), D56 (fixed crests).
- Changed by the answers: D35 (picks changeable until the last confirmation), D37 (the notebook with statistics), D42 (admin powers replace the two-minute takeover; two details provisional), D43 (undo off once a second human sits down), D46 (the public release waits for Scholar).
- New: D60, the shared lobby, awaiting the owner.
