package ws

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/wolfword/internal/game"
	"github.com/wolfword/internal/room"
	"github.com/wolfword/internal/wordlib"
)

const (
	defaultDifficulty = wordlib.Easy
)

type Envelope struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type OutEnvelope struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

type Client struct {
	ID       string
	Nickname string
	RoomCode string
	BaseURL  string
	Conn     *websocket.Conn
	Send     chan OutEnvelope
}

type Hub struct {
	upgrader websocket.Upgrader

	roomManager *room.Manager
	wordLibrary *wordlib.Library

	mu             sync.RWMutex
	games          map[string]*game.Game
	clients        map[string]*Client
	roomMembers    map[string]map[string]*Client
	roomDifficulty map[string]wordlib.Difficulty
	roomTimers     map[string]*roomTimer

	dayTimeout  time.Duration
	voteTimeout time.Duration
}

type roomTimer struct {
	dayCancel  chan struct{}
	voteCancel chan struct{}
}

func NewHub(dayTimeout, voteTimeout time.Duration) *Hub {
	return &Hub{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(_ *http.Request) bool { return true },
		},
		roomManager:    room.NewManager(),
		wordLibrary:    wordlib.New(),
		games:          make(map[string]*game.Game),
		clients:        make(map[string]*Client),
		roomMembers:    make(map[string]map[string]*Client),
		roomDifficulty: make(map[string]wordlib.Difficulty),
		roomTimers:     make(map[string]*roomTimer),
		dayTimeout:     dayTimeout,
		voteTimeout:    voteTimeout,
	}
}

func (h *Hub) HandleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	client := &Client{
		ID:      newID(),
		BaseURL: baseURLFromRequest(r),
		Conn:    conn,
		Send:    make(chan OutEnvelope, 32),
	}

	h.mu.Lock()
	h.clients[client.ID] = client
	h.mu.Unlock()

	go h.writeLoop(client)
	h.sendToClient(client, "connected", map[string]interface{}{"playerId": client.ID})
	h.readLoop(client)
}

func (h *Hub) readLoop(client *Client) {
	defer h.handleDisconnect(client)

	for {
		_, data, err := client.Conn.ReadMessage()
		if err != nil {
			return
		}

		var in Envelope
		if err := json.Unmarshal(data, &in); err != nil {
			h.sendError(client, "invalid_message")
			continue
		}

		h.dispatch(client, in)
	}
}

func (h *Hub) writeLoop(client *Client) {
	defer client.Conn.Close()
	for msg := range client.Send {
		if err := client.Conn.WriteJSON(msg); err != nil {
			return
		}
	}
}

func (h *Hub) dispatch(client *Client, in Envelope) {
	switch in.Type {
	case "create_room":
		h.handleCreateRoom(client, in.Payload)
	case "join_room":
		h.handleJoinRoom(client, in.Payload)
	case "leave_room":
		h.handleLeaveRoom(client)
	case "start_game":
		h.handleStartGame(client)
	case "night_pick_word":
		h.handleNightPickWord(client, in.Payload)
	case "night_confirm":
		h.handleNightConfirm(client)
	case "day_token":
		h.handleDayToken(client, in.Payload)
	case "vote_cast":
		h.handleVoteCast(client, in.Payload)
	case "ping":
		h.sendToClient(client, "pong", map[string]interface{}{})
	default:
		h.sendError(client, "unsupported_message_type")
	}
}

type createRoomPayload struct {
	Nickname      string `json:"nickname"`
	TargetPlayers int    `json:"targetPlayers"`
	Difficulty    string `json:"difficulty"`
}

func (h *Hub) handleCreateRoom(client *Client, raw json.RawMessage) {
	var p createRoomPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		h.sendError(client, "invalid_payload")
		return
	}
	p.Nickname = strings.TrimSpace(p.Nickname)
	if p.Nickname == "" {
		h.sendError(client, "invalid_nickname")
		return
	}

	rm, err := h.roomManager.CreateRoom(client.ID, p.Nickname, p.TargetPlayers)
	if err != nil {
		h.sendError(client, err.Error())
		return
	}

	difficulty := parseDifficulty(p.Difficulty)

	h.mu.Lock()
	client.Nickname = p.Nickname
	client.RoomCode = rm.Code
	if _, ok := h.roomMembers[rm.Code]; !ok {
		h.roomMembers[rm.Code] = make(map[string]*Client)
	}
	h.roomMembers[rm.Code][client.ID] = client
	h.roomDifficulty[rm.Code] = difficulty
	h.mu.Unlock()

	h.sendToClient(client, "room_created", map[string]interface{}{
		"roomCode":      rm.Code,
		"targetPlayers": rm.TargetPlayers,
		"players":       rm.Players,
		"difficulty":    string(difficulty),
		"joinUrl":       client.BaseURL + "/?roomCode=" + rm.Code,
	})
}

type joinRoomPayload struct {
	RoomCode string `json:"roomCode"`
	Nickname string `json:"nickname"`
}

func (h *Hub) handleJoinRoom(client *Client, raw json.RawMessage) {
	var p joinRoomPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		h.sendError(client, "invalid_payload")
		return
	}
	p.RoomCode = strings.ToUpper(strings.TrimSpace(p.RoomCode))
	p.Nickname = strings.TrimSpace(p.Nickname)
	if p.RoomCode == "" || p.Nickname == "" {
		h.sendError(client, "invalid_payload")
		return
	}

	rm, err := h.roomManager.JoinRoom(p.RoomCode, client.ID, p.Nickname)
	if err != nil {
		h.sendError(client, err.Error())
		return
	}

	h.mu.Lock()
	client.Nickname = p.Nickname
	client.RoomCode = p.RoomCode
	if _, ok := h.roomMembers[p.RoomCode]; !ok {
		h.roomMembers[p.RoomCode] = make(map[string]*Client)
	}
	h.roomMembers[p.RoomCode][client.ID] = client
	h.mu.Unlock()

	h.broadcastRoom(rm.Code, "player_joined", map[string]interface{}{
		"roomCode":      rm.Code,
		"targetPlayers": rm.TargetPlayers,
		"players":       rm.Players,
		"canStart":      len(rm.Players) >= rm.TargetPlayers,
	})
}

func (h *Hub) handleLeaveRoom(client *Client) {
	code := client.RoomCode
	if code == "" {
		h.sendError(client, "room_not_found")
		return
	}

	rm, hostLeft, err := h.roomManager.LeaveRoom(code, client.ID)
	if err != nil {
		h.sendError(client, err.Error())
		return
	}

	if hostLeft {
		h.closeRoom(code, "host_disconnected")
		return
	}

	h.mu.Lock()
	if members, ok := h.roomMembers[code]; ok {
		delete(members, client.ID)
		if len(members) == 0 {
			delete(h.roomMembers, code)
		}
	}
	client.RoomCode = ""
	h.mu.Unlock()

	h.broadcastRoom(code, "player_left", map[string]interface{}{
		"players": rm.Players,
	})
}

func (h *Hub) handleStartGame(client *Client) {
	code := client.RoomCode
	if code == "" {
		h.sendError(client, "room_not_found")
		return
	}

	rm, err := h.roomManager.StartGame(code, client.ID)
	if err != nil {
		h.sendError(client, err.Error())
		return
	}

	g, err := game.NewGame(code, rm.PlayerIDs(), rm.HostID)
	if err != nil {
		h.sendError(client, err.Error())
		return
	}

	h.mu.Lock()
	h.games[code] = g
	h.ensureTimers(code)
	difficulty := h.roomDifficulty[code]
	h.mu.Unlock()

	for _, msg := range g.RoleMessages() {
		h.deliverGameOut(code, msg)
	}

	if difficulty == "" {
		difficulty = defaultDifficulty
	}
	candidates, err := h.wordLibrary.GetCandidates(difficulty, 3)
	if err != nil {
		h.sendError(client, "word_library_unavailable")
		return
	}
	for _, msg := range g.StartNight(candidates) {
		h.deliverGameOut(code, msg)
	}
}

type nightPickPayload struct {
	Word string `json:"word"`
}

func (h *Hub) handleNightPickWord(client *Client, raw json.RawMessage) {
	g, code := h.currentGameForClient(client)
	if g == nil {
		h.sendError(client, "game_not_found")
		return
	}
	var p nightPickPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		h.sendError(client, "invalid_payload")
		return
	}

	out, err := g.MayorPickWord(client.ID, strings.TrimSpace(p.Word))
	if err != nil {
		h.sendError(client, err.Error())
		return
	}
	h.deliverGameOutBatch(code, out)
	h.scheduleIfPhaseTransitioned(code, g)
}

func (h *Hub) handleNightConfirm(client *Client) {
	g, code := h.currentGameForClient(client)
	if g == nil {
		h.sendError(client, "game_not_found")
		return
	}

	out, err := g.NightConfirm(client.ID)
	if err != nil {
		h.sendError(client, err.Error())
		return
	}
	h.deliverGameOutBatch(code, out)
	h.scheduleIfPhaseTransitioned(code, g)
}

type dayTokenPayload struct {
	Token string `json:"token"`
}

func (h *Hub) handleDayToken(client *Client, raw json.RawMessage) {
	g, code := h.currentGameForClient(client)
	if g == nil {
		h.sendError(client, "game_not_found")
		return
	}
	var p dayTokenPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		h.sendError(client, "invalid_payload")
		return
	}

	out, err := g.MayorToken(client.ID, strings.TrimSpace(p.Token))
	if err != nil {
		h.sendError(client, err.Error())
		return
	}
	h.deliverGameOutBatch(code, out)
	h.scheduleIfPhaseTransitioned(code, g)
}

type voteCastPayload struct {
	Target string `json:"target"`
}

func (h *Hub) handleVoteCast(client *Client, raw json.RawMessage) {
	g, code := h.currentGameForClient(client)
	if g == nil {
		h.sendError(client, "game_not_found")
		return
	}
	var p voteCastPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		h.sendError(client, "invalid_payload")
		return
	}
	out, err := g.CastVote(client.ID, strings.TrimSpace(p.Target))
	if err != nil {
		h.sendError(client, err.Error())
		return
	}
	h.deliverGameOutBatch(code, out)
	h.scheduleIfPhaseTransitioned(code, g)
}

func (h *Hub) scheduleIfPhaseTransitioned(code string, g *game.Game) {
	switch g.Phase {
	case game.PhaseDay:
		h.startDayTimer(code, g)
	case game.PhaseVote:
		h.startVoteTimer(code, g)
	case game.PhaseResult:
		h.scheduleRoomClose(code, "game_ended", 10*time.Second)
	}
}

func (h *Hub) currentGameForClient(client *Client) (*game.Game, string) {
	code := client.RoomCode
	if code == "" {
		return nil, ""
	}
	h.mu.RLock()
	g := h.games[code]
	h.mu.RUnlock()
	return g, code
}

func (h *Hub) deliverGameOutBatch(code string, out []game.OutMsg) {
	for _, m := range out {
		h.deliverGameOut(code, m)
	}
}

func (h *Hub) deliverGameOut(code string, m game.OutMsg) {
	if m.Type == "" {
		return
	}
	if m.To == "" {
		h.broadcastRoom(code, m.Type, m.Payload)
		return
	}
	h.mu.RLock()
	c := h.clients[m.To]
	h.mu.RUnlock()
	if c != nil {
		h.sendToClient(c, m.Type, m.Payload)
	}
}

func (h *Hub) sendToClient(client *Client, typ string, payload interface{}) {
	select {
	case client.Send <- OutEnvelope{Type: typ, Payload: payload}:
	default:
		// Drop message if the send buffer is full to avoid blocking the hub.
	}
}

func (h *Hub) sendError(client *Client, message string) {
	h.sendToClient(client, "error", map[string]interface{}{"message": message})
}

func (h *Hub) broadcastRoom(code, typ string, payload interface{}) {
	h.mu.RLock()
	members := h.roomMembers[code]
	clients := make([]*Client, 0, len(members))
	for _, c := range members {
		clients = append(clients, c)
	}
	h.mu.RUnlock()
	for _, c := range clients {
		h.sendToClient(c, typ, payload)
	}
}

func (h *Hub) handleDisconnect(client *Client) {
	code := client.RoomCode

	h.mu.Lock()
	delete(h.clients, client.ID)
	if code != "" {
		if members, ok := h.roomMembers[code]; ok {
			delete(members, client.ID)
			if len(members) == 0 {
				delete(h.roomMembers, code)
			}
		}
	}
	h.mu.Unlock()

	close(client.Send)

	if code == "" {
		return
	}

	rm, ok := h.roomManager.GetRoom(code)
	if !ok {
		return
	}
	if rm.HostID == client.ID {
		h.closeRoom(code, "host_disconnected")
		return
	}

	h.mu.RLock()
	g := h.games[code]
	h.mu.RUnlock()
	if g != nil && g.Phase != game.PhaseResult {
		for _, msg := range g.Abort("player_disconnected") {
			h.deliverGameOut(code, msg)
		}
		h.scheduleRoomClose(code, "game_ended", 2*time.Second)
		return
	}

	leftRoom, hostLeft, err := h.roomManager.LeaveRoom(code, client.ID)
	if err == nil && !hostLeft {
		h.broadcastRoom(code, "player_left", map[string]interface{}{"players": leftRoom.Players})
	}
}

func (h *Hub) closeRoom(code, reason string) {
	h.stopTimers(code)
	h.roomManager.RemoveRoom(code)

	h.mu.Lock()
	members := h.roomMembers[code]
	delete(h.roomMembers, code)
	delete(h.games, code)
	delete(h.roomDifficulty, code)
	h.mu.Unlock()

	for _, c := range members {
		h.sendToClient(c, "room_closed", map[string]interface{}{"reason": reason})
		_ = c.Conn.Close()
	}
}

func (h *Hub) scheduleRoomClose(code, reason string, delay time.Duration) {
	go func() {
		t := time.NewTimer(delay)
		defer t.Stop()
		<-t.C
		h.closeRoom(code, reason)
	}()
}

func (h *Hub) ensureTimers(code string) {
	if _, ok := h.roomTimers[code]; !ok {
		h.roomTimers[code] = &roomTimer{}
	}
}

func (h *Hub) stopTimers(code string) {
	h.mu.Lock()
	rt := h.roomTimers[code]
	delete(h.roomTimers, code)
	h.mu.Unlock()
	if rt == nil {
		return
	}
	if rt.dayCancel != nil {
		close(rt.dayCancel)
	}
	if rt.voteCancel != nil {
		close(rt.voteCancel)
	}
}

func (h *Hub) startDayTimer(code string, g *game.Game) {
	h.mu.Lock()
	rt := h.roomTimers[code]
	if rt == nil {
		rt = &roomTimer{}
		h.roomTimers[code] = rt
	}
	if rt.dayCancel != nil {
		h.mu.Unlock()
		return
	}
	rt.dayCancel = make(chan struct{})
	cancel := rt.dayCancel
	h.mu.Unlock()

	go func() {
		t := time.NewTimer(h.dayTimeout)
		defer t.Stop()
		select {
		case <-t.C:
			out := g.DayTimeUp()
			h.deliverGameOutBatch(code, out)
			h.scheduleIfPhaseTransitioned(code, g)
		case <-cancel:
		}
	}()
}

func (h *Hub) startVoteTimer(code string, g *game.Game) {
	h.mu.Lock()
	rt := h.roomTimers[code]
	if rt == nil {
		rt = &roomTimer{}
		h.roomTimers[code] = rt
	}
	if rt.dayCancel != nil {
		close(rt.dayCancel)
		rt.dayCancel = nil
	}
	if rt.voteCancel != nil {
		h.mu.Unlock()
		return
	}
	rt.voteCancel = make(chan struct{})
	cancel := rt.voteCancel
	h.mu.Unlock()

	go func() {
		t := time.NewTimer(h.voteTimeout)
		defer t.Stop()
		select {
		case <-t.C:
			out := g.VoteTimeUp()
			h.deliverGameOutBatch(code, out)
			h.scheduleIfPhaseTransitioned(code, g)
		case <-cancel:
		}
	}()
}

func parseDifficulty(s string) wordlib.Difficulty {
	s = strings.ToLower(strings.TrimSpace(s))
	switch wordlib.Difficulty(s) {
	case wordlib.Easy, wordlib.Medium, wordlib.Hard:
		return wordlib.Difficulty(s)
	default:
		return defaultDifficulty
	}
}

func baseURLFromRequest(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if xfp := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")); xfp != "" {
		scheme = xfp
	}
	return scheme + "://" + r.Host
}

func newID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return hex.EncodeToString([]byte(time.Now().Format("150405.000")))
	}
	return hex.EncodeToString(b)
}
