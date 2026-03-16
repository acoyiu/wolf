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
	defaultDifficulty     = wordlib.Easy
	defaultReconnectGrace = 20 * time.Second
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

	once   sync.Once
	closed chan struct{}
}

type Hub struct {
	upgrader websocket.Upgrader

	roomManager *room.Manager
	wordLibrary *wordlib.Library

	mu                sync.RWMutex
	games             map[string]*game.Game
	clients           map[string]*Client
	roomMembers       map[string]map[string]*Client
	roomDifficulty    map[string]wordlib.Difficulty
	roomTimers        map[string]*roomTimer
	pendingReconnects map[string]*pendingReconnect

	dayTimeout     time.Duration
	voteTimeout    time.Duration
	reconnectGrace time.Duration
}

type roomTimer struct {
	dayCancel  chan struct{}
	voteCancel chan struct{}
	dayStart   time.Time
	voteStart  time.Time
}

type pendingReconnect struct {
	RoomCode string
	Nickname string
	timer    *time.Timer
}

func NewHub(dayTimeout, voteTimeout time.Duration) *Hub {
	return &Hub{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(_ *http.Request) bool { return true },
		},
		roomManager:       room.NewManager(),
		wordLibrary:       wordlib.New(),
		games:             make(map[string]*game.Game),
		clients:           make(map[string]*Client),
		roomMembers:       make(map[string]map[string]*Client),
		roomDifficulty:    make(map[string]wordlib.Difficulty),
		roomTimers:        make(map[string]*roomTimer),
		pendingReconnects: make(map[string]*pendingReconnect),
		dayTimeout:        dayTimeout,
		voteTimeout:       voteTimeout,
		reconnectGrace:    defaultReconnectGrace,
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
		closed:  make(chan struct{}),
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
	for {
		select {
		case msg, ok := <-client.Send:
			if !ok {
				return
			}
			if err := client.Conn.WriteJSON(msg); err != nil {
				return
			}
		case <-client.closed:
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
	case "resume_session":
		h.handleResumeSession(client, in.Payload)
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
	case "play_again":
		h.handlePlayAgain(client)
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
	snap := rm.Snapshot()

	difficulty := parseDifficulty(p.Difficulty)

	h.mu.Lock()
	client.Nickname = p.Nickname
	client.RoomCode = snap.Code
	if _, ok := h.roomMembers[snap.Code]; !ok {
		h.roomMembers[snap.Code] = make(map[string]*Client)
	}
	h.roomMembers[snap.Code][client.ID] = client
	h.roomDifficulty[snap.Code] = difficulty
	h.mu.Unlock()

	h.sendToClient(client, "room_created", map[string]interface{}{
		"roomCode":      snap.Code,
		"targetPlayers": snap.TargetPlayers,
		"players":       snap.Players,
		"difficulty":    string(difficulty),
		"joinUrl":       client.BaseURL + "/?roomCode=" + snap.Code,
	})
}

type joinRoomPayload struct {
	RoomCode string `json:"roomCode"`
	Nickname string `json:"nickname"`
}

type resumeSessionPayload struct {
	PlayerID string `json:"playerId"`
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
	snap := rm.Snapshot()

	h.mu.Lock()
	client.Nickname = p.Nickname
	client.RoomCode = p.RoomCode
	if _, ok := h.roomMembers[p.RoomCode]; !ok {
		h.roomMembers[p.RoomCode] = make(map[string]*Client)
	}
	h.roomMembers[p.RoomCode][client.ID] = client
	h.mu.Unlock()

	h.broadcastRoom(snap.Code, "player_joined", map[string]interface{}{
		"roomCode":      snap.Code,
		"targetPlayers": snap.TargetPlayers,
		"players":       snap.Players,
		"canStart":      len(snap.Players) >= snap.TargetPlayers,
	})
}

func (h *Hub) handleResumeSession(client *Client, raw json.RawMessage) {
	var p resumeSessionPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		h.sendError(client, "invalid_payload")
		return
	}
	p.PlayerID = strings.TrimSpace(p.PlayerID)
	p.RoomCode = strings.ToUpper(strings.TrimSpace(p.RoomCode))
	p.Nickname = strings.TrimSpace(p.Nickname)
	if p.PlayerID == "" || p.RoomCode == "" {
		h.sendError(client, "invalid_payload")
		return
	}

	h.mu.Lock()
	rm, ok := h.roomManager.GetRoom(p.RoomCode)
	if !ok {
		h.mu.Unlock()
		h.sendError(client, "resume_room_not_found")
		return
	}
	if !roomHasPlayer(rm, p.PlayerID) {
		h.mu.Unlock()
		h.sendError(client, "resume_player_not_found")
		return
	}

	pending, hasPending := h.pendingReconnects[p.PlayerID]
	if hasPending && pending.RoomCode != p.RoomCode {
		h.mu.Unlock()
		h.sendError(client, "resume_not_available")
		return
	}
	var oldConn *Client
	if !hasPending {
		if existing := h.clients[p.PlayerID]; existing != nil {
			oldConn = existing
		} else {
			g := h.games[p.RoomCode]
			if g != nil && g.Snapshot().Phase == game.PhaseResult {
				h.mu.Unlock()
				h.sendError(client, "resume_not_available")
				return
			}
		}
	}
	delete(h.pendingReconnects, p.PlayerID)
	delete(h.clients, client.ID)

	client.ID = p.PlayerID
	if hasPending && pending.Nickname != "" {
		client.Nickname = pending.Nickname
	} else {
		client.Nickname = p.Nickname
	}
	client.RoomCode = p.RoomCode
	h.clients[client.ID] = client
	if _, ok := h.roomMembers[p.RoomCode]; !ok {
		h.roomMembers[p.RoomCode] = make(map[string]*Client)
	}
	h.roomMembers[p.RoomCode][client.ID] = client
	h.mu.Unlock()

	if oldConn != nil {
		oldConn.once.Do(func() {
			close(oldConn.closed)
		})
		_ = oldConn.Conn.Close()
	}
	if hasPending && pending.timer != nil {
		pending.timer.Stop()
	}

	h.sendToClient(client, "session_resumed", map[string]interface{}{
		"playerId": client.ID,
		"roomCode": client.RoomCode,
	})
	h.sendRoomState(client, rm)
	h.sendGameState(client)

	h.broadcastRoom(p.RoomCode, "player_reconnected", map[string]interface{}{
		"playerId": client.ID,
	})
}

func (h *Hub) handleLeaveRoom(client *Client) {
	code := client.RoomCode
	if code == "" {
		h.sendError(client, "room_not_found")
		return
	}
	h.cancelPendingReconnect(client.ID)

	rm, hostLeft, err := h.roomManager.LeaveRoom(code, client.ID)
	if err != nil {
		h.sendError(client, err.Error())
		return
	}

	if hostLeft {
		h.closeRoom(code, "host_disconnected")
		return
	}
	snap := rm.Snapshot()

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
		"players": snap.Players,
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
	snap := rm.Snapshot()

	playerIDs := make([]string, len(snap.Players))
	for i, p := range snap.Players {
		playerIDs[i] = p.ID
	}
	g, err := game.NewGame(code, playerIDs, snap.HostID)
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

func (h *Hub) handlePlayAgain(client *Client) {
	code := client.RoomCode
	if code == "" {
		h.sendError(client, "room_not_found")
		return
	}
	h.mu.RLock()
	g := h.games[code]
	h.mu.RUnlock()
	if g == nil || g.Snapshot().Phase != game.PhaseResult {
		h.sendError(client, "invalid_phase")
		return
	}

	rm, err := h.roomManager.ResetToWaiting(code, client.ID)
	if err != nil {
		h.sendError(client, err.Error())
		return
	}
	snap := rm.Snapshot()

	h.mu.Lock()
	delete(h.games, code)
	h.mu.Unlock()

	h.broadcastRoom(code, "room_state", map[string]interface{}{
		"roomCode":      snap.Code,
		"targetPlayers": snap.TargetPlayers,
		"players":       snap.Players,
		"canStart":      len(snap.Players) >= snap.TargetPlayers,
	})
}

func (h *Hub) scheduleIfPhaseTransitioned(code string, g *game.Game) {
	switch g.Snapshot().Phase {
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
	case <-client.closed:
		return
	default:
	}
	select {
	case client.Send <- OutEnvelope{Type: typ, Payload: payload}:
	case <-client.closed:
	default:
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
	// Only remove from maps if this client is still the registered one.
	// A resumed session may have already replaced this client with a new
	// connection under the same player ID.
	replaced := false
	if current := h.clients[client.ID]; current == client {
		delete(h.clients, client.ID)
		if code != "" {
			if members, ok := h.roomMembers[code]; ok {
				delete(members, client.ID)
				if len(members) == 0 {
					delete(h.roomMembers, code)
				}
			}
		}
	} else {
		replaced = true
	}
	h.mu.Unlock()

	client.once.Do(func() {
		close(client.closed)
	})

	if code == "" || replaced {
		return
	}

	h.mu.RLock()
	g := h.games[code]
	h.mu.RUnlock()
	if g != nil && g.Snapshot().Phase != game.PhaseResult {
		h.beginReconnectGrace(client.ID, client.Nickname, code)
		h.broadcastRoom(code, "player_reconnecting", map[string]interface{}{"playerId": client.ID, "nickname": client.Nickname})
		return
	}

	if g != nil {
		return
	}

	if _, ok := h.roomManager.GetRoom(code); ok {
		h.beginReconnectGrace(client.ID, client.Nickname, code)
		h.broadcastRoom(code, "player_reconnecting", map[string]interface{}{"playerId": client.ID, "nickname": client.Nickname})
	}
}

func (h *Hub) beginReconnectGrace(playerID, nickname, roomCode string) {
	h.mu.Lock()
	if existing := h.pendingReconnects[playerID]; existing != nil && existing.timer != nil {
		existing.timer.Stop()
	}
	p := &pendingReconnect{RoomCode: roomCode, Nickname: nickname}
	p.timer = time.AfterFunc(h.reconnectGrace, func() {
		h.expireReconnectGrace(playerID, roomCode)
	})
	h.pendingReconnects[playerID] = p
	h.mu.Unlock()
}

func (h *Hub) expireReconnectGrace(playerID, roomCode string) {
	h.mu.Lock()
	pending, ok := h.pendingReconnects[playerID]
	if !ok || pending.RoomCode != roomCode {
		h.mu.Unlock()
		return
	}
	delete(h.pendingReconnects, playerID)
	h.mu.Unlock()

	h.mu.RLock()
	g := h.games[roomCode]
	h.mu.RUnlock()

	if g != nil && g.Snapshot().Phase != game.PhaseResult {
		// Active game: abort and close room.
		for _, msg := range g.Abort("player_disconnected") {
			h.deliverGameOut(roomCode, msg)
		}
		h.scheduleRoomClose(roomCode, "game_ended", 2*time.Second)
		return
	}

	if g != nil {
		// Result phase: nothing to do.
		return
	}

	rm, ok := h.roomManager.GetRoom(roomCode)
	if !ok {
		return
	}
	rmSnap := rm.Snapshot()
	if rmSnap.HostID == playerID {
		h.closeRoom(roomCode, "host_disconnected")
		return
	}
	leftRoom, hostLeft, err := h.roomManager.LeaveRoom(roomCode, playerID)
	if err == nil && !hostLeft {
		leftSnap := leftRoom.Snapshot()
		h.broadcastRoom(roomCode, "player_left", map[string]interface{}{"players": leftSnap.Players})
	}
}

func (h *Hub) cancelPendingReconnect(playerID string) {
	h.mu.Lock()
	pending := h.pendingReconnects[playerID]
	delete(h.pendingReconnects, playerID)
	h.mu.Unlock()
	if pending != nil && pending.timer != nil {
		pending.timer.Stop()
	}
}

func (h *Hub) sendRoomState(client *Client, rm *room.Room) {
	snap := rm.Snapshot()
	h.sendToClient(client, "room_state", map[string]interface{}{
		"roomCode":      snap.Code,
		"targetPlayers": snap.TargetPlayers,
		"players":       snap.Players,
		"canStart":      len(snap.Players) >= snap.TargetPlayers,
	})
}

func (h *Hub) sendGameState(client *Client) {
	h.mu.RLock()
	g := h.games[client.RoomCode]
	h.mu.RUnlock()
	if g == nil {
		return
	}
	s := g.Snapshot()
	role := roleForPlayer(s, client.ID)

	if client.ID == s.MayorID {
		h.sendToClient(client, "role_assigned", map[string]interface{}{"role": "mayor"})
		h.sendToClient(client, "mayor_secret", map[string]interface{}{"secretRole": string(s.MayorSecret)})
	} else {
		h.sendToClient(client, "role_assigned", map[string]interface{}{"role": string(role)})
	}

	switch s.Phase {
	case game.PhaseNightStep1:
		if client.ID == s.MayorID {
			h.sendToClient(client, "night_step", map[string]interface{}{"step": 1, "candidates": s.Candidates})
		} else {
			h.sendToClient(client, "night_step", map[string]interface{}{"step": 1, "message": "waiting"})
		}
	case game.PhaseNightStep2:
		if client.ID == s.MayorID {
			h.sendToClient(client, "night_step", map[string]interface{}{"step": 2, "message": "waiting"})
		} else if role == game.RoleSeer || role == game.RoleWerewolf {
			h.sendToClient(client, "night_reveal", map[string]interface{}{"step": 2, "word": s.Word})
		} else {
			h.sendToClient(client, "night_step", map[string]interface{}{"step": 2, "message": "waiting"})
		}
	case game.PhaseDay:
		h.sendToClient(client, "phase_change", map[string]interface{}{"phase": "day"})
		h.sendToClient(client, "day_state", map[string]interface{}{"remaining": s.Tokens, "history": s.TokenHistory})
		h.mu.RLock()
		rt := h.roomTimers[client.RoomCode]
		h.mu.RUnlock()
		if rt != nil && !rt.dayStart.IsZero() {
			remaining := h.dayTimeout - time.Since(rt.dayStart)
			if remaining < 0 {
				remaining = 0
			}
			h.sendToClient(client, "timer_sync", map[string]interface{}{
				"phase":       "day",
				"remainingMs": remaining.Milliseconds(),
			})
		}
	case game.PhaseVote:
		payload := map[string]interface{}{"voteType": string(s.VoteType)}
		if votedFor, ok := s.Votes[client.ID]; ok {
			payload["votedFor"] = votedFor
		}
		h.sendToClient(client, "vote_state", payload)
		h.mu.RLock()
		rt := h.roomTimers[client.RoomCode]
		h.mu.RUnlock()
		if rt != nil && !rt.voteStart.IsZero() {
			remaining := h.voteTimeout - time.Since(rt.voteStart)
			if remaining < 0 {
				remaining = 0
			}
			h.sendToClient(client, "timer_sync", map[string]interface{}{
				"phase":       "vote",
				"remainingMs": remaining.Milliseconds(),
			})
		}
	case game.PhaseResult:
		h.sendToClient(client, "game_over", map[string]interface{}{
			"winner":      s.Winner,
			"reason":      s.Reason,
			"word":        s.Word,
			"roles":       roleMapForGameOver(s),
			"mayorSecret": string(s.MayorSecret),
			"votes":       s.Votes,
		})
	}
}

func roleForPlayer(s game.Snapshot, playerID string) game.Role {
	if playerID == s.MayorID {
		return s.MayorSecret
	}
	return s.Roles[playerID]
}

func roleMapForGameOver(s game.Snapshot) map[string]string {
	roles := make(map[string]string, len(s.PlayerIDs))
	for _, id := range s.PlayerIDs {
		if id == s.MayorID {
			roles[id] = "mayor"
			continue
		}
		roles[id] = string(s.Roles[id])
	}
	return roles
}

func roomHasPlayer(rm *room.Room, playerID string) bool {
	snap := rm.Snapshot()
	for _, p := range snap.Players {
		if p.ID == playerID {
			return true
		}
	}
	return false
}

func (h *Hub) closeRoom(code, reason string) {
	h.stopTimers(code)
	h.roomManager.RemoveRoom(code)

	h.mu.Lock()
	members := h.roomMembers[code]
	var pendingToStop []*pendingReconnect
	for playerID, pending := range h.pendingReconnects {
		if pending.RoomCode == code {
			delete(h.pendingReconnects, playerID)
			pendingToStop = append(pendingToStop, pending)
		}
	}
	delete(h.roomMembers, code)
	delete(h.games, code)
	delete(h.roomDifficulty, code)
	h.mu.Unlock()

	for _, pending := range pendingToStop {
		if pending.timer != nil {
			pending.timer.Stop()
		}
	}

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
	timeout := h.dayTimeout
	rt.dayStart = time.Now()
	h.mu.Unlock()

	h.broadcastRoom(code, "timer_sync", map[string]interface{}{
		"phase":       "day",
		"remainingMs": timeout.Milliseconds(),
	})

	go func() {
		deadline := time.NewTimer(timeout)
		tick := time.NewTicker(1 * time.Second)
		defer deadline.Stop()
		defer tick.Stop()
		start := time.Now()
		for {
			select {
			case <-deadline.C:
				out := g.DayTimeUp()
				h.deliverGameOutBatch(code, out)
				h.scheduleIfPhaseTransitioned(code, g)
				return
			case <-tick.C:
				elapsed := time.Since(start)
				remaining := timeout - elapsed
				if remaining < 0 {
					remaining = 0
				}
				h.broadcastRoom(code, "timer_sync", map[string]interface{}{
					"phase":       "day",
					"remainingMs": remaining.Milliseconds(),
				})
			case <-cancel:
				return
			}
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
	timeout := h.voteTimeout
	rt.voteStart = time.Now()
	h.mu.Unlock()

	h.broadcastRoom(code, "timer_sync", map[string]interface{}{
		"phase":       "vote",
		"remainingMs": timeout.Milliseconds(),
	})

	go func() {
		deadline := time.NewTimer(timeout)
		tick := time.NewTicker(1 * time.Second)
		defer deadline.Stop()
		defer tick.Stop()
		start := time.Now()
		for {
			select {
			case <-deadline.C:
				out := g.VoteTimeUp()
				h.deliverGameOutBatch(code, out)
				h.scheduleIfPhaseTransitioned(code, g)
				return
			case <-tick.C:
				elapsed := time.Since(start)
				remaining := timeout - elapsed
				if remaining < 0 {
					remaining = 0
				}
				h.broadcastRoom(code, "timer_sync", map[string]interface{}{
					"phase":       "vote",
					"remainingMs": remaining.Milliseconds(),
				})
			case <-cancel:
				return
			}
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
