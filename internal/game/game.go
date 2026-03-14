package game

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"sync"
)

// Role represents a player's game-mechanical role.
type Role string

const (
	RoleSeer     Role = "seer"
	RoleWerewolf Role = "werewolf"
	RoleVillager Role = "villager"
)

// Phase represents the current game phase.
type Phase string

const (
	PhaseNightStep1 Phase = "night_step1"
	PhaseNightStep2 Phase = "night_step2"
	PhaseDay        Phase = "day"
	PhaseVote       Phase = "vote"
	PhaseResult     Phase = "result"
)

// VoteType indicates what kind of vote is happening.
type VoteType string

const (
	VoteGuessSeer VoteType = "guess_seer"
	VoteGuessWolf VoteType = "guess_wolf"
)

// OutMsg is a message to send to a player (To non-empty) or broadcast (To empty).
type OutMsg struct {
	To      string
	Type    string
	Payload map[string]interface{}
}

// Snapshot is a stable copy of current game state for reconnect/resume flows.
type Snapshot struct {
	RoomCode     string
	PlayerIDs    []string
	MayorID      string
	Roles        map[string]Role
	MayorSecret  Role
	Phase        Phase
	Candidates   []string
	Word         string
	Tokens       TokenPool
	TokenHistory []string
	WordGuessed  bool
	VoteType     VoteType
	Votes        map[string]string
	Winner       string
	Reason       string
}

// TokenPool tracks remaining tokens for the day phase.
type TokenPool struct {
	Yes     int `json:"yes"`
	No      int `json:"no"`
	Maybe   int `json:"maybe"`
	Close   int `json:"close"`
	Far     int `json:"far"`
	Correct int `json:"correct"`
}

func NewTokenPool() TokenPool {
	return TokenPool{Yes: 48, No: 48, Maybe: 1, Close: 1, Far: 1, Correct: 1}
}

func (t *TokenPool) Consume(token string) error {
	switch token {
	case "yes":
		if t.Yes <= 0 {
			return fmt.Errorf("token_exhausted")
		}
		t.Yes--
	case "no":
		if t.No <= 0 {
			return fmt.Errorf("token_exhausted")
		}
		t.No--
	case "maybe":
		if t.Maybe <= 0 {
			return fmt.Errorf("token_exhausted")
		}
		t.Maybe--
	case "close":
		if t.Close <= 0 {
			return fmt.Errorf("token_exhausted")
		}
		t.Close--
	case "far":
		if t.Far <= 0 {
			return fmt.Errorf("token_exhausted")
		}
		t.Far--
	case "correct":
		if t.Correct <= 0 {
			return fmt.Errorf("token_exhausted")
		}
		t.Correct--
	default:
		return fmt.Errorf("invalid_token")
	}
	return nil
}

// Game holds the full state of one game session.
type Game struct {
	RoomCode  string
	PlayerIDs []string
	MayorID   string

	Roles       map[string]Role
	MayorSecret Role

	Phase      Phase
	Candidates []string
	Word       string
	Confirmed  map[string]bool

	Tokens       TokenPool
	TokenHistory []string
	WordGuessed  bool

	VoteType VoteType
	Votes    map[string]string

	Winner string
	Reason string

	mu sync.Mutex
}

// WolfCount returns the number of werewolves for a given player count.
func WolfCount(n int) int {
	switch {
	case n <= 6:
		return 1
	case n <= 8:
		return 2
	default:
		return 3
	}
}

// NewGame creates a game with random role assignment.
func NewGame(roomCode string, playerIDs []string, mayorID string) (*Game, error) {
	n := len(playerIDs)
	if n < 4 {
		return nil, fmt.Errorf("not_enough_players")
	}
	wolves := WolfCount(n)
	pool := buildRolePool(n, wolves)
	shuffleRoles(pool)

	nonMayor := make([]string, 0, n-1)
	for _, id := range playerIDs {
		if id != mayorID {
			nonMayor = append(nonMayor, id)
		}
	}
	roles := make(map[string]Role, n-1)
	for i, id := range nonMayor {
		roles[id] = pool[i+1]
	}

	return &Game{
		RoomCode:    roomCode,
		PlayerIDs:   playerIDs,
		MayorID:     mayorID,
		Roles:       roles,
		MayorSecret: pool[0],
		Phase:       PhaseNightStep1,
		Confirmed:   make(map[string]bool),
		Tokens:      NewTokenPool(),
		Votes:       make(map[string]string),
	}, nil
}

// NewGameWithRoles creates a game with predetermined roles (for testing).
func NewGameWithRoles(roomCode string, playerIDs []string, mayorID string, mayorSecret Role, roles map[string]Role) *Game {
	return &Game{
		RoomCode:    roomCode,
		PlayerIDs:   playerIDs,
		MayorID:     mayorID,
		Roles:       roles,
		MayorSecret: mayorSecret,
		Phase:       PhaseNightStep1,
		Confirmed:   make(map[string]bool),
		Tokens:      NewTokenPool(),
		Votes:       make(map[string]string),
	}
}

func buildRolePool(n, wolves int) []Role {
	pool := make([]Role, 0, n)
	pool = append(pool, RoleSeer)
	for i := 0; i < wolves; i++ {
		pool = append(pool, RoleWerewolf)
	}
	for len(pool) < n {
		pool = append(pool, RoleVillager)
	}
	return pool
}

func shuffleRoles(roles []Role) {
	for i := len(roles) - 1; i > 0; i-- {
		j, _ := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		roles[i], roles[j.Int64()] = roles[j.Int64()], roles[i]
	}
}

// EffectiveRole returns the game-mechanical role of a player.
func (g *Game) EffectiveRole(playerID string) Role {
	if playerID == g.MayorID {
		return g.MayorSecret
	}
	return g.Roles[playerID]
}

func (g *Game) IsWerewolf(playerID string) bool {
	return g.EffectiveRole(playerID) == RoleWerewolf
}

func (g *Game) IsSeer(playerID string) bool {
	return g.EffectiveRole(playerID) == RoleSeer
}

// RoleMessages returns per-player role assignment messages.
func (g *Game) RoleMessages() []OutMsg {
	g.mu.Lock()
	defer g.mu.Unlock()

	msgs := make([]OutMsg, 0, len(g.PlayerIDs)+1)
	for _, id := range g.PlayerIDs {
		if id == g.MayorID {
			msgs = append(msgs, OutMsg{
				To:      id,
				Type:    "role_assigned",
				Payload: map[string]interface{}{"role": "mayor"},
			})
			msgs = append(msgs, OutMsg{
				To:      id,
				Type:    "mayor_secret",
				Payload: map[string]interface{}{"secretRole": string(g.MayorSecret)},
			})
		} else {
			msgs = append(msgs, OutMsg{
				To:      id,
				Type:    "role_assigned",
				Payload: map[string]interface{}{"role": string(g.Roles[id])},
			})
		}
	}
	return msgs
}

// StartNight initiates night step 1 and returns per-player messages.
func (g *Game) StartNight(candidates []string) []OutMsg {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.Candidates = candidates
	g.Phase = PhaseNightStep1
	g.Confirmed = make(map[string]bool)

	msgs := make([]OutMsg, 0, len(g.PlayerIDs))
	for _, id := range g.PlayerIDs {
		if id == g.MayorID {
			msgs = append(msgs, OutMsg{
				To:   id,
				Type: "night_step",
				Payload: map[string]interface{}{
					"step":       1,
					"candidates": candidates,
				},
			})
		} else {
			msgs = append(msgs, OutMsg{
				To:   id,
				Type: "night_step",
				Payload: map[string]interface{}{
					"step":    1,
					"message": "waiting",
				},
			})
		}
	}
	return msgs
}

// MayorPickWord is called when the mayor selects a word in night step 1.
func (g *Game) MayorPickWord(playerID, word string) ([]OutMsg, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if playerID != g.MayorID {
		return nil, fmt.Errorf("mayor_only")
	}
	if g.Phase != PhaseNightStep1 {
		return nil, fmt.Errorf("invalid_phase")
	}
	valid := false
	for _, c := range g.Candidates {
		if c == word {
			valid = true
			break
		}
	}
	if !valid {
		return nil, fmt.Errorf("invalid_word")
	}

	g.Word = word
	g.Confirmed[playerID] = true
	if len(g.Confirmed) == len(g.PlayerIDs) {
		return g.advanceNight(), nil
	}
	return nil, nil
}

// NightConfirm is called when a player taps confirm during the night.
func (g *Game) NightConfirm(playerID string) ([]OutMsg, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.Phase != PhaseNightStep1 && g.Phase != PhaseNightStep2 {
		return nil, fmt.Errorf("invalid_phase")
	}
	if g.Phase == PhaseNightStep1 && playerID == g.MayorID {
		return nil, fmt.Errorf("mayor_must_pick_word")
	}
	if g.Confirmed[playerID] {
		return nil, nil
	}

	g.Confirmed[playerID] = true
	if len(g.Confirmed) < len(g.PlayerIDs) {
		return nil, nil
	}
	return g.advanceNight(), nil
}

// advanceNight moves step1->step2 or step2->day. Caller must hold mu.
func (g *Game) advanceNight() []OutMsg {
	if g.Phase == PhaseNightStep1 {
		g.Phase = PhaseNightStep2
		g.Confirmed = make(map[string]bool)
		return g.nightStep2Messages()
	}
	g.Phase = PhaseDay
	g.Tokens = NewTokenPool()
	g.TokenHistory = nil
	return []OutMsg{{
		Type:    "phase_change",
		Payload: map[string]interface{}{"phase": "day"},
	}}
}

// nightStep2Messages builds per-player messages for night step 2.
func (g *Game) nightStep2Messages() []OutMsg {
	msgs := make([]OutMsg, 0, len(g.PlayerIDs))
	for _, id := range g.PlayerIDs {
		if id == g.MayorID {
			msgs = append(msgs, OutMsg{
				To:   id,
				Type: "night_step",
				Payload: map[string]interface{}{
					"step":    2,
					"message": "waiting",
				},
			})
		} else if g.IsWerewolf(id) || g.IsSeer(id) {
			msgs = append(msgs, OutMsg{
				To:   id,
				Type: "night_reveal",
				Payload: map[string]interface{}{
					"step": 2,
					"word": g.Word,
				},
			})
		} else {
			msgs = append(msgs, OutMsg{
				To:   id,
				Type: "night_step",
				Payload: map[string]interface{}{
					"step":    2,
					"message": "waiting",
				},
			})
		}
	}
	return msgs
}

// MayorToken processes a token used by the mayor during the day.
func (g *Game) MayorToken(playerID, token string) ([]OutMsg, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if playerID != g.MayorID {
		return nil, fmt.Errorf("mayor_only")
	}
	if g.Phase != PhaseDay {
		return nil, fmt.Errorf("invalid_phase")
	}
	if err := g.Tokens.Consume(token); err != nil {
		return nil, err
	}
	g.TokenHistory = append(g.TokenHistory, token)

	if token == "correct" {
		g.WordGuessed = true
		g.Phase = PhaseVote
		g.VoteType = VoteGuessSeer
		g.Votes = make(map[string]string)
		return []OutMsg{{
			Type:    "word_guessed",
			Payload: map[string]interface{}{},
		}}, nil
	}

	msgs := []OutMsg{{
		Type: "mayor_response",
		Payload: map[string]interface{}{
			"token":     token,
			"remaining": g.Tokens,
		},
	}}

	if g.Tokens.Yes == 0 && g.Tokens.No == 0 {
		g.Phase = PhaseVote
		g.VoteType = VoteGuessWolf
		g.Votes = make(map[string]string)
		msgs = append(msgs, OutMsg{
			Type:    "tokens_depleted",
			Payload: map[string]interface{}{},
		})
	}
	return msgs, nil
}

// DayTimeUp ends the day phase due to timer expiry.
func (g *Game) DayTimeUp() []OutMsg {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.Phase != PhaseDay {
		return nil
	}
	g.Phase = PhaseVote
	g.VoteType = VoteGuessWolf
	g.Votes = make(map[string]string)
	return []OutMsg{{
		Type:    "time_up",
		Payload: map[string]interface{}{},
	}}
}

func (g *Game) eligibleVoters() []string {
	if g.VoteType == VoteGuessSeer {
		var voters []string
		for _, id := range g.PlayerIDs {
			if g.IsWerewolf(id) {
				voters = append(voters, id)
			}
		}
		return voters
	}
	return g.PlayerIDs
}

func (g *Game) eligibleTargets(voterID string) []string {
	var targets []string
	for _, id := range g.PlayerIDs {
		if id != voterID {
			targets = append(targets, id)
		}
	}
	return targets
}

// CastVote records a player's vote.
func (g *Game) CastVote(voterID, targetID string) ([]OutMsg, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.Phase != PhaseVote {
		return nil, fmt.Errorf("invalid_phase")
	}
	eligible := false
	for _, v := range g.eligibleVoters() {
		if v == voterID {
			eligible = true
			break
		}
	}
	if !eligible {
		return nil, fmt.Errorf("not_eligible_voter")
	}
	if _, ok := g.Votes[voterID]; ok {
		return nil, fmt.Errorf("already_voted")
	}
	if targetID == voterID {
		return nil, fmt.Errorf("cannot_vote_self")
	}
	validTarget := false
	for _, id := range g.PlayerIDs {
		if id == targetID {
			validTarget = true
			break
		}
	}
	if !validTarget {
		return nil, fmt.Errorf("invalid_target")
	}

	g.Votes[voterID] = targetID
	msgs := []OutMsg{{
		Type: "vote_cast",
		Payload: map[string]interface{}{
			"voter":  voterID,
			"target": targetID,
		},
	}}

	if len(g.Votes) >= len(g.eligibleVoters()) {
		msgs = append(msgs, g.resolveVotes()...)
	}
	return msgs, nil
}

// VoteTimeUp handles vote timer expiry, assigning random votes for non-voters.
func (g *Game) VoteTimeUp() []OutMsg {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.Phase != PhaseVote {
		return nil
	}
	for _, v := range g.eligibleVoters() {
		if _, ok := g.Votes[v]; !ok {
			targets := g.eligibleTargets(v)
			if len(targets) > 0 {
				idx, _ := rand.Int(rand.Reader, big.NewInt(int64(len(targets))))
				g.Votes[v] = targets[idx.Int64()]
			}
		}
	}
	return g.resolveVotes()
}

func (g *Game) resolveVotes() []OutMsg {
	counts := make(map[string]int)
	for _, target := range g.Votes {
		counts[target]++
	}

	maxVotes := 0
	for _, c := range counts {
		if c > maxVotes {
			maxVotes = c
		}
	}
	var topVoted []string
	for target, c := range counts {
		if c == maxVotes {
			topVoted = append(topVoted, target)
		}
	}

	g.Phase = PhaseResult

	if len(topVoted) == 1 {
		target := topVoted[0]
		role := g.EffectiveRole(target)
		isCorrect := g.isVoteCorrect(role)
		g.setWinner(isCorrect)
		return []OutMsg{
			{
				Type: "vote_result",
				Payload: map[string]interface{}{
					"topVoted":     topVoted,
					"revealedRole": string(role),
					"isCorrect":    isCorrect,
				},
			},
			g.gameOverMsg(),
		}
	}

	// Tie: reveal all tied players
	revealedRoles := make([]map[string]string, len(topVoted))
	isCorrect := false
	for i, id := range topVoted {
		role := g.EffectiveRole(id)
		revealedRoles[i] = map[string]string{id: string(role)}
		if g.VoteType == VoteGuessSeer && role == RoleSeer {
			isCorrect = true
		}
		if g.VoteType == VoteGuessWolf && role == RoleWerewolf {
			isCorrect = true
		}
	}
	g.setWinner(isCorrect)

	return []OutMsg{
		{
			Type: "vote_result",
			Payload: map[string]interface{}{
				"topVoted":      topVoted,
				"revealedRoles": revealedRoles,
				"isCorrect":     isCorrect,
			},
		},
		g.gameOverMsg(),
	}
}

func (g *Game) isVoteCorrect(role Role) bool {
	if g.VoteType == VoteGuessSeer {
		return role == RoleSeer
	}
	return role == RoleWerewolf
}

func (g *Game) setWinner(isCorrect bool) {
	if g.WordGuessed {
		if isCorrect {
			g.Winner = "werewolves"
			g.Reason = "word_guessed_seer_found"
		} else {
			g.Winner = "villagers"
			g.Reason = "word_guessed_seer_safe"
		}
	} else {
		if isCorrect {
			g.Winner = "villagers"
			g.Reason = "word_missed_wolf_caught"
		} else {
			g.Winner = "werewolves"
			g.Reason = "word_missed_wolf_safe"
		}
	}
}

func (g *Game) gameOverMsg() OutMsg {
	roles := make(map[string]string, len(g.PlayerIDs))
	for _, id := range g.PlayerIDs {
		if id == g.MayorID {
			roles[id] = "mayor"
		} else {
			roles[id] = string(g.Roles[id])
		}
	}
	return OutMsg{
		Type: "game_over",
		Payload: map[string]interface{}{
			"winner":      g.Winner,
			"reason":      g.Reason,
			"word":        g.Word,
			"roles":       roles,
			"mayorSecret": string(g.MayorSecret),
		},
	}
}

// Abort ends the game abnormally.
func (g *Game) Abort(reason string) []OutMsg {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.Phase = PhaseResult
	return []OutMsg{{
		Type:    "game_aborted",
		Payload: map[string]interface{}{"reason": reason},
	}}
}

// Snapshot returns a thread-safe copy of game state.
func (g *Game) Snapshot() Snapshot {
	g.mu.Lock()
	defer g.mu.Unlock()

	roles := make(map[string]Role, len(g.Roles))
	for k, v := range g.Roles {
		roles[k] = v
	}

	votes := make(map[string]string, len(g.Votes))
	for k, v := range g.Votes {
		votes[k] = v
	}

	playerIDs := make([]string, len(g.PlayerIDs))
	copy(playerIDs, g.PlayerIDs)

	candidates := make([]string, len(g.Candidates))
	copy(candidates, g.Candidates)

	tokenHistory := make([]string, len(g.TokenHistory))
	copy(tokenHistory, g.TokenHistory)

	return Snapshot{
		RoomCode:     g.RoomCode,
		PlayerIDs:    playerIDs,
		MayorID:      g.MayorID,
		Roles:        roles,
		MayorSecret:  g.MayorSecret,
		Phase:        g.Phase,
		Candidates:   candidates,
		Word:         g.Word,
		Tokens:       g.Tokens,
		TokenHistory: tokenHistory,
		WordGuessed:  g.WordGuessed,
		VoteType:     g.VoteType,
		Votes:        votes,
		Winner:       g.Winner,
		Reason:       g.Reason,
	}
}
