package room

import (
	"fmt"
	"testing"
)

func TestCreateRoom(t *testing.T) {
	m := NewManager()
	room, err := m.CreateRoom("p1", "Alice", 6)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if room.Code == "" {
		t.Fatal("room code should not be empty")
	}
	if len(room.Code) != codeLen {
		t.Fatalf("expected code length %d, got %d", codeLen, len(room.Code))
	}
	if room.TargetPlayers != 6 {
		t.Fatalf("expected targetPlayers 6, got %d", room.TargetPlayers)
	}
	if len(room.Players) != 1 {
		t.Fatalf("expected 1 player, got %d", len(room.Players))
	}
	if !room.Players[0].IsHost {
		t.Fatal("first player should be host")
	}
	if room.Players[0].Nickname != "Alice" {
		t.Fatalf("expected nickname Alice, got %s", room.Players[0].Nickname)
	}
}

func TestJoinRoom(t *testing.T) {
	m := NewManager()
	room, _ := m.CreateRoom("p1", "Alice", 6)
	room, err := m.JoinRoom(room.Code, "p2", "Bob")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(room.Players) != 2 {
		t.Fatalf("expected 2 players, got %d", len(room.Players))
	}
}

func TestJoinRoomDuplicateNickname(t *testing.T) {
	m := NewManager()
	room, _ := m.CreateRoom("p1", "Alice", 6)
	_, err := m.JoinRoom(room.Code, "p2", "Alice")
	if err == nil {
		t.Fatal("expected error for duplicate nickname")
	}
	if err.Error() != "nickname_already_taken" {
		t.Fatalf("expected nickname_already_taken, got %v", err)
	}
}

func TestJoinRoomFull(t *testing.T) {
	m := NewManager()
	room, _ := m.CreateRoom("p1", "Alice", 4)
	m.JoinRoom(room.Code, "p2", "Bob")
	m.JoinRoom(room.Code, "p3", "Carol")
	m.JoinRoom(room.Code, "p4", "Dave")
	_, err := m.JoinRoom(room.Code, "p5", "Eve")
	if err == nil {
		t.Fatal("expected error for full room")
	}
	if err.Error() != "room_full" {
		t.Fatalf("expected room_full, got %v", err)
	}
}

func TestJoinRoomNotFound(t *testing.T) {
	m := NewManager()
	_, err := m.JoinRoom("ZZZZ", "p1", "Alice")
	if err == nil {
		t.Fatal("expected error for nonexistent room")
	}
	if err.Error() != "room_not_found" {
		t.Fatalf("expected room_not_found, got %v", err)
	}
}

func TestLeaveRoom(t *testing.T) {
	m := NewManager()
	room, _ := m.CreateRoom("p1", "Alice", 6)
	m.JoinRoom(room.Code, "p2", "Bob")
	room, hostLeft, err := m.LeaveRoom(room.Code, "p2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hostLeft {
		t.Fatal("non-host leaving should not set hostLeft")
	}
	if len(room.Players) != 1 {
		t.Fatalf("expected 1 player, got %d", len(room.Players))
	}
}

func TestHostLeaveDestroysRoom(t *testing.T) {
	m := NewManager()
	room, _ := m.CreateRoom("p1", "Alice", 6)
	code := room.Code
	_, hostLeft, err := m.LeaveRoom(code, "p1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hostLeft {
		t.Fatal("host leaving should set hostLeft")
	}
	_, ok := m.GetRoom(code)
	if ok {
		t.Fatal("room should be deleted after host leaves")
	}
}

func TestStartGameNotEnoughPlayers(t *testing.T) {
	m := NewManager()
	room, _ := m.CreateRoom("p1", "Alice", 6)
	m.JoinRoom(room.Code, "p2", "Bob")
	_, err := m.StartGame(room.Code, "p1")
	if err == nil {
		t.Fatal("expected error for not enough players")
	}
	if err.Error() != "not_enough_players" {
		t.Fatalf("expected not_enough_players, got %v", err)
	}
}

func TestStartGameNotHost(t *testing.T) {
	m := NewManager()
	room, _ := m.CreateRoom("p1", "Alice", 4)
	m.JoinRoom(room.Code, "p2", "Bob")
	m.JoinRoom(room.Code, "p3", "Carol")
	m.JoinRoom(room.Code, "p4", "Dave")
	_, err := m.StartGame(room.Code, "p2")
	if err == nil {
		t.Fatal("expected error for non-host start")
	}
	if err.Error() != "host_only" {
		t.Fatalf("expected host_only, got %v", err)
	}
}

func TestStartGameSuccess(t *testing.T) {
	m := NewManager()
	room, _ := m.CreateRoom("p1", "Alice", 4)
	m.JoinRoom(room.Code, "p2", "Bob")
	m.JoinRoom(room.Code, "p3", "Carol")
	m.JoinRoom(room.Code, "p4", "Dave")
	room, err := m.StartGame(room.Code, "p1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if room.State != StatePlaying {
		t.Fatalf("expected state playing, got %s", room.State)
	}
}

func TestHostDisconnect(t *testing.T) {
	m := NewManager()
	room, _ := m.CreateRoom("p1", "Alice", 6)
	code := room.Code
	isHost := m.HandleDisconnect(code, "p1")
	if !isHost {
		t.Fatal("expected isHost=true")
	}
	_, ok := m.GetRoom(code)
	if ok {
		t.Fatal("room should be deleted after host disconnect")
	}
}

func TestRoomCodeAvoidConfusingChars(t *testing.T) {
	m := NewManager()
	for i := 0; i < 50; i++ {
		room, err := m.CreateRoom(fmt.Sprintf("p%d", i), fmt.Sprintf("Player%d", i), 4)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		for _, c := range room.Code {
			switch c {
			case '0', 'O', '1', 'I', 'L':
				t.Fatalf("room code contains confusing char: %c in %s", c, room.Code)
			}
		}
	}
}

// --- Additional edge-case tests ---

func TestCreateRoomInvalidTargetPlayersTooLow(t *testing.T) {
	m := NewManager()
	_, err := m.CreateRoom("p1", "Alice", 3)
	if err == nil {
		t.Fatal("expected error for targetPlayers < 4")
	}
}

func TestCreateRoomInvalidTargetPlayersTooHigh(t *testing.T) {
	m := NewManager()
	_, err := m.CreateRoom("p1", "Alice", 11)
	if err == nil {
		t.Fatal("expected error for targetPlayers > 10")
	}
}

func TestJoinRoomGameAlreadyStarted(t *testing.T) {
	m := NewManager()
	room, _ := m.CreateRoom("p1", "Alice", 4)
	m.JoinRoom(room.Code, "p2", "Bob")
	m.JoinRoom(room.Code, "p3", "Carol")
	m.JoinRoom(room.Code, "p4", "Dave")
	m.StartGame(room.Code, "p1")

	_, err := m.JoinRoom(room.Code, "p5", "Eve")
	if err == nil || err.Error() != "game_already_started" {
		t.Fatalf("expected game_already_started, got %v", err)
	}
}

func TestLeaveRoomNotFound(t *testing.T) {
	m := NewManager()
	_, _, err := m.LeaveRoom("ZZZZ", "p1")
	if err == nil || err.Error() != "room_not_found" {
		t.Fatalf("expected room_not_found, got %v", err)
	}
}

func TestLeaveRoomPlayerNotFound(t *testing.T) {
	m := NewManager()
	room, _ := m.CreateRoom("p1", "Alice", 6)
	_, _, err := m.LeaveRoom(room.Code, "nonexistent")
	if err == nil || err.Error() != "player_not_found" {
		t.Fatalf("expected player_not_found, got %v", err)
	}
}

func TestStartGameRoomNotFound(t *testing.T) {
	m := NewManager()
	_, err := m.StartGame("ZZZZ", "p1")
	if err == nil || err.Error() != "room_not_found" {
		t.Fatalf("expected room_not_found, got %v", err)
	}
}

func TestHandleDisconnectNonHost(t *testing.T) {
	m := NewManager()
	room, _ := m.CreateRoom("p1", "Alice", 6)
	code := room.Code
	isHost := m.HandleDisconnect(code, "p2")
	if isHost {
		t.Fatal("expected isHost=false for non-host disconnect")
	}
	_, ok := m.GetRoom(code)
	if !ok {
		t.Fatal("room should still exist after non-host disconnect")
	}
}

func TestHandleDisconnectRoomNotFound(t *testing.T) {
	m := NewManager()
	isHost := m.HandleDisconnect("ZZZZ", "p1")
	if isHost {
		t.Fatal("expected isHost=false for nonexistent room")
	}
}

func TestPlayerIDsAndPlayerCount(t *testing.T) {
	m := NewManager()
	room, _ := m.CreateRoom("p1", "Alice", 6)
	m.JoinRoom(room.Code, "p2", "Bob")
	m.JoinRoom(room.Code, "p3", "Carol")

	ids := room.PlayerIDs()
	if len(ids) != 3 {
		t.Fatalf("PlayerIDs len=%d want 3", len(ids))
	}
	if room.PlayerCount() != 3 {
		t.Fatalf("PlayerCount=%d want 3", room.PlayerCount())
	}
	// Verify IDs match
	expected := map[string]bool{"p1": true, "p2": true, "p3": true}
	for _, id := range ids {
		if !expected[id] {
			t.Fatalf("unexpected player ID: %s", id)
		}
	}
}

func TestRemoveRoom(t *testing.T) {
	m := NewManager()
	room, _ := m.CreateRoom("p1", "Alice", 4)
	code := room.Code
	m.RemoveRoom(code)
	_, ok := m.GetRoom(code)
	if ok {
		t.Fatal("room should be deleted after RemoveRoom")
	}
}

func TestCreateRoomBoundaryValues(t *testing.T) {
	m := NewManager()
	// Min boundary
	room, err := m.CreateRoom("p1", "Alice", MinPlayers)
	if err != nil {
		t.Fatalf("unexpected error at min boundary: %v", err)
	}
	if room.TargetPlayers != MinPlayers {
		t.Fatalf("targetPlayers=%d want %d", room.TargetPlayers, MinPlayers)
	}
	// Max boundary
	room2, err := m.CreateRoom("p2", "Bob", MaxPlayers)
	if err != nil {
		t.Fatalf("unexpected error at max boundary: %v", err)
	}
	if room2.TargetPlayers != MaxPlayers {
		t.Fatalf("targetPlayers=%d want %d", room2.TargetPlayers, MaxPlayers)
	}
}
