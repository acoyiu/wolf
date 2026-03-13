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
