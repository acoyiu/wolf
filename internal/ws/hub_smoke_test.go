package ws

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

type wsBot struct {
	nickname string
	conn     *websocket.Conn
	done     chan struct{}

	mu          sync.RWMutex
	playerID    string
	roomCode    string
	players     int
	role        string
	mayorSecret string
	nightStep   int
	candidates  []string
	phase       string
	voteType    string
	gameOver    bool
	roomClosed  bool
	gameAborted bool
	lastError   string
	readErr     error

	timerSyncs     []timerSyncMsg
	voteCasts      []voteCastMsg
	gameOverResult *gameOverPayload
	winner         string
	reason         string
	word           string
	gameOverVotes  map[string]string
	gameOverRoles  map[string]string
}

type timerSyncMsg struct {
	Phase       string
	RemainingMs int
}

type voteCastMsg struct {
	Voter      string
	Target     string
	VotedCount int
	TotalVoters int
}

type gameOverPayload struct {
	Winner      string
	Reason      string
	Word        string
	Votes       map[string]string
	Roles       map[string]string
	MayorSecret string
}

type wsOutMsg struct {
	Type    string                 `json:"type"`
	Payload map[string]interface{} `json:"payload"`
}

func newWSBot(t *testing.T, wsURL, nickname string) *wsBot {
	t.Helper()
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial ws for %s: %v", nickname, err)
	}

	b := &wsBot{
		nickname: nickname,
		conn:     conn,
		done:     make(chan struct{}),
	}
	go b.readLoop()

	waitUntil(t, 10*time.Second, "connected message", func() bool {
		return b.PlayerID() != ""
	})
	return b
}

func (b *wsBot) readLoop() {
	defer close(b.done)
	for {
		var msg wsOutMsg
		if err := b.conn.ReadJSON(&msg); err != nil {
			b.mu.Lock()
			b.readErr = err
			b.mu.Unlock()
			return
		}
		b.apply(msg)
	}
}

func (b *wsBot) apply(msg wsOutMsg) {
	b.mu.Lock()
	defer b.mu.Unlock()

	switch msg.Type {
	case "connected":
		if id := payloadString(msg.Payload, "playerId"); id != "" {
			b.playerID = id
		}
	case "room_created", "room_state", "player_joined", "player_left":
		if roomCode := payloadString(msg.Payload, "roomCode"); roomCode != "" {
			b.roomCode = roomCode
		}
		if players, ok := msg.Payload["players"].([]interface{}); ok {
			b.players = len(players)
		}
	case "role_assigned":
		b.role = payloadString(msg.Payload, "role")
	case "mayor_secret":
		b.mayorSecret = payloadString(msg.Payload, "secretRole")
	case "night_step":
		if step := payloadInt(msg.Payload, "step"); step > 0 {
			b.nightStep = step
		}
		if candidates, ok := msg.Payload["candidates"].([]interface{}); ok {
			list := make([]string, 0, len(candidates))
			for _, v := range candidates {
				if s, ok := v.(string); ok && s != "" {
					list = append(list, s)
				}
			}
			b.candidates = list
		}
	case "night_reveal":
		if step := payloadInt(msg.Payload, "step"); step > 0 {
			b.nightStep = step
		}
	case "phase_change":
		if payloadString(msg.Payload, "phase") == "day" {
			b.phase = "day"
		}
	case "word_guessed":
		b.phase = "vote"
		b.voteType = "guess_seer"
	case "time_up", "tokens_depleted":
		b.phase = "vote"
		b.voteType = "guess_wolf"
	case "vote_state":
		b.phase = "vote"
		if vt := payloadString(msg.Payload, "voteType"); vt != "" {
			b.voteType = vt
		}
	case "timer_sync":
		ts := timerSyncMsg{
			Phase:       payloadString(msg.Payload, "phase"),
			RemainingMs: payloadInt(msg.Payload, "remainingMs"),
		}
		b.timerSyncs = append(b.timerSyncs, ts)
	case "vote_cast":
		vc := voteCastMsg{
			Voter:       payloadString(msg.Payload, "voter"),
			Target:      payloadString(msg.Payload, "target"),
			VotedCount:  payloadInt(msg.Payload, "votedCount"),
			TotalVoters: payloadInt(msg.Payload, "totalVoters"),
		}
		b.voteCasts = append(b.voteCasts, vc)
	case "game_over":
		b.phase = "result"
		b.gameOver = true
		b.winner = payloadString(msg.Payload, "winner")
		b.reason = payloadString(msg.Payload, "reason")
		b.word = payloadString(msg.Payload, "word")
		if votesRaw, ok := msg.Payload["votes"].(map[string]interface{}); ok {
			b.gameOverVotes = make(map[string]string, len(votesRaw))
			for k, v := range votesRaw {
				if s, ok := v.(string); ok {
					b.gameOverVotes[k] = s
				}
			}
		}
		if rolesRaw, ok := msg.Payload["roles"].(map[string]interface{}); ok {
			b.gameOverRoles = make(map[string]string, len(rolesRaw))
			for k, v := range rolesRaw {
				if s, ok := v.(string); ok {
					b.gameOverRoles[k] = s
				}
			}
		}
	case "game_aborted":
		b.gameAborted = true
	case "room_closed":
		b.roomClosed = true
	case "error":
		if e := payloadString(msg.Payload, "message"); e != "" {
			b.lastError = e
		}
	}
}

func (b *wsBot) Close() {
	_ = b.conn.Close()
	select {
	case <-b.done:
	case <-time.After(2 * time.Second):
	}
}

func (b *wsBot) Send(t *testing.T, typ string, payload map[string]interface{}) {
	t.Helper()
	env := map[string]interface{}{"type": typ, "payload": payload}
	if err := b.conn.WriteJSON(env); err != nil {
		t.Fatalf("send %s for %s: %v", typ, b.nickname, err)
	}
}

func (b *wsBot) PlayerID() string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.playerID
}

func (b *wsBot) RoomCode() string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.roomCode
}

func (b *wsBot) PlayersCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.players
}

func (b *wsBot) Role() string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.role
}

func (b *wsBot) EffectiveRole() string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.role == "mayor" {
		return b.mayorSecret
	}
	return b.role
}

func (b *wsBot) NightStep() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.nightStep
}

func (b *wsBot) Candidates() []string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	out := make([]string, len(b.candidates))
	copy(out, b.candidates)
	return out
}

func (b *wsBot) Phase() string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.phase
}

func (b *wsBot) VoteType() string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.voteType
}

func (b *wsBot) GameOver() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.gameOver
}

func (b *wsBot) RoomClosed() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.roomClosed
}

func (b *wsBot) GameAborted() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.gameAborted
}

func (b *wsBot) LastError() string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.lastError
}

func (b *wsBot) TimerSyncs() []timerSyncMsg {
	b.mu.RLock()
	defer b.mu.RUnlock()
	out := make([]timerSyncMsg, len(b.timerSyncs))
	copy(out, b.timerSyncs)
	return out
}

func (b *wsBot) VoteCasts() []voteCastMsg {
	b.mu.RLock()
	defer b.mu.RUnlock()
	out := make([]voteCastMsg, len(b.voteCasts))
	copy(out, b.voteCasts)
	return out
}

func (b *wsBot) GameOverVotes() map[string]string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.gameOverVotes == nil {
		return nil
	}
	out := make(map[string]string, len(b.gameOverVotes))
	for k, v := range b.gameOverVotes {
		out[k] = v
	}
	return out
}

func (b *wsBot) GameOverRoles() map[string]string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.gameOverRoles == nil {
		return nil
	}
	out := make(map[string]string, len(b.gameOverRoles))
	for k, v := range b.gameOverRoles {
		out[k] = v
	}
	return out
}

func (b *wsBot) Winner() string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.winner
}

func (b *wsBot) Word() string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.word
}


func payloadString(payload map[string]interface{}, key string) string {
	v, ok := payload[key]
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

func payloadInt(payload map[string]interface{}, key string) int {
	v, ok := payload[key]
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case int:
		return n
	case float64:
		return int(n)
	default:
		return 0
	}
}

func waitUntil(t *testing.T, timeout time.Duration, label string, fn func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for %s", label)
}

func pickOther(ids []string, self string) string {
	for _, id := range ids {
		if id != self {
			return id
		}
	}
	return ""
}

func botCanVote(b *wsBot, voteType string) bool {
	if voteType == "guess_seer" && b.EffectiveRole() != "werewolf" {
		return false
	}
	if voteType == "guess_wolf" && b.EffectiveRole() == "werewolf" {
		return false
	}
	return true
}

func TestHubSmokeFourPlayersNoBrowsers(t *testing.T) {
	h := NewHub(5*time.Second, 5*time.Second)
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", h.HandleWS)

	server := httptest.NewServer(mux)
	defer func() { h.Shutdown(); server.Close() }()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

	bots := []*wsBot{
		newWSBot(t, wsURL, "P1"),
		newWSBot(t, wsURL, "P2"),
		newWSBot(t, wsURL, "P3"),
		newWSBot(t, wsURL, "P4"),
	}
	defer func() {
		for _, b := range bots {
			b.Close()
		}
	}()

	host := bots[0]
	host.Send(t, "create_room", map[string]interface{}{
		"nickname":      host.nickname,
		"targetPlayers": 4,
		"difficulty":    "easy",
	})

	waitUntil(t, 10*time.Second, "host room created", func() bool {
		return host.RoomCode() != ""
	})
	roomCode := host.RoomCode()

	for i := 1; i < len(bots); i++ {
		bots[i].Send(t, "join_room", map[string]interface{}{
			"roomCode": roomCode,
			"nickname": bots[i].nickname,
		})
	}

	waitUntil(t, 15*time.Second, "all players joined", func() bool {
		for _, b := range bots {
			if b.RoomCode() != roomCode || b.PlayersCount() != 4 {
				return false
			}
		}
		return true
	})

	host.Send(t, "start_game", map[string]interface{}{})

	waitUntil(t, 10*time.Second, "roles assigned", func() bool {
		for _, b := range bots {
			if b.Role() == "" {
				return false
			}
		}
		return true
	})

	waitUntil(t, 10*time.Second, "mayor candidate words", func() bool {
		return host.NightStep() == 1 && len(host.Candidates()) >= 1
	})

	chosenWord := host.Candidates()[0]
	host.Send(t, "night_pick_word", map[string]interface{}{"word": chosenWord})

	for i := 1; i < len(bots); i++ {
		bots[i].Send(t, "night_confirm", map[string]interface{}{})
	}

	waitUntil(t, 10*time.Second, "night step 2 started", func() bool {
		return host.NightStep() == 2 || host.Phase() == "day"
	})

	for i := 0; i < len(bots); i++ {
		bots[i].Send(t, "night_confirm", map[string]interface{}{})
	}

	waitUntil(t, 10*time.Second, "day phase started", func() bool {
		for _, b := range bots {
			if b.Phase() != "day" {
				return false
			}
		}
		return true
	})

	host.Send(t, "day_token", map[string]interface{}{"token": "correct"})

	waitUntil(t, 10*time.Second, "vote phase started", func() bool {
		for _, b := range bots {
			if b.Phase() != "vote" {
				return false
			}
		}
		return true
	})

	voteType := bots[0].VoteType()
	if voteType == "" {
		voteType = "guess_seer"
	}

	ids := make([]string, 0, len(bots))
	for _, b := range bots {
		ids = append(ids, b.PlayerID())
	}

	var voters []*wsBot
	for _, b := range bots {
		if voteType == "guess_seer" {
			if b.EffectiveRole() == "werewolf" {
				voters = append(voters, b)
			}
			continue
		}
		voters = append(voters, b)
	}
	if len(voters) == 0 {
		t.Fatalf("no eligible voters for vote type %s", voteType)
	}

	for _, v := range voters {
		target := pickOther(ids, v.PlayerID())
		if target == "" {
			t.Fatalf("no valid vote target for voter %s", v.PlayerID())
		}
		v.Send(t, "vote_cast", map[string]interface{}{"target": target})
	}

	waitUntil(t, 10*time.Second, "game over", func() bool {
		for _, b := range bots {
			if !b.GameOver() {
				return false
			}
		}
		return true
	})

	for _, b := range bots {
		if errMsg := b.LastError(); errMsg != "" {
			t.Fatalf("bot %s received unexpected error: %s", b.nickname, errMsg)
		}
	}
}

func TestHubSmokeDayTimeoutGuessWolf(t *testing.T) {
	// Use short day timeout so the test doesn't wait long.
	h := NewHub(2*time.Second, 5*time.Second)
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", h.HandleWS)

	server := httptest.NewServer(mux)
	defer func() { h.Shutdown(); server.Close() }()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

	bots := []*wsBot{
		newWSBot(t, wsURL, "A1"),
		newWSBot(t, wsURL, "A2"),
		newWSBot(t, wsURL, "A3"),
		newWSBot(t, wsURL, "A4"),
	}
	defer func() {
		for _, b := range bots {
			b.Close()
		}
	}()

	host := bots[0]
	host.Send(t, "create_room", map[string]interface{}{
		"nickname":      host.nickname,
		"targetPlayers": 4,
		"difficulty":    "easy",
	})

	waitUntil(t, 10*time.Second, "host room created", func() bool {
		return host.RoomCode() != ""
	})
	roomCode := host.RoomCode()

	for i := 1; i < len(bots); i++ {
		bots[i].Send(t, "join_room", map[string]interface{}{
			"roomCode": roomCode,
			"nickname": bots[i].nickname,
		})
	}

	waitUntil(t, 15*time.Second, "all players joined", func() bool {
		for _, b := range bots {
			if b.RoomCode() != roomCode || b.PlayersCount() != 4 {
				return false
			}
		}
		return true
	})

	host.Send(t, "start_game", map[string]interface{}{})

	waitUntil(t, 10*time.Second, "roles assigned", func() bool {
		for _, b := range bots {
			if b.Role() == "" {
				return false
			}
		}
		return true
	})

	waitUntil(t, 10*time.Second, "mayor candidate words", func() bool {
		return host.NightStep() == 1 && len(host.Candidates()) >= 1
	})

	chosenWord := host.Candidates()[0]
	host.Send(t, "night_pick_word", map[string]interface{}{"word": chosenWord})

	for i := 1; i < len(bots); i++ {
		bots[i].Send(t, "night_confirm", map[string]interface{}{})
	}

	waitUntil(t, 10*time.Second, "night step 2 started", func() bool {
		return host.NightStep() == 2 || host.Phase() == "day"
	})

	for i := 0; i < len(bots); i++ {
		bots[i].Send(t, "night_confirm", map[string]interface{}{})
	}

	waitUntil(t, 10*time.Second, "day phase started", func() bool {
		for _, b := range bots {
			if b.Phase() != "day" {
				return false
			}
		}
		return true
	})

	// Do NOT send "correct" token. Let the day timer expire (2s).
	waitUntil(t, 10*time.Second, "vote phase via timeout", func() bool {
		for _, b := range bots {
			if b.Phase() != "vote" {
				return false
			}
		}
		return true
	})

	// Verify it's guess_wolf vote type
	for _, b := range bots {
		if b.VoteType() != "guess_wolf" {
			t.Fatalf("bot %s voteType=%s want guess_wolf", b.nickname, b.VoteType())
		}
	}

	// In guess_wolf, only non-wolf players can vote.
	ids := make([]string, 0, len(bots))
	for _, b := range bots {
		ids = append(ids, b.PlayerID())
	}
	for _, b := range bots {
		if !botCanVote(b, "guess_wolf") {
			continue
		}
		target := pickOther(ids, b.PlayerID())
		b.Send(t, "vote_cast", map[string]interface{}{"target": target})
	}

	waitUntil(t, 10*time.Second, "game over", func() bool {
		for _, b := range bots {
			if !b.GameOver() {
				return false
			}
		}
		return true
	})

	for _, b := range bots {
		if errMsg := b.LastError(); errMsg != "" {
			t.Fatalf("bot %s received unexpected error: %s", b.nickname, errMsg)
		}
	}
}

func TestHubSmokeResumeSession(t *testing.T) {
	h := NewHub(30*time.Second, 30*time.Second)
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", h.HandleWS)

	server := httptest.NewServer(mux)
	defer func() { h.Shutdown(); server.Close() }()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

	bots := []*wsBot{
		newWSBot(t, wsURL, "R1"),
		newWSBot(t, wsURL, "R2"),
		newWSBot(t, wsURL, "R3"),
		newWSBot(t, wsURL, "R4"),
	}
	defer func() {
		for _, b := range bots {
			b.Close()
		}
	}()

	host := bots[0]
	host.Send(t, "create_room", map[string]interface{}{
		"nickname":      host.nickname,
		"targetPlayers": 4,
		"difficulty":    "easy",
	})

	waitUntil(t, 10*time.Second, "host room created", func() bool {
		return host.RoomCode() != ""
	})
	roomCode := host.RoomCode()

	for i := 1; i < len(bots); i++ {
		bots[i].Send(t, "join_room", map[string]interface{}{
			"roomCode": roomCode,
			"nickname": bots[i].nickname,
		})
	}

	waitUntil(t, 15*time.Second, "all players joined", func() bool {
		for _, b := range bots {
			if b.RoomCode() != roomCode || b.PlayersCount() != 4 {
				return false
			}
		}
		return true
	})

	host.Send(t, "start_game", map[string]interface{}{})

	waitUntil(t, 10*time.Second, "roles assigned", func() bool {
		for _, b := range bots {
			if b.Role() == "" {
				return false
			}
		}
		return true
	})

	// Save disconnecting player's info
	disconnected := bots[1]
	savedPlayerID := disconnected.PlayerID()
	savedRole := disconnected.Role()

	// Disconnect the player
	_ = disconnected.conn.Close()
	<-disconnected.done

	// Wait a moment for the server to notice the disconnect
	time.Sleep(500 * time.Millisecond)

	// Reconnect with new WebSocket and resume session
	resumed := newWSBot(t, wsURL, "R2_new")
	defer resumed.Close()

	resumed.Send(t, "resume_session", map[string]interface{}{
		"playerId": savedPlayerID,
		"roomCode": roomCode,
		"nickname": "R2",
	})

	// The bot should receive its role back
	waitUntil(t, 5*time.Second, "resumed bot gets role", func() bool {
		return resumed.Role() != ""
	})

	if resumed.Role() != savedRole {
		t.Fatalf("resumed role=%s want %s", resumed.Role(), savedRole)
	}

	// Replace the old bot reference so we can clean up
	bots[1] = resumed
}

func TestHubSmokeRefreshDuringDayDoesNotCrashOthers(t *testing.T) {
	h := NewHub(30*time.Second, 30*time.Second)
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", h.HandleWS)

	server := httptest.NewServer(mux)
	defer func() { h.Shutdown(); server.Close() }()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

	bots := []*wsBot{
		newWSBot(t, wsURL, "D1"),
		newWSBot(t, wsURL, "D2"),
		newWSBot(t, wsURL, "D3"),
		newWSBot(t, wsURL, "D4"),
	}
	defer func() {
		for _, b := range bots {
			b.Close()
		}
	}()

	host := bots[0]
	host.Send(t, "create_room", map[string]interface{}{
		"nickname":      host.nickname,
		"targetPlayers": 4,
		"difficulty":    "easy",
	})
	waitUntil(t, 10*time.Second, "host room created", func() bool {
		return host.RoomCode() != ""
	})
	roomCode := host.RoomCode()

	for i := 1; i < len(bots); i++ {
		bots[i].Send(t, "join_room", map[string]interface{}{
			"roomCode": roomCode,
			"nickname": bots[i].nickname,
		})
	}
	waitUntil(t, 15*time.Second, "all joined", func() bool {
		for _, b := range bots {
			if b.PlayersCount() != 4 {
				return false
			}
		}
		return true
	})

	host.Send(t, "start_game", map[string]interface{}{})
	waitUntil(t, 10*time.Second, "roles assigned", func() bool {
		for _, b := range bots {
			if b.Role() == "" {
				return false
			}
		}
		return true
	})

	waitUntil(t, 10*time.Second, "mayor candidates", func() bool {
		return host.NightStep() == 1 && len(host.Candidates()) >= 1
	})
	host.Send(t, "night_pick_word", map[string]interface{}{"word": host.Candidates()[0]})

	for i := 1; i < len(bots); i++ {
		bots[i].Send(t, "night_confirm", map[string]interface{}{})
	}
	waitUntil(t, 10*time.Second, "step 2", func() bool {
		return host.NightStep() == 2 || host.Phase() == "day"
	})
	for i := 0; i < len(bots); i++ {
		bots[i].Send(t, "night_confirm", map[string]interface{}{})
	}
	waitUntil(t, 10*time.Second, "day phase", func() bool {
		for _, b := range bots {
			if b.Phase() != "day" {
				return false
			}
		}
		return true
	})

	// --- Simulate a browser refresh: disconnect and quickly reconnect ---
	refresher := bots[2]
	savedID := refresher.PlayerID()
	savedRole := refresher.Role()

	_ = refresher.conn.Close()
	<-refresher.done
	time.Sleep(200 * time.Millisecond)

	// Reconnect
	newBot := newWSBot(t, wsURL, "D3_new")
	newBot.Send(t, "resume_session", map[string]interface{}{
		"playerId": savedID,
		"roomCode": roomCode,
		"nickname": "D3",
	})
	waitUntil(t, 5*time.Second, "resumed gets role", func() bool {
		return newBot.Role() != ""
	})
	if newBot.Role() != savedRole {
		t.Fatalf("resumed role=%s want %s", newBot.Role(), savedRole)
	}
	bots[2] = newBot

	// Wait 2 seconds and check other bots are NOT crashed
	time.Sleep(2 * time.Second)

	for i, b := range bots {
		if b.RoomClosed() {
			t.Fatalf("bot[%d] %s received room_closed after player refresh", i, b.nickname)
		}
		if b.GameAborted() {
			t.Fatalf("bot[%d] %s received game_aborted after player refresh", i, b.nickname)
		}
	}

	// Game should still be in day phase
	for _, b := range bots {
		if b.Phase() != "day" {
			t.Fatalf("bot %s phase=%s want day", b.nickname, b.Phase())
		}
	}

	// --- Now test HOST refresh ---
	hostSavedID := host.PlayerID()
	hostSavedRole := host.Role()

	_ = host.conn.Close()
	<-host.done
	time.Sleep(200 * time.Millisecond)

	newHost := newWSBot(t, wsURL, "D1_new")
	newHost.Send(t, "resume_session", map[string]interface{}{
		"playerId": hostSavedID,
		"roomCode": roomCode,
		"nickname": "D1",
	})
	waitUntil(t, 5*time.Second, "host resumed gets role", func() bool {
		return newHost.Role() != ""
	})
	if newHost.Role() != hostSavedRole {
		t.Fatalf("host resumed role=%s want %s", newHost.Role(), hostSavedRole)
	}
	bots[0] = newHost

	time.Sleep(2 * time.Second)

	for i, b := range bots {
		if b.RoomClosed() {
			t.Fatalf("bot[%d] %s received room_closed after host refresh", i, b.nickname)
		}
		if b.GameAborted() {
			t.Fatalf("bot[%d] %s received game_aborted after host refresh", i, b.nickname)
		}
	}

	// Everyone should still be in day phase
	for _, b := range bots {
		if b.Phase() != "day" {
			t.Fatalf("after host refresh: bot %s phase=%s want day", b.nickname, b.Phase())
		}
	}

	// Clean up
	for _, b := range bots {
		b.Close()
	}
}

func TestHubSmokeRefreshInWaitingRoom(t *testing.T) {
	h := NewHub(30*time.Second, 30*time.Second)
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", h.HandleWS)

	server := httptest.NewServer(mux)
	defer func() { h.Shutdown(); server.Close() }()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

	bots := []*wsBot{
		newWSBot(t, wsURL, "W1"),
		newWSBot(t, wsURL, "W2"),
		newWSBot(t, wsURL, "W3"),
		newWSBot(t, wsURL, "W4"),
	}
	defer func() {
		for _, b := range bots {
			b.Close()
		}
	}()

	host := bots[0]
	host.Send(t, "create_room", map[string]interface{}{
		"nickname":      host.nickname,
		"targetPlayers": 4,
		"difficulty":    "easy",
	})
	waitUntil(t, 10*time.Second, "host room created", func() bool {
		return host.RoomCode() != ""
	})
	roomCode := host.RoomCode()

	for i := 1; i < len(bots); i++ {
		bots[i].Send(t, "join_room", map[string]interface{}{
			"roomCode": roomCode,
			"nickname": bots[i].nickname,
		})
	}
	waitUntil(t, 15*time.Second, "all joined", func() bool {
		for _, b := range bots {
			if b.PlayersCount() != 4 {
				return false
			}
		}
		return true
	})

	// --- Host refreshes in waiting room ---
	hostSavedID := host.PlayerID()
	_ = host.conn.Close()
	<-host.done
	time.Sleep(200 * time.Millisecond)

	// Other players should NOT have received room_closed
	for i := 1; i < len(bots); i++ {
		if bots[i].RoomClosed() {
			t.Fatalf("bot[%d] %s received room_closed after host refresh in waiting room", i, bots[i].nickname)
		}
	}

	// Host reconnects and resumes
	newHost := newWSBot(t, wsURL, "W1_new")
	newHost.Send(t, "resume_session", map[string]interface{}{
		"playerId": hostSavedID,
		"roomCode": roomCode,
		"nickname": "W1",
	})
	waitUntil(t, 5*time.Second, "host resumed in waiting room", func() bool {
		return newHost.RoomCode() != ""
	})
	bots[0] = newHost

	// Verify room still has 4 players
	waitUntil(t, 10*time.Second, "still 4 players after host resume", func() bool {
		return newHost.PlayersCount() == 4
	})

	// --- Non-host refreshes in waiting room ---
	p2SavedID := bots[2].PlayerID()
	_ = bots[2].conn.Close()
	<-bots[2].done
	time.Sleep(200 * time.Millisecond)

	// Other players should NOT have received room_closed
	for i, b := range bots {
		if i == 2 {
			continue
		}
		if b.RoomClosed() {
			t.Fatalf("bot[%d] %s received room_closed after non-host refresh", i, b.nickname)
		}
	}

	// Non-host reconnects and resumes
	newP2 := newWSBot(t, wsURL, "W3_new")
	newP2.Send(t, "resume_session", map[string]interface{}{
		"playerId": p2SavedID,
		"roomCode": roomCode,
		"nickname": "W3",
	})
	waitUntil(t, 5*time.Second, "non-host resumed in waiting room", func() bool {
		return newP2.RoomCode() != ""
	})
	bots[2] = newP2

	// Verify room still intact
	waitUntil(t, 10*time.Second, "still 4 players after non-host resume", func() bool {
		return newP2.PlayersCount() == 4
	})

	for _, b := range bots {
		b.Close()
	}
}

// playThroughNight is a helper that advances bots through the night phase to day.
func playThroughNight(t *testing.T, bots []*wsBot) {
	t.Helper()
	host := bots[0]

	waitUntil(t, 10*time.Second, "roles assigned", func() bool {
		for _, b := range bots {
			if b.Role() == "" {
				return false
			}
		}
		return true
	})

	waitUntil(t, 10*time.Second, "mayor candidates", func() bool {
		return host.NightStep() == 1 && len(host.Candidates()) >= 1
	})

	host.Send(t, "night_pick_word", map[string]interface{}{"word": host.Candidates()[0]})

	for i := 1; i < len(bots); i++ {
		bots[i].Send(t, "night_confirm", map[string]interface{}{})
	}
	waitUntil(t, 10*time.Second, "step 2", func() bool {
		return host.NightStep() == 2 || host.Phase() == "day"
	})
	for i := 0; i < len(bots); i++ {
		bots[i].Send(t, "night_confirm", map[string]interface{}{})
	}
	waitUntil(t, 5*time.Second, "day phase", func() bool {
		for _, b := range bots {
			if b.Phase() != "day" {
				return false
			}
		}
		return true
	})
}

// setupAndJoinRoom creates a room with the host and joins all other bots.
func setupAndJoinRoom(t *testing.T, bots []*wsBot) string {
	t.Helper()
	host := bots[0]
	host.Send(t, "create_room", map[string]interface{}{
		"nickname":      host.nickname,
		"targetPlayers": len(bots),
		"difficulty":    "easy",
	})
	waitUntil(t, 10*time.Second, "host room created", func() bool {
		return host.RoomCode() != ""
	})
	roomCode := host.RoomCode()

	for i := 1; i < len(bots); i++ {
		bots[i].Send(t, "join_room", map[string]interface{}{
			"roomCode": roomCode,
			"nickname": bots[i].nickname,
		})
	}
	waitUntil(t, 15*time.Second, "all joined", func() bool {
		for _, b := range bots {
			if b.RoomCode() != roomCode || b.PlayersCount() != len(bots) {
				return false
			}
		}
		return true
	})
	return roomCode
}

func TestHubTimerSyncDuringDayAndVote(t *testing.T) {
	h := NewHub(3*time.Second, 3*time.Second)
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", h.HandleWS)
	server := httptest.NewServer(mux)
	defer func() { h.Shutdown(); server.Close() }()
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

	bots := []*wsBot{
		newWSBot(t, wsURL, "T1"),
		newWSBot(t, wsURL, "T2"),
		newWSBot(t, wsURL, "T3"),
		newWSBot(t, wsURL, "T4"),
	}
	defer func() { for _, b := range bots { b.Close() } }()

	setupAndJoinRoom(t, bots)
	bots[0].Send(t, "start_game", map[string]interface{}{})
	playThroughNight(t, bots)

	// After day phase starts, all bots should receive timer_sync messages
	waitUntil(t, 5*time.Second, "timer_sync for day phase", func() bool {
		for _, b := range bots {
			syncs := b.TimerSyncs()
			hasDaySync := false
			for _, s := range syncs {
				if s.Phase == "day" && s.RemainingMs > 0 {
					hasDaySync = true
					break
				}
			}
			if !hasDaySync {
				return false
			}
		}
		return true
	})

	// Verify timer_sync has reasonable remaining time (should be roughly 3000ms or less)
	for _, b := range bots {
		syncs := b.TimerSyncs()
		for _, s := range syncs {
			if s.Phase == "day" && s.RemainingMs > 4000 {
				t.Fatalf("bot %s got unreasonable day timer: %dms", b.nickname, s.RemainingMs)
			}
		}
	}

	// Let day timer expire to enter vote phase
	waitUntil(t, 10*time.Second, "vote phase via timeout", func() bool {
		for _, b := range bots {
			if b.Phase() != "vote" {
				return false
			}
		}
		return true
	})

	// Verify timer_sync for vote phase too
	waitUntil(t, 5*time.Second, "timer_sync for vote phase", func() bool {
		for _, b := range bots {
			syncs := b.TimerSyncs()
			hasVoteSync := false
			for _, s := range syncs {
				if s.Phase == "vote" && s.RemainingMs > 0 {
					hasVoteSync = true
					break
				}
			}
			if !hasVoteSync {
				return false
			}
		}
		return true
	})

	// Vote to end the game (wolves excluded in guess_wolf)
	ids := make([]string, len(bots))
	for i, b := range bots {
		ids[i] = b.PlayerID()
	}
	for _, b := range bots {
		if !botCanVote(b, "guess_wolf") {
			continue
		}
		b.Send(t, "vote_cast", map[string]interface{}{"target": pickOther(ids, b.PlayerID())})
	}

	waitUntil(t, 10*time.Second, "game over", func() bool {
		for _, b := range bots {
			if !b.GameOver() {
				return false
			}
		}
		return true
	})
}

func TestHubVoteProgressBroadcast(t *testing.T) {
	h := NewHub(30*time.Second, 30*time.Second)
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", h.HandleWS)
	server := httptest.NewServer(mux)
	defer func() { h.Shutdown(); server.Close() }()
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

	bots := []*wsBot{
		newWSBot(t, wsURL, "V1"),
		newWSBot(t, wsURL, "V2"),
		newWSBot(t, wsURL, "V3"),
		newWSBot(t, wsURL, "V4"),
	}
	defer func() { for _, b := range bots { b.Close() } }()

	setupAndJoinRoom(t, bots)
	bots[0].Send(t, "start_game", map[string]interface{}{})
	playThroughNight(t, bots)

	// Send "correct" to trigger guess_seer vote
	bots[0].Send(t, "day_token", map[string]interface{}{"token": "correct"})
	waitUntil(t, 5*time.Second, "vote phase", func() bool {
		for _, b := range bots {
			if b.Phase() != "vote" {
				return false
			}
		}
		return true
	})

	voteType := bots[0].VoteType()

	// Determine eligible voters and vote one at a time
	ids := make([]string, len(bots))
	for i, b := range bots {
		ids[i] = b.PlayerID()
	}

	var voters []*wsBot
	for _, b := range bots {
		if botCanVote(b, voteType) {
			voters = append(voters, b)
		}
	}

	for i, v := range voters {
		target := pickOther(ids, v.PlayerID())
		v.Send(t, "vote_cast", map[string]interface{}{"target": target})

		expectedCount := i + 1
		waitUntil(t, 10*time.Second, "vote_cast broadcast received", func() bool {
			for _, b := range bots {
				casts := b.VoteCasts()
				if len(casts) < expectedCount {
					return false
				}
				last := casts[expectedCount-1]
				if last.VotedCount != expectedCount {
					return false
				}
			}
			return true
		})
	}

	// Verify the final vote_cast has correct totalVoters
	waitUntil(t, 5*time.Second, "game over after all votes", func() bool {
		for _, b := range bots {
			if !b.GameOver() {
				return false
			}
		}
		return true
	})

	// Verify votedCount progression in vote_cast messages
	for _, b := range bots {
		casts := b.VoteCasts()
		if len(casts) == 0 {
			t.Fatalf("bot %s received no vote_cast messages", b.nickname)
		}
		lastCast := casts[len(casts)-1]
		if lastCast.TotalVoters <= 0 {
			t.Fatalf("bot %s last vote_cast has totalVoters=%d", b.nickname, lastCast.TotalVoters)
		}
		if lastCast.VotedCount != lastCast.TotalVoters {
			t.Fatalf("bot %s last vote_cast votedCount=%d != totalVoters=%d",
				b.nickname, lastCast.VotedCount, lastCast.TotalVoters)
		}
	}
}

func TestHubGameOverIncludesVotesAndRoles(t *testing.T) {
	h := NewHub(30*time.Second, 30*time.Second)
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", h.HandleWS)
	server := httptest.NewServer(mux)
	defer func() { h.Shutdown(); server.Close() }()
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

	bots := []*wsBot{
		newWSBot(t, wsURL, "G1"),
		newWSBot(t, wsURL, "G2"),
		newWSBot(t, wsURL, "G3"),
		newWSBot(t, wsURL, "G4"),
	}
	defer func() { for _, b := range bots { b.Close() } }()

	setupAndJoinRoom(t, bots)
	bots[0].Send(t, "start_game", map[string]interface{}{})
	playThroughNight(t, bots)

	bots[0].Send(t, "day_token", map[string]interface{}{"token": "correct"})
	waitUntil(t, 5*time.Second, "vote phase", func() bool {
		for _, b := range bots {
			if b.Phase() != "vote" {
				return false
			}
		}
		return true
	})

	voteType := bots[0].VoteType()
	ids := make([]string, len(bots))
	for i, b := range bots {
		ids[i] = b.PlayerID()
	}

	for _, b := range bots {
		if !botCanVote(b, voteType) {
			continue
		}
		target := pickOther(ids, b.PlayerID())
		b.Send(t, "vote_cast", map[string]interface{}{"target": target})
	}

	waitUntil(t, 5*time.Second, "game over", func() bool {
		for _, b := range bots {
			if !b.GameOver() {
				return false
			}
		}
		return true
	})

	// Verify game_over contains votes map
	for _, b := range bots {
		votes := b.GameOverVotes()
		if votes == nil {
			t.Fatalf("bot %s game_over payload missing votes", b.nickname)
		}
		if len(votes) == 0 {
			t.Fatalf("bot %s game_over has empty votes map", b.nickname)
		}
		for voterID, targetID := range votes {
			if voterID == "" || targetID == "" {
				t.Fatalf("bot %s has empty voter/target in votes", b.nickname)
			}
		}
	}

	// Verify game_over contains roles map
	for _, b := range bots {
		roles := b.GameOverRoles()
		if roles == nil {
			t.Fatalf("bot %s game_over payload missing roles", b.nickname)
		}
		if len(roles) != len(bots) {
			t.Fatalf("bot %s game_over roles count=%d want %d", b.nickname, len(roles), len(bots))
		}
		for playerID, role := range roles {
			if playerID == "" || role == "" {
				t.Fatalf("bot %s has empty playerID/role in roles", b.nickname)
			}
		}
	}

	// Verify winner is not empty
	for _, b := range bots {
		if b.Winner() == "" {
			t.Fatalf("bot %s game_over missing winner", b.nickname)
		}
		if b.Word() == "" {
			t.Fatalf("bot %s game_over missing word", b.nickname)
		}
	}
}


func TestHubReconnectDuringVotePhase(t *testing.T) {
	h := NewHub(30*time.Second, 30*time.Second)
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", h.HandleWS)
	server := httptest.NewServer(mux)
	defer func() { h.Shutdown(); server.Close() }()
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

	bots := []*wsBot{
		newWSBot(t, wsURL, "RV1"),
		newWSBot(t, wsURL, "RV2"),
		newWSBot(t, wsURL, "RV3"),
		newWSBot(t, wsURL, "RV4"),
	}
	defer func() { for _, b := range bots { b.Close() } }()

	roomCode := setupAndJoinRoom(t, bots)
	bots[0].Send(t, "start_game", map[string]interface{}{})
	playThroughNight(t, bots)

	bots[0].Send(t, "day_token", map[string]interface{}{"token": "correct"})
	waitUntil(t, 5*time.Second, "vote phase", func() bool {
		for _, b := range bots {
			if b.Phase() != "vote" {
				return false
			}
		}
		return true
	})

	voteType := bots[0].VoteType()

	// Find a voter who hasn't voted yet and disconnect them
	var disconnectedBot *wsBot
	var disconnectedIdx int
	for i, b := range bots {
		if !botCanVote(b, voteType) {
			continue
		}
		disconnectedBot = b
		disconnectedIdx = i
		break
	}
	if disconnectedBot == nil {
		t.Fatal("no eligible voter to disconnect")
	}

	savedID := disconnectedBot.PlayerID()
	savedRole := disconnectedBot.Role()
	_ = disconnectedBot.conn.Close()
	<-disconnectedBot.done
	time.Sleep(300 * time.Millisecond)

	// Reconnect
	newBot := newWSBot(t, wsURL, disconnectedBot.nickname+"_new")
	newBot.Send(t, "resume_session", map[string]interface{}{
		"playerId": savedID,
		"roomCode": roomCode,
		"nickname": disconnectedBot.nickname,
	})

	waitUntil(t, 5*time.Second, "reconnected gets role back", func() bool {
		return newBot.Role() != ""
	})

	if newBot.Role() != savedRole {
		t.Fatalf("reconnected role=%s want %s", newBot.Role(), savedRole)
	}
	bots[disconnectedIdx] = newBot

	// Verify no other bots crashed
	for i, b := range bots {
		if b.RoomClosed() {
			t.Fatalf("bot[%d] %s got room_closed after vote-phase reconnect", i, b.nickname)
		}
		if b.GameAborted() {
			t.Fatalf("bot[%d] %s got game_aborted after vote-phase reconnect", i, b.nickname)
		}
	}

	// The reconnected bot should be able to vote
	ids := make([]string, len(bots))
	for i, b := range bots {
		ids[i] = b.PlayerID()
	}

	// All eligible voters vote now (including the reconnected one)
	for _, b := range bots {
		if !botCanVote(b, voteType) {
			continue
		}
		target := pickOther(ids, b.PlayerID())
		b.Send(t, "vote_cast", map[string]interface{}{"target": target})
	}

	waitUntil(t, 5*time.Second, "game over after reconnect vote", func() bool {
		for _, b := range bots {
			if !b.GameOver() {
				return false
			}
		}
		return true
	})
}

func TestHubMultipleRapidRefreshes(t *testing.T) {
	h := NewHub(30*time.Second, 30*time.Second)
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", h.HandleWS)
	server := httptest.NewServer(mux)
	defer func() { h.Shutdown(); server.Close() }()
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

	bots := []*wsBot{
		newWSBot(t, wsURL, "MR1"),
		newWSBot(t, wsURL, "MR2"),
		newWSBot(t, wsURL, "MR3"),
		newWSBot(t, wsURL, "MR4"),
	}
	defer func() { for _, b := range bots { b.Close() } }()

	roomCode := setupAndJoinRoom(t, bots)
	bots[0].Send(t, "start_game", map[string]interface{}{})
	playThroughNight(t, bots)

	// Rapidly refresh the same player 3 times in a row
	refreshPlayer := bots[1]
	savedID := refreshPlayer.PlayerID()
	savedRole := refreshPlayer.Role()

	for attempt := 0; attempt < 3; attempt++ {
		_ = refreshPlayer.conn.Close()
		<-refreshPlayer.done
		time.Sleep(100 * time.Millisecond)

		newBot := newWSBot(t, wsURL, refreshPlayer.nickname+"_new")
		newBot.Send(t, "resume_session", map[string]interface{}{
			"playerId": savedID,
			"roomCode": roomCode,
			"nickname": "MR2",
		})
		waitUntil(t, 5*time.Second, "resume attempt "+string(rune('1'+attempt)), func() bool {
			return newBot.Role() != ""
		})
		if newBot.Role() != savedRole {
			t.Fatalf("attempt %d: role=%s want %s", attempt, newBot.Role(), savedRole)
		}
		refreshPlayer = newBot
		bots[1] = newBot
	}

	// Verify all other bots are fine
	for i, b := range bots {
		if b.RoomClosed() {
			t.Fatalf("bot[%d] %s room_closed after rapid refreshes", i, b.nickname)
		}
		if b.GameAborted() {
			t.Fatalf("bot[%d] %s game_aborted after rapid refreshes", i, b.nickname)
		}
	}

	// Game should still be in day phase
	for _, b := range bots {
		if b.Phase() != "day" {
			t.Fatalf("bot %s phase=%s want day after rapid refreshes", b.nickname, b.Phase())
		}
	}
}

func TestHubReconnectGetsTimerSync(t *testing.T) {
	h := NewHub(30*time.Second, 30*time.Second)
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", h.HandleWS)
	server := httptest.NewServer(mux)
	defer func() { h.Shutdown(); server.Close() }()
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

	bots := []*wsBot{
		newWSBot(t, wsURL, "TS1"),
		newWSBot(t, wsURL, "TS2"),
		newWSBot(t, wsURL, "TS3"),
		newWSBot(t, wsURL, "TS4"),
	}
	defer func() { for _, b := range bots { b.Close() } }()

	roomCode := setupAndJoinRoom(t, bots)
	bots[0].Send(t, "start_game", map[string]interface{}{})
	playThroughNight(t, bots)

	// Now we're in day phase. Wait a second for timer sync to be sent.
	time.Sleep(1500 * time.Millisecond)

	// Disconnect player 2
	savedID := bots[2].PlayerID()
	_ = bots[2].conn.Close()
	<-bots[2].done
	time.Sleep(300 * time.Millisecond)

	// Reconnect
	newBot := newWSBot(t, wsURL, "TS3_new")
	newBot.Send(t, "resume_session", map[string]interface{}{
		"playerId": savedID,
		"roomCode": roomCode,
		"nickname": "TS3",
	})
	waitUntil(t, 5*time.Second, "reconnected gets role", func() bool {
		return newBot.Role() != ""
	})
	bots[2] = newBot

	// The reconnected bot should receive a timer_sync for the day phase
	waitUntil(t, 5*time.Second, "reconnected bot gets timer_sync", func() bool {
		syncs := newBot.TimerSyncs()
		for _, s := range syncs {
			if s.Phase == "day" && s.RemainingMs > 0 {
				return true
			}
		}
		return false
	})

	// The remaining time should be less than the original 30s (since time has passed)
	syncs := newBot.TimerSyncs()
	for _, s := range syncs {
		if s.Phase == "day" && s.RemainingMs >= 30000 {
			t.Fatalf("reconnected bot got timer_sync with full time: %dms", s.RemainingMs)
		}
	}
}
