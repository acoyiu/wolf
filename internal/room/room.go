package room

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"sync"
	"time"
)

const (
	MinPlayers  = 4
	MaxPlayers  = 10
	IdleTimeout = 5 * time.Minute
	codeChars   = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
	codeLen     = 4
)

type Player struct {
	ID       string `json:"id"`
	Nickname string `json:"nickname"`
	IsHost   bool   `json:"isHost"`
}

type RoomState string

const (
	StateWaiting RoomState = "waiting"
	StatePlaying RoomState = "playing"
)

type Room struct {
	Code          string    `json:"roomCode"`
	TargetPlayers int       `json:"targetPlayers"`
	Players       []Player  `json:"players"`
	State         RoomState `json:"state"`
	HostID        string    `json:"hostId"`
	LastActivity  time.Time `json:"-"`
	mu            sync.RWMutex
}

type Manager struct {
	rooms map[string]*Room
	mu    sync.RWMutex
}

func NewManager() *Manager {
	m := &Manager{rooms: make(map[string]*Room)}
	go m.cleanupLoop()
	return m
}

func (m *Manager) CreateRoom(playerID, nickname string, targetPlayers int) (*Room, error) {
	if targetPlayers < MinPlayers || targetPlayers > MaxPlayers {
		return nil, fmt.Errorf("targetPlayers must be between %d and %d", MinPlayers, MaxPlayers)
	}
	code, err := m.generateUniqueCode()
	if err != nil {
		return nil, err
	}
	room := &Room{
		Code:          code,
		TargetPlayers: targetPlayers,
		Players:       []Player{{ID: playerID, Nickname: nickname, IsHost: true}},
		State:         StateWaiting,
		HostID:        playerID,
		LastActivity:  time.Now(),
	}
	m.mu.Lock()
	m.rooms[code] = room
	m.mu.Unlock()
	return room, nil
}

func (m *Manager) JoinRoom(code, playerID, nickname string) (*Room, error) {
	m.mu.RLock()
	room, ok := m.rooms[code]
	m.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("room_not_found")
	}
	room.mu.Lock()
	defer room.mu.Unlock()
	if room.State != StateWaiting {
		return nil, fmt.Errorf("game_already_started")
	}
	if len(room.Players) >= room.TargetPlayers {
		return nil, fmt.Errorf("room_full")
	}
	for _, p := range room.Players {
		if p.Nickname == nickname {
			return nil, fmt.Errorf("nickname_already_taken")
		}
	}
	room.Players = append(room.Players, Player{ID: playerID, Nickname: nickname, IsHost: false})
	room.LastActivity = time.Now()
	return room, nil
}

func (m *Manager) LeaveRoom(code, playerID string) (*Room, bool, error) {
	m.mu.RLock()
	room, ok := m.rooms[code]
	m.mu.RUnlock()
	if !ok {
		return nil, false, fmt.Errorf("room_not_found")
	}
	room.mu.Lock()
	defer room.mu.Unlock()
	if room.HostID == playerID {
		m.mu.Lock()
		delete(m.rooms, code)
		m.mu.Unlock()
		return room, true, nil
	}
	idx := -1
	for i, p := range room.Players {
		if p.ID == playerID {
			idx = i
			break
		}
	}
	if idx == -1 {
		return nil, false, fmt.Errorf("player_not_found")
	}
	room.Players = append(room.Players[:idx], room.Players[idx+1:]...)
	room.LastActivity = time.Now()
	return room, false, nil
}

func (m *Manager) StartGame(code, playerID string) (*Room, error) {
	m.mu.RLock()
	room, ok := m.rooms[code]
	m.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("room_not_found")
	}
	room.mu.Lock()
	defer room.mu.Unlock()
	if room.HostID != playerID {
		return nil, fmt.Errorf("host_only")
	}
	if len(room.Players) < room.TargetPlayers {
		return nil, fmt.Errorf("not_enough_players")
	}
	room.State = StatePlaying
	room.LastActivity = time.Now()
	return room, nil
}

func (m *Manager) GetRoom(code string) (*Room, bool) {
	m.mu.RLock()
	room, ok := m.rooms[code]
	m.mu.RUnlock()
	return room, ok
}

func (m *Manager) RemoveRoom(code string) {
	m.mu.Lock()
	delete(m.rooms, code)
	m.mu.Unlock()
}

func (m *Manager) HandleDisconnect(code, playerID string) (isHost bool) {
	m.mu.RLock()
	room, ok := m.rooms[code]
	m.mu.RUnlock()
	if !ok {
		return false
	}
	room.mu.RLock()
	isHost = room.HostID == playerID
	room.mu.RUnlock()
	if isHost {
		m.mu.Lock()
		delete(m.rooms, code)
		m.mu.Unlock()
	}
	return isHost
}

func (m *Manager) generateUniqueCode() (string, error) {
	for attempts := 0; attempts < 100; attempts++ {
		code := make([]byte, codeLen)
		for i := range code {
			idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(codeChars))))
			if err != nil {
				return "", err
			}
			code[i] = codeChars[idx.Int64()]
		}
		c := string(code)
		m.mu.RLock()
		_, exists := m.rooms[c]
		m.mu.RUnlock()
		if !exists {
			return c, nil
		}
	}
	return "", fmt.Errorf("failed to generate unique room code")
}

func (m *Manager) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		m.mu.Lock()
		now := time.Now()
		for code, room := range m.rooms {
			room.mu.RLock()
			idle := room.State != StatePlaying && now.Sub(room.LastActivity) > IdleTimeout
			room.mu.RUnlock()
			if idle {
				delete(m.rooms, code)
			}
		}
		m.mu.Unlock()
	}
}

func (r *Room) Snapshot() Room {
	r.mu.RLock()
	defer r.mu.RUnlock()
	players := make([]Player, len(r.Players))
	copy(players, r.Players)
	return Room{
		Code:          r.Code,
		TargetPlayers: r.TargetPlayers,
		Players:       players,
		State:         r.State,
		HostID:        r.HostID,
		LastActivity:  r.LastActivity,
	}
}

func (r *Room) PlayerIDs() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ids := make([]string, len(r.Players))
	for i, p := range r.Players {
		ids[i] = p.ID
	}
	return ids
}

func (r *Room) PlayerCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.Players)
}
