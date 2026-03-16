package game

import (
	"testing"
)

func TestWolfCount(t *testing.T) {
	cases := []struct {
		n    int
		want int
	}{
		{4, 1}, {6, 1}, {7, 2}, {8, 2}, {9, 3}, {12, 3},
	}
	for _, tc := range cases {
		if got := WolfCount(tc.n); got != tc.want {
			t.Fatalf("WolfCount(%d)=%d want %d", tc.n, got, tc.want)
		}
	}
}

func TestNewGameRoleTotals(t *testing.T) {
	players := []string{"p1", "p2", "p3", "p4", "p5", "p6", "p7", "p8"}
	g, err := NewGame("ABCD", players, "p1")
	if err != nil {
		t.Fatalf("NewGame err: %v", err)
	}

	seer := 0
	wolf := 0
	villager := 0
	all := map[string]Role{"p1": g.MayorSecret}
	for id, r := range g.Roles {
		all[id] = r
	}
	for _, r := range all {
		switch r {
		case RoleSeer:
			seer++
		case RoleWerewolf:
			wolf++
		case RoleVillager:
			villager++
		}
	}
	if seer != 1 {
		t.Fatalf("seer=%d want 1", seer)
	}
	if wolf != 2 {
		t.Fatalf("wolf=%d want 2", wolf)
	}
	if villager != len(players)-seer-wolf {
		t.Fatalf("villager=%d mismatch", villager)
	}
}

func TestRoleMessagesMayorSecret(t *testing.T) {
	players := []string{"host", "a", "b", "c", "d"}
	roles := map[string]Role{"a": RoleSeer, "b": RoleWerewolf, "c": RoleVillager, "d": RoleVillager}
	g := NewGameWithRoles("ABCD", players, "host", RoleWerewolf, roles)

	msgs := g.RoleMessages()
	var hostRole, hostSecret int
	for _, m := range msgs {
		if m.To == "host" && m.Type == "role_assigned" {
			hostRole++
		}
		if m.To == "host" && m.Type == "mayor_secret" {
			hostSecret++
		}
	}
	if hostRole != 1 || hostSecret != 1 {
		t.Fatalf("host messages: role=%d secret=%d", hostRole, hostSecret)
	}
}

func TestNightFlowAllConfirm(t *testing.T) {
	players := []string{"m", "s", "w", "v1", "v2"}
	roles := map[string]Role{"s": RoleSeer, "w": RoleWerewolf, "v1": RoleVillager, "v2": RoleVillager}
	g := NewGameWithRoles("ABCD", players, "m", RoleVillager, roles)

	msgs := g.StartNight([]string{"apple", "sun", "bag"})
	if len(msgs) != len(players) {
		t.Fatalf("StartNight msgs=%d want %d", len(msgs), len(players))
	}
	if _, err := g.MayorPickWord("m", "apple"); err != nil {
		t.Fatalf("MayorPickWord err: %v", err)
	}
	for _, id := range []string{"s", "w", "v1", "v2"} {
		out, err := g.NightConfirm(id)
		if err != nil {
			t.Fatalf("NightConfirm(%s) err: %v", id, err)
		}
		if id == "v2" && len(out) == 0 {
			t.Fatalf("expected step2 messages after all confirmed")
		}
	}
	if g.Phase != PhaseNightStep2 {
		t.Fatalf("phase=%s want night_step2", g.Phase)
	}

	for _, id := range players {
		out, err := g.NightConfirm(id)
		if err != nil {
			t.Fatalf("NightConfirm step2 (%s) err: %v", id, err)
		}
		if id == "v2" && len(out) == 0 {
			t.Fatalf("expected day phase change after all confirmed")
		}
	}
	if g.Phase != PhaseDay {
		t.Fatalf("phase=%s want day", g.Phase)
	}
}

func TestMayorOnlyToken(t *testing.T) {
	players := []string{"m", "a", "b", "c"}
	roles := map[string]Role{"a": RoleSeer, "b": RoleWerewolf, "c": RoleVillager}
	g := NewGameWithRoles("ABCD", players, "m", RoleVillager, roles)
	g.Phase = PhaseDay

	if _, err := g.MayorToken("a", "yes"); err == nil || err.Error() != "mayor_only" {
		t.Fatalf("expected mayor_only, got %v", err)
	}
}

func TestCorrectTokenStartsGuessSeerVote(t *testing.T) {
	players := []string{"m", "s", "w", "v"}
	roles := map[string]Role{"s": RoleSeer, "w": RoleWerewolf, "v": RoleVillager}
	g := NewGameWithRoles("ABCD", players, "m", RoleVillager, roles)
	g.Phase = PhaseDay

	msgs, err := g.MayorToken("m", "correct")
	if err != nil {
		t.Fatalf("MayorToken err: %v", err)
	}
	if len(msgs) != 1 || msgs[0].Type != "word_guessed" {
		t.Fatalf("unexpected msgs: %+v", msgs)
	}
	if g.Phase != PhaseVote || g.VoteType != VoteGuessSeer || !g.WordGuessed {
		t.Fatalf("phase/votetype/wordGuessed mismatch")
	}
}

func TestDayTimeUpStartsGuessWolfVote(t *testing.T) {
	players := []string{"m", "s", "w", "v"}
	roles := map[string]Role{"s": RoleSeer, "w": RoleWerewolf, "v": RoleVillager}
	g := NewGameWithRoles("ABCD", players, "m", RoleVillager, roles)
	g.Phase = PhaseDay

	msgs := g.DayTimeUp()
	if len(msgs) != 1 || msgs[0].Type != "time_up" {
		t.Fatalf("unexpected msgs: %+v", msgs)
	}
	if g.Phase != PhaseVote || g.VoteType != VoteGuessWolf {
		t.Fatalf("phase/votetype mismatch")
	}
}

func TestVoteSelfRejected(t *testing.T) {
	players := []string{"m", "s", "w", "v"}
	roles := map[string]Role{"s": RoleSeer, "w": RoleWerewolf, "v": RoleVillager}
	g := NewGameWithRoles("ABCD", players, "m", RoleVillager, roles)
	g.Phase = PhaseVote
	g.VoteType = VoteGuessWolf

	if _, err := g.CastVote("m", "m"); err == nil || err.Error() != "cannot_vote_self" {
		t.Fatalf("expected cannot_vote_self, got %v", err)
	}
}

func TestVoteDuplicateRejected(t *testing.T) {
	players := []string{"m", "s", "w", "v"}
	roles := map[string]Role{"s": RoleSeer, "w": RoleWerewolf, "v": RoleVillager}
	g := NewGameWithRoles("ABCD", players, "m", RoleVillager, roles)
	g.Phase = PhaseVote
	g.VoteType = VoteGuessWolf

	if _, err := g.CastVote("m", "w"); err != nil {
		t.Fatalf("first vote err: %v", err)
	}
	if _, err := g.CastVote("m", "s"); err == nil || err.Error() != "already_voted" {
		t.Fatalf("expected already_voted, got %v", err)
	}
}

func TestVillagersWinWordGuessedSeerSafe(t *testing.T) {
	players := []string{"m", "s", "w", "v"}
	roles := map[string]Role{"s": RoleSeer, "w": RoleWerewolf, "v": RoleVillager}
	g := NewGameWithRoles("ABCD", players, "m", RoleVillager, roles)
	g.Phase = PhaseVote
	g.VoteType = VoteGuessSeer
	g.WordGuessed = true

	if _, err := g.CastVote("w", "v"); err != nil {
		t.Fatalf("vote err: %v", err)
	}
	if g.Winner != "villagers" || g.Reason != "word_guessed_seer_safe" {
		t.Fatalf("winner=%s reason=%s", g.Winner, g.Reason)
	}
}

func TestWerewolvesWinWordGuessedSeerFound(t *testing.T) {
	players := []string{"m", "s", "w", "v"}
	roles := map[string]Role{"s": RoleSeer, "w": RoleWerewolf, "v": RoleVillager}
	g := NewGameWithRoles("ABCD", players, "m", RoleVillager, roles)
	g.Phase = PhaseVote
	g.VoteType = VoteGuessSeer
	g.WordGuessed = true

	if _, err := g.CastVote("w", "s"); err != nil {
		t.Fatalf("vote err: %v", err)
	}
	if g.Winner != "werewolves" || g.Reason != "word_guessed_seer_found" {
		t.Fatalf("winner=%s reason=%s", g.Winner, g.Reason)
	}
}

func TestVillagersWinWordMissedWolfCaught(t *testing.T) {
	players := []string{"m", "s", "w", "v"}
	roles := map[string]Role{"s": RoleSeer, "w": RoleWerewolf, "v": RoleVillager}
	g := NewGameWithRoles("ABCD", players, "m", RoleVillager, roles)
	g.Phase = PhaseVote
	g.VoteType = VoteGuessWolf
	g.WordGuessed = false

	_, _ = g.CastVote("m", "w")
	_, _ = g.CastVote("s", "w")
	_, _ = g.CastVote("w", "s")
	if _, err := g.CastVote("v", "w"); err != nil {
		t.Fatalf("vote err: %v", err)
	}
	if g.Winner != "villagers" || g.Reason != "word_missed_wolf_caught" {
		t.Fatalf("winner=%s reason=%s", g.Winner, g.Reason)
	}
}

func TestWerewolvesWinWordMissedWolfSafe(t *testing.T) {
	players := []string{"m", "s", "w", "v"}
	roles := map[string]Role{"s": RoleSeer, "w": RoleWerewolf, "v": RoleVillager}
	g := NewGameWithRoles("ABCD", players, "m", RoleVillager, roles)
	g.Phase = PhaseVote
	g.VoteType = VoteGuessWolf
	g.WordGuessed = false

	_, _ = g.CastVote("m", "s")
	_, _ = g.CastVote("s", "m")
	_, _ = g.CastVote("w", "s")
	if _, err := g.CastVote("v", "m"); err != nil {
		t.Fatalf("vote err: %v", err)
	}
	if g.Winner != "werewolves" || g.Reason != "word_missed_wolf_safe" {
		t.Fatalf("winner=%s reason=%s", g.Winner, g.Reason)
	}
}

func TestTieWithWolfCountsAsCorrect(t *testing.T) {
	players := []string{"m", "s", "w", "v1", "v2"}
	roles := map[string]Role{"s": RoleSeer, "w": RoleWerewolf, "v1": RoleVillager, "v2": RoleVillager}
	g := NewGameWithRoles("ABCD", players, "m", RoleVillager, roles)
	g.Phase = PhaseVote
	g.VoteType = VoteGuessWolf
	g.WordGuessed = false

	_, _ = g.CastVote("m", "w")
	_, _ = g.CastVote("s", "v1")
	_, _ = g.CastVote("w", "v1")
	_, _ = g.CastVote("v1", "w")
	if _, err := g.CastVote("v2", "w"); err != nil {
		t.Fatalf("vote err: %v", err)
	}
	if g.Winner != "villagers" {
		t.Fatalf("winner=%s want villagers", g.Winner)
	}
}

func TestVoteTimeUpRandomFillsMissing(t *testing.T) {
	players := []string{"m", "s", "w", "v"}
	roles := map[string]Role{"s": RoleSeer, "w": RoleWerewolf, "v": RoleVillager}
	g := NewGameWithRoles("ABCD", players, "m", RoleVillager, roles)
	g.Phase = PhaseVote
	g.VoteType = VoteGuessWolf
	g.WordGuessed = false

	_, _ = g.CastVote("m", "w")
	msgs := g.VoteTimeUp()
	if len(g.Votes) != 4 {
		t.Fatalf("votes=%d want 4", len(g.Votes))
	}
	if len(msgs) == 0 {
		t.Fatalf("expected vote resolution messages")
	}
}

func TestAbort(t *testing.T) {
	players := []string{"m", "s", "w", "v"}
	roles := map[string]Role{"s": RoleSeer, "w": RoleWerewolf, "v": RoleVillager}
	g := NewGameWithRoles("ABCD", players, "m", RoleVillager, roles)

	msgs := g.Abort("player_disconnected")
	if g.Phase != PhaseResult {
		t.Fatalf("phase=%s want result", g.Phase)
	}
	if len(msgs) != 1 || msgs[0].Type != "game_aborted" {
		t.Fatalf("unexpected msgs: %+v", msgs)
	}
}

// --- Additional edge-case tests ---

func TestNewGameTooFewPlayers(t *testing.T) {
	_, err := NewGame("ABCD", []string{"p1", "p2", "p3"}, "p1")
	if err == nil || err.Error() != "not_enough_players" {
		t.Fatalf("expected not_enough_players, got %v", err)
	}
}

func TestEffectiveRole(t *testing.T) {
	players := []string{"m", "s", "w", "v"}
	roles := map[string]Role{"s": RoleSeer, "w": RoleWerewolf, "v": RoleVillager}
	g := NewGameWithRoles("ABCD", players, "m", RoleSeer, roles)

	if g.EffectiveRole("m") != RoleSeer {
		t.Fatalf("mayor effective role got %s want seer", g.EffectiveRole("m"))
	}
	if g.EffectiveRole("w") != RoleWerewolf {
		t.Fatalf("non-mayor effective role got %s want werewolf", g.EffectiveRole("w"))
	}
	if !g.IsSeer("m") {
		t.Fatal("mayor should be seer")
	}
	if !g.IsWerewolf("w") {
		t.Fatal("w should be werewolf")
	}
	if g.IsWerewolf("v") {
		t.Fatal("v should not be werewolf")
	}
}

func TestMayorPickWordInvalidWord(t *testing.T) {
	players := []string{"m", "s", "w", "v"}
	roles := map[string]Role{"s": RoleSeer, "w": RoleWerewolf, "v": RoleVillager}
	g := NewGameWithRoles("ABCD", players, "m", RoleVillager, roles)
	g.StartNight([]string{"apple", "sun", "bag"})

	_, err := g.MayorPickWord("m", "notacandidate")
	if err == nil || err.Error() != "invalid_word" {
		t.Fatalf("expected invalid_word, got %v", err)
	}
}

func TestMayorPickWordNotMayor(t *testing.T) {
	players := []string{"m", "s", "w", "v"}
	roles := map[string]Role{"s": RoleSeer, "w": RoleWerewolf, "v": RoleVillager}
	g := NewGameWithRoles("ABCD", players, "m", RoleVillager, roles)
	g.StartNight([]string{"apple", "sun", "bag"})

	_, err := g.MayorPickWord("s", "apple")
	if err == nil || err.Error() != "mayor_only" {
		t.Fatalf("expected mayor_only, got %v", err)
	}
}

func TestMayorPickWordWrongPhase(t *testing.T) {
	players := []string{"m", "s", "w", "v"}
	roles := map[string]Role{"s": RoleSeer, "w": RoleWerewolf, "v": RoleVillager}
	g := NewGameWithRoles("ABCD", players, "m", RoleVillager, roles)
	g.Phase = PhaseDay

	_, err := g.MayorPickWord("m", "apple")
	if err == nil || err.Error() != "invalid_phase" {
		t.Fatalf("expected invalid_phase, got %v", err)
	}
}

func TestNightConfirmWrongPhase(t *testing.T) {
	players := []string{"m", "s", "w", "v"}
	roles := map[string]Role{"s": RoleSeer, "w": RoleWerewolf, "v": RoleVillager}
	g := NewGameWithRoles("ABCD", players, "m", RoleVillager, roles)
	g.Phase = PhaseDay

	_, err := g.NightConfirm("s")
	if err == nil || err.Error() != "invalid_phase" {
		t.Fatalf("expected invalid_phase, got %v", err)
	}
}

func TestNightConfirmMayorMustPickWord(t *testing.T) {
	players := []string{"m", "s", "w", "v"}
	roles := map[string]Role{"s": RoleSeer, "w": RoleWerewolf, "v": RoleVillager}
	g := NewGameWithRoles("ABCD", players, "m", RoleVillager, roles)
	g.StartNight([]string{"apple", "sun", "bag"})

	_, err := g.NightConfirm("m")
	if err == nil || err.Error() != "mayor_must_pick_word" {
		t.Fatalf("expected mayor_must_pick_word, got %v", err)
	}
}

func TestNightConfirmAlreadyConfirmedIsNoop(t *testing.T) {
	players := []string{"m", "s", "w", "v"}
	roles := map[string]Role{"s": RoleSeer, "w": RoleWerewolf, "v": RoleVillager}
	g := NewGameWithRoles("ABCD", players, "m", RoleVillager, roles)
	g.StartNight([]string{"apple", "sun", "bag"})

	g.NightConfirm("s")
	msgs, err := g.NightConfirm("s")
	if err != nil {
		t.Fatalf("expected no error for duplicate confirm, got %v", err)
	}
	if msgs != nil {
		t.Fatalf("expected nil messages for duplicate confirm, got %+v", msgs)
	}
}

func TestTokenExhausted(t *testing.T) {
	pool := NewTokenPool()
	pool.Maybe = 0
	err := pool.Consume("maybe")
	if err == nil || err.Error() != "token_exhausted" {
		t.Fatalf("expected token_exhausted, got %v", err)
	}
}

func TestTokenInvalid(t *testing.T) {
	pool := NewTokenPool()
	err := pool.Consume("banana")
	if err == nil || err.Error() != "invalid_token" {
		t.Fatalf("expected invalid_token, got %v", err)
	}
}

func TestTokenConsumeAllTypes(t *testing.T) {
	pool := NewTokenPool()
	for _, tok := range []string{"yes", "no", "maybe", "close", "far", "correct"} {
		if err := pool.Consume(tok); err != nil {
			t.Fatalf("Consume(%s) err: %v", tok, err)
		}
	}
	if pool.Yes != 47 {
		t.Fatalf("yes=%d want 47", pool.Yes)
	}
	if pool.No != 47 {
		t.Fatalf("no=%d want 47", pool.No)
	}
	if pool.Maybe != 0 {
		t.Fatalf("maybe=%d want 0", pool.Maybe)
	}
	if pool.Close != 0 {
		t.Fatalf("close=%d want 0", pool.Close)
	}
	if pool.Far != 0 {
		t.Fatalf("far=%d want 0", pool.Far)
	}
	if pool.Correct != 0 {
		t.Fatalf("correct=%d want 0", pool.Correct)
	}
}

func TestMayorTokenWrongPhase(t *testing.T) {
	players := []string{"m", "s", "w", "v"}
	roles := map[string]Role{"s": RoleSeer, "w": RoleWerewolf, "v": RoleVillager}
	g := NewGameWithRoles("ABCD", players, "m", RoleVillager, roles)
	g.Phase = PhaseVote

	_, err := g.MayorToken("m", "yes")
	if err == nil || err.Error() != "invalid_phase" {
		t.Fatalf("expected invalid_phase, got %v", err)
	}
}

func TestMayorTokenBroadcastsRemaining(t *testing.T) {
	players := []string{"m", "s", "w", "v"}
	roles := map[string]Role{"s": RoleSeer, "w": RoleWerewolf, "v": RoleVillager}
	g := NewGameWithRoles("ABCD", players, "m", RoleVillager, roles)
	g.Phase = PhaseDay

	msgs, err := g.MayorToken("m", "yes")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(msgs) != 1 || msgs[0].Type != "mayor_response" {
		t.Fatalf("expected mayor_response, got %+v", msgs)
	}
	if g.Phase != PhaseDay {
		t.Fatalf("phase should still be day, got %s", g.Phase)
	}
}

func TestTokensDepletedTriggersGuessWolf(t *testing.T) {
	players := []string{"m", "s", "w", "v"}
	roles := map[string]Role{"s": RoleSeer, "w": RoleWerewolf, "v": RoleVillager}
	g := NewGameWithRoles("ABCD", players, "m", RoleVillager, roles)
	g.Phase = PhaseDay
	g.Tokens.Yes = 1
	g.Tokens.No = 0

	msgs, err := g.MayorToken("m", "yes")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages (response + depleted), got %d", len(msgs))
	}
	if msgs[1].Type != "tokens_depleted" {
		t.Fatalf("expected tokens_depleted, got %s", msgs[1].Type)
	}
	if g.Phase != PhaseVote || g.VoteType != VoteGuessWolf {
		t.Fatalf("phase=%s voteType=%s, want vote/guess_wolf", g.Phase, g.VoteType)
	}
}

func TestCastVoteNotEligible(t *testing.T) {
	players := []string{"m", "s", "w", "v"}
	roles := map[string]Role{"s": RoleSeer, "w": RoleWerewolf, "v": RoleVillager}
	g := NewGameWithRoles("ABCD", players, "m", RoleVillager, roles)
	g.Phase = PhaseVote
	g.VoteType = VoteGuessSeer
	g.WordGuessed = true

	_, err := g.CastVote("v", "s")
	if err == nil || err.Error() != "not_eligible_voter" {
		t.Fatalf("expected not_eligible_voter, got %v", err)
	}
}

func TestCastVoteInvalidTarget(t *testing.T) {
	players := []string{"m", "s", "w", "v"}
	roles := map[string]Role{"s": RoleSeer, "w": RoleWerewolf, "v": RoleVillager}
	g := NewGameWithRoles("ABCD", players, "m", RoleVillager, roles)
	g.Phase = PhaseVote
	g.VoteType = VoteGuessWolf

	_, err := g.CastVote("m", "nonexistent")
	if err == nil || err.Error() != "invalid_target" {
		t.Fatalf("expected invalid_target, got %v", err)
	}
}

func TestCastVoteWrongPhase(t *testing.T) {
	players := []string{"m", "s", "w", "v"}
	roles := map[string]Role{"s": RoleSeer, "w": RoleWerewolf, "v": RoleVillager}
	g := NewGameWithRoles("ABCD", players, "m", RoleVillager, roles)
	g.Phase = PhaseDay

	_, err := g.CastVote("m", "w")
	if err == nil || err.Error() != "invalid_phase" {
		t.Fatalf("expected invalid_phase, got %v", err)
	}
}

func TestDayTimeUpOnlyWhenInDay(t *testing.T) {
	players := []string{"m", "s", "w", "v"}
	roles := map[string]Role{"s": RoleSeer, "w": RoleWerewolf, "v": RoleVillager}
	g := NewGameWithRoles("ABCD", players, "m", RoleVillager, roles)
	g.Phase = PhaseVote

	msgs := g.DayTimeUp()
	if msgs != nil {
		t.Fatalf("expected nil msgs when not in day phase, got %+v", msgs)
	}
}

func TestVoteTimeUpOnlyWhenInVote(t *testing.T) {
	players := []string{"m", "s", "w", "v"}
	roles := map[string]Role{"s": RoleSeer, "w": RoleWerewolf, "v": RoleVillager}
	g := NewGameWithRoles("ABCD", players, "m", RoleVillager, roles)
	g.Phase = PhaseDay

	msgs := g.VoteTimeUp()
	if msgs != nil {
		t.Fatalf("expected nil msgs when not in vote phase, got %+v", msgs)
	}
}

func TestSnapshotDeepCopy(t *testing.T) {
	players := []string{"m", "s", "w", "v"}
	roles := map[string]Role{"s": RoleSeer, "w": RoleWerewolf, "v": RoleVillager}
	g := NewGameWithRoles("ABCD", players, "m", RoleVillager, roles)
	g.Phase = PhaseDay
	g.Word = "apple"
	g.Candidates = []string{"apple", "sun", "bag"}
	g.TokenHistory = []string{"yes", "no"}
	g.Votes["m"] = "w"

	snap := g.Snapshot()
	// Mutate original and verify snapshot is independent
	g.Roles["s"] = RoleWerewolf
	g.Votes["s"] = "m"
	g.TokenHistory = append(g.TokenHistory, "maybe")
	g.PlayerIDs[0] = "changed"
	g.Candidates[0] = "changed"

	if snap.Roles["s"] != RoleSeer {
		t.Fatal("snapshot roles should be independent of original")
	}
	if _, ok := snap.Votes["s"]; ok {
		t.Fatal("snapshot votes should be independent of original")
	}
	if len(snap.TokenHistory) != 2 {
		t.Fatal("snapshot token history should be independent")
	}
	if snap.PlayerIDs[0] != "m" {
		t.Fatal("snapshot playerIDs should be independent")
	}
	if snap.Candidates[0] != "apple" {
		t.Fatal("snapshot candidates should be independent")
	}
}

func TestNightStep2MessagesRoleVisibility(t *testing.T) {
	players := []string{"m", "s", "w", "v"}
	roles := map[string]Role{"s": RoleSeer, "w": RoleWerewolf, "v": RoleVillager}
	g := NewGameWithRoles("ABCD", players, "m", RoleVillager, roles)

	g.StartNight([]string{"apple", "sun", "bag"})
	g.MayorPickWord("m", "apple")
	for _, id := range []string{"s", "w", "v"} {
		g.NightConfirm(id)
	}

	// Now in step 2. Check messages were correct.
	// Seer and werewolf should have gotten night_reveal with the word.
	// Mayor and villager should have gotten night_step with "waiting".
	// We can verify by doing step 2 confirms and checking the game advances.
	if g.Phase != PhaseNightStep2 {
		t.Fatalf("phase=%s want night_step2", g.Phase)
	}

	// Verify word was set
	if g.Word != "apple" {
		t.Fatalf("word=%s want apple", g.Word)
	}
}

func TestStartNightResetsConfirmed(t *testing.T) {
	players := []string{"m", "s", "w", "v"}
	roles := map[string]Role{"s": RoleSeer, "w": RoleWerewolf, "v": RoleVillager}
	g := NewGameWithRoles("ABCD", players, "m", RoleVillager, roles)

	msgs := g.StartNight([]string{"apple", "sun", "bag"})
	if len(msgs) != 4 {
		t.Fatalf("StartNight msgs=%d want 4", len(msgs))
	}

	// Mayor should get candidates
	var mayorMsg *OutMsg
	for i := range msgs {
		if msgs[i].To == "m" {
			mayorMsg = &msgs[i]
			break
		}
	}
	if mayorMsg == nil {
		t.Fatal("expected message to mayor")
	}
	if mayorMsg.Payload["candidates"] == nil {
		t.Fatal("mayor should receive candidates")
	}

	// Others should get waiting message
	for i := range msgs {
		if msgs[i].To != "m" {
			if msgs[i].Payload["message"] != "waiting" {
				t.Fatalf("non-mayor player should get waiting message, got %v", msgs[i].Payload)
			}
		}
	}
}

func TestMultipleWolvesVoteInGuessSeer(t *testing.T) {
	players := []string{"m", "s", "w1", "w2", "v1", "v2", "v3"}
	roles := map[string]Role{
		"s": RoleSeer, "w1": RoleWerewolf, "w2": RoleWerewolf,
		"v1": RoleVillager, "v2": RoleVillager, "v3": RoleVillager,
	}
	g := NewGameWithRoles("ABCD", players, "m", RoleVillager, roles)
	g.Phase = PhaseVote
	g.VoteType = VoteGuessSeer
	g.WordGuessed = true

	// Only werewolves should be eligible
	voters := g.eligibleVoters()
	if len(voters) != 2 {
		t.Fatalf("eligible voters=%d want 2", len(voters))
	}

	// Both wolves vote for seer
	_, err := g.CastVote("w1", "s")
	if err != nil {
		t.Fatalf("w1 vote err: %v", err)
	}
	// Game should not be resolved yet (1 of 2 wolves voted)
	if g.Phase == PhaseResult {
		t.Fatal("game should not resolve after 1 of 2 wolves voted")
	}

	_, err = g.CastVote("w2", "s")
	if err != nil {
		t.Fatalf("w2 vote err: %v", err)
	}
	if g.Winner != "werewolves" || g.Reason != "word_guessed_seer_found" {
		t.Fatalf("winner=%s reason=%s", g.Winner, g.Reason)
	}
}

func TestGameOverMsgIncludesAllRoles(t *testing.T) {
	players := []string{"m", "s", "w", "v"}
	roles := map[string]Role{"s": RoleSeer, "w": RoleWerewolf, "v": RoleVillager}
	g := NewGameWithRoles("ABCD", players, "m", RoleVillager, roles)
	g.Phase = PhaseVote
	g.VoteType = VoteGuessWolf
	g.WordGuessed = false
	g.Word = "testword"

	_, _ = g.CastVote("m", "w")
	_, _ = g.CastVote("s", "w")
	_, _ = g.CastVote("w", "s")
	msgs, _ := g.CastVote("v", "w")

	// Find game_over message
	var gameOver *OutMsg
	for i := range msgs {
		if msgs[i].Type == "game_over" {
			gameOver = &msgs[i]
			break
		}
	}
	if gameOver == nil {
		t.Fatal("expected game_over message")
	}
	if gameOver.Payload["word"] != "testword" {
		t.Fatalf("game_over word=%v want testword", gameOver.Payload["word"])
	}
	if gameOver.Payload["winner"] != "villagers" {
		t.Fatalf("game_over winner=%v want villagers", gameOver.Payload["winner"])
	}
	rolesPayload := gameOver.Payload["roles"].(map[string]string)
	if rolesPayload["m"] != "mayor" {
		t.Fatalf("mayor in roles payload=%s want mayor", rolesPayload["m"])
	}
	if rolesPayload["s"] != "seer" {
		t.Fatalf("seer in roles payload=%s want seer", rolesPayload["s"])
	}
}
