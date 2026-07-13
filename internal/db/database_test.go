package db

import (
	"errors"
	"testing"
)

func TestContains(t *testing.T) {
	tests := []struct {
		s, substr string
		want      bool
	}{
		{"hello world", "world", true},
		{"hello world", "hello", true},
		{"hello world", "xyz", false},
		{"", "test", false},
		{"test", "", true},
		{"Hello World", "world", false}, // case-sensitive: W != w
		{"hello世界", "世界", true},
		{"abc", "abcdef", false},
	}
	for _, tt := range tests {
		got := contains(tt.s, tt.substr)
		if got != tt.want {
			t.Errorf("contains(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.want)
		}
	}
}

func TestSearchString(t *testing.T) {
	tests := []struct {
		s, substr string
		want      bool
	}{
		{"hello world", "world", true},
		{"hello world", "hello", true},
		{"hello world", "xyz", false},
		{"", "test", false},
		{"a", "a", true},
		{"abc", "ab", true},
		{"abc", "bc", true},
	}
	for _, tt := range tests {
		got := searchString(tt.s, tt.substr)
		if got != tt.want {
			t.Errorf("searchString(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.want)
		}
	}
}

func TestFriendlyError_Nil(t *testing.T) {
	if got := FriendlyError(nil); got != "" {
		t.Errorf("FriendlyError(nil) = %q, want empty", got)
	}
}

func TestFriendlyError_ConnectionRefused(t *testing.T) {
	err := errors.New("connection refused")
	got := FriendlyError(err)
	want := "Could not connect to the database server. Is it running?"
	if got != want {
		t.Errorf("FriendlyError(connection refused) = %q, want %q", got, want)
	}
}

func TestFriendlyError_AuthFailed(t *testing.T) {
	tests := []string{
		"authentication failed",
		"password authentication failed for user",
	}
	for _, msg := range tests {
		err := errors.New(msg)
		got := FriendlyError(err)
		want := "Authentication failed. Check your username and password."
		if got != want {
			t.Errorf("FriendlyError(%q) = %q, want %q", msg, got, want)
		}
	}
}

func TestFriendlyError_DatabaseNotFound(t *testing.T) {
	err := errors.New("database does not exist")
	got := FriendlyError(err)
	want := "The specified database does not exist on the server."
	if got != want {
		t.Errorf("FriendlyError(db not found) = %q, want %q", got, want)
	}
}

func TestFriendlyError_NoSuchHost(t *testing.T) {
	err := errors.New("no such host")
	got := FriendlyError(err)
	want := "Could not resolve the database hostname. Check the host address."
	if got != want {
		t.Errorf("FriendlyError(no such host) = %q, want %q", got, want)
	}
}

func TestFriendlyError_Timeout(t *testing.T) {
	err := errors.New("connection timeout")
	got := FriendlyError(err)
	want := "Connection timed out. Check network connectivity and server status."
	if got != want {
		t.Errorf("FriendlyError(timeout) = %q, want %q", got, want)
	}
}

func TestFriendlyError_SSL(t *testing.T) {
	err := errors.New("SSL error: certificate verify failed")
	got := FriendlyError(err)
	want := "SSL connection failed. Try setting sslmode=disable."
	if got != want {
		t.Errorf("FriendlyError(SSL) = %q, want %q", got, want)
	}
}

func TestFriendlyError_NoSuchTable(t *testing.T) {
	err := errors.New("no such table: foo")
	got := FriendlyError(err)
	want := "The specified table does not exist in the current schema."
	if got != want {
		t.Errorf("FriendlyError(no such table) = %q, want %q", got, want)
	}
}

func TestFriendlyError_SyntaxError(t *testing.T) {
	err := errors.New("syntax error at or near")
	got := FriendlyError(err)
	want := "SQL syntax error. Check your query."
	if got != want {
		t.Errorf("FriendlyError(syntax error) = %q, want %q", got, want)
	}
}

func TestFriendlyError_Default(t *testing.T) {
	err := errors.New("some other database error")
	got := FriendlyError(err)
	want := "some other database error"
	if got != want {
		t.Errorf("FriendlyError(default) = %q, want %q", got, want)
	}
}

func TestWrapError_Nil(t *testing.T) {
	if err := wrapError("op", nil); err != nil {
		t.Errorf("wrapError(op, nil) = %v, want nil", err)
	}
}

func TestWrapError_WithError(t *testing.T) {
	inner := errors.New("inner error")
	err := wrapError("myop", inner)
	if err == nil {
		t.Fatal("wrapError should not return nil for non-nil error")
	}
	if !errors.Is(err, inner) {
		t.Error("wrapError should wrap the original error")
	}
	got := err.Error()
	want := "myop: inner error"
	if got != want {
		t.Errorf("wrapError message = %q, want %q", got, want)
	}
}

func TestFormatDuration_Microseconds(t *testing.T) {
	got := formatDuration(500) // 500ns -> 0.5μs
	// 500ns is < 1μs, so it formats as 0μs
	if got == "" {
		t.Error("formatDuration should not return empty string")
	}
}

func TestFormatDuration_Milliseconds(t *testing.T) {
	got := formatDuration(500 * 1000 * 1000) // 500ms in nanoseconds
	want := "500ms"
	if got != want {
		t.Errorf("formatDuration(500ms) = %q, want %q", got, want)
	}
}

func TestFormatDuration_Seconds(t *testing.T) {
	got := formatDuration(2500 * 1000 * 1000) // 2.5s in nanoseconds
	want := "2.50s"
	if got != want {
		t.Errorf("formatDuration(2.5s) = %q, want %q", got, want)
	}
}

func TestFormatDuration_ExactSecond(t *testing.T) {
	got := formatDuration(1000 * 1000 * 1000) // 1s in nanoseconds
	want := "1.00s"
	if got != want {
		t.Errorf("formatDuration(1s) = %q, want %q", got, want)
	}
}

func TestFormatDuration_Nano(t *testing.T) {
	got := formatDuration(500) // 500ns
	want := "0μs"
	if got != want {
		t.Errorf("formatDuration(500ns) = %q, want %q", got, want)
	}
}
