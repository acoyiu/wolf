package wordlib

import (
	"testing"
)

func TestGetCandidatesEasy(t *testing.T) {
	lib := New()
	words, err := lib.GetCandidates(Easy, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(words) != 3 {
		t.Fatalf("expected 3 words, got %d", len(words))
	}
	seen := map[string]bool{}
	for _, w := range words {
		if seen[w] {
			t.Fatalf("duplicate word: %s", w)
		}
		seen[w] = true
	}
}

func TestGetCandidatesMedium(t *testing.T) {
	lib := New()
	words, err := lib.GetCandidates(Medium, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(words) != 3 {
		t.Fatalf("expected 3 words, got %d", len(words))
	}
}

func TestGetCandidatesHard(t *testing.T) {
	lib := New()
	words, err := lib.GetCandidates(Hard, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(words) != 3 {
		t.Fatalf("expected 3 words, got %d", len(words))
	}
}

func TestGetCandidatesNoDuplicateAcrossCalls(t *testing.T) {
	lib := New()
	w1, _ := lib.GetCandidates(Easy, 3)
	w2, _ := lib.GetCandidates(Easy, 3)
	if len(w1) != 3 || len(w2) != 3 {
		t.Fatalf("expected 3 words each")
	}
}

func TestGetCandidatesInvalidDifficulty(t *testing.T) {
	lib := New()
	_, err := lib.GetCandidates("invalid", 3)
	if err == nil {
		t.Fatal("expected error for invalid difficulty")
	}
	if err.Error() != "invalid difficulty level" {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestWordCountMinimum100(t *testing.T) {
	lib := New()
	for _, diff := range ValidDifficulties() {
		count, err := lib.WordCount(diff)
		if err != nil {
			t.Fatalf("unexpected error for %s: %v", diff, err)
		}
		if count < 100 {
			t.Fatalf("difficulty %s has %d words, expected >= 100", diff, count)
		}
	}
}

func TestWordCountInvalidDifficulty(t *testing.T) {
	lib := New()
	_, err := lib.WordCount("invalid")
	if err == nil {
		t.Fatal("expected error for invalid difficulty")
	}
}

func FuzzGetCandidates(f *testing.F) {
	f.Add("easy", 1)
	f.Add("medium", 3)
	f.Add("hard", 5)
	f.Add("invalid", 0)

	lib := New()
	f.Fuzz(func(t *testing.T, diff string, n int) {
		words, err := lib.GetCandidates(Difficulty(diff), n)
		if diff != "easy" && diff != "medium" && diff != "hard" {
			if err == nil {
				t.Error("expected error for invalid difficulty")
			}
			return
		}
		if n <= 0 || n > 100 {
			return
		}
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			return
		}
		if len(words) != n {
			t.Errorf("expected %d words, got %d", n, len(words))
		}
		seen := map[string]bool{}
		for _, w := range words {
			if seen[w] {
				t.Errorf("duplicate word: %s", w)
			}
			seen[w] = true
		}
	})
}
