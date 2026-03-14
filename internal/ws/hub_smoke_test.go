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
	lastError   string
	readErr     error
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

	waitUntil(t, 3*time.Second, "connected message", func() bool {
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
	case "game_over":
		b.phase = "result"
		b.gameOver = true
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

func (b *wsBot) LastError() string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.lastError
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

func TestHubSmokeFourPlayersNoBrowsers(t *testing.T) {
	h := NewHub(5*time.Second, 5*time.Second)
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", h.HandleWS)

	server := httptest.NewServer(mux)
	defer server.Close()

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

	waitUntil(t, 3*time.Second, "host room created", func() bool {
		return host.RoomCode() != ""
	})
	roomCode := host.RoomCode()

	for i := 1; i < len(bots); i++ {
		bots[i].Send(t, "join_room", map[string]interface{}{
			"roomCode": roomCode,
			"nickname": bots[i].nickname,
		})
	}

	waitUntil(t, 3*time.Second, "all players joined", func() bool {
		for _, b := range bots {
			if b.RoomCode() != roomCode || b.PlayersCount() != 4 {
				return false
			}
		}
		return true
	})

	host.Send(t, "start_game", map[string]interface{}{})

	waitUntil(t, 3*time.Second, "roles assigned", func() bool {
		for _, b := range bots {
			if b.Role() == "" {
				return false
			}
		}
		return true
	})

	waitUntil(t, 3*time.Second, "mayor candidate words", func() bool {
		return host.NightStep() == 1 && len(host.Candidates()) >= 1
	})

	chosenWord := host.Candidates()[0]
	host.Send(t, "night_pick_word", map[string]interface{}{"word": chosenWord})

	for i := 1; i < len(bots); i++ {
		bots[i].Send(t, "night_confirm", map[string]interface{}{})
	}

	waitUntil(t, 3*time.Second, "night step 2 started", func() bool {
		return host.NightStep() == 2 || host.Phase() == "day"
	})

	for i := 0; i < len(bots); i++ {
		bots[i].Send(t, "night_confirm", map[string]interface{}{})
	}

	waitUntil(t, 3*time.Second, "day phase started", func() bool {
		for _, b := range bots {
			if b.Phase() != "day" {
				return false
			}
		}
		return true
	})

	host.Send(t, "day_token", map[string]interface{}{"token": "correct"})

	waitUntil(t, 3*time.Second, "vote phase started", func() bool {
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

	waitUntil(t, 3*time.Second, "game over", func() bool {
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
