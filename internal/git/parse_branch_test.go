package git

import (
	"testing"
)

func TestParseBranchList(t *testing.T) {
	input := "*\x00main\x00abc1234\x00Initial commit\x00origin/main\x00[ahead 1]\x1e \x00feature\x00def5678\x00Add feature\x00\x00\x1e"

	branches := ParseBranchList(input)

	if len(branches) != 2 {
		t.Fatalf("branches count = %d, want 2", len(branches))
	}

	if !branches[0].IsCurrent {
		t.Error("branch 0 should be current")
	}

	if branches[0].Name != "main" {
		t.Errorf("branch 0 name = %q, want %q", branches[0].Name, "main")
	}

	if branches[0].Hash != "abc1234" {
		t.Errorf("branch 0 hash = %q, want %q", branches[0].Hash, "abc1234")
	}

	if branches[0].Subject != "Initial commit" {
		t.Errorf("branch 0 subject = %q, want %q", branches[0].Subject, "Initial commit")
	}

	if branches[0].Upstream != "origin/main" {
		t.Errorf("branch 0 upstream = %q, want %q", branches[0].Upstream, "origin/main")
	}

	if branches[0].Track != "[ahead 1]" {
		t.Errorf("branch 0 track = %q, want %q", branches[0].Track, "[ahead 1]")
	}

	if branches[1].IsCurrent {
		t.Error("branch 1 should not be current")
	}

	if branches[1].Name != "feature" {
		t.Errorf("branch 1 name = %q, want %q", branches[1].Name, "feature")
	}
}

func TestParseBranchListEmpty(t *testing.T) {
	branches := ParseBranchList("")

	if len(branches) != 0 {
		t.Errorf("branches count = %d, want 0", len(branches))
	}
}
