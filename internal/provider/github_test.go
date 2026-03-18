package provider

import (
	"testing"
)

func TestBuildAuthURL_injectsToken(t *testing.T) {
	got, err := buildAuthURL("https://github.com/org/repo", "ghs_token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "https://x-access-token:ghs_token@github.com/org/repo"
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

func TestBuildAuthURL_emptyTokenLeavesURLUnchanged(t *testing.T) {
	input := "https://github.com/org/repo"
	got, err := buildAuthURL(input, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != input {
		t.Errorf("expected %q, got %q", input, got)
	}
}

func TestBuildAuthURL_fileURLIgnoresToken(t *testing.T) {
	input := "file:///tmp/repo"
	got, err := buildAuthURL(input, "ghs_token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != input {
		t.Errorf("expected file URL unchanged %q, got %q", input, got)
	}
}

func TestBuildAuthURL_invalidURL(t *testing.T) {
	_, err := buildAuthURL("://bad-url", "token")
	if err == nil {
		t.Fatal("expected error for invalid URL, got nil")
	}
}

func TestBranchFromRef_stripsBranchPrefix(t *testing.T) {
	tests := []struct {
		ref  string
		want string
	}{
		{"refs/heads/main", "main"},
		{"refs/heads/feature/my-branch", "feature/my-branch"},
		{"main", "main"},            // no prefix — returned as-is
		{"refs/tags/v1.0", "refs/tags/v1.0"}, // tag ref — returned as-is
	}
	for _, tt := range tests {
		got := branchFromRef(tt.ref)
		if got != tt.want {
			t.Errorf("branchFromRef(%q) = %q, want %q", tt.ref, got, tt.want)
		}
	}
}
