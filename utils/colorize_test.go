package utils

import "testing"

func TestColorize(t *testing.T) {
	got := Colorize("hello", Red)
	expected := "\x1b[31mhello\x1b[0m"
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}

	if Colorize("world", 0) != "world" {
		t.Errorf("colorize with 0 should return input unchanged")
	}
}

func TestANSIColorize(t *testing.T) {
	got := ANSIColorize("hi", 120)
	expected := "\x1b[38;5;120mhi\x1b[0m"
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestHighlight(t *testing.T) {
	in := []byte("abc")
	out := Highlight(in, []int{1}, 200)
	expected := append([]byte{'a'}, []byte("\x1b[38;5;209mb\x1b[0m")...)
	expected = append(expected, 'c')
	if string(out) != string(expected) {
		t.Errorf("expected %q, got %q", expected, out)
	}
}
