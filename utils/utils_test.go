package utils

import "testing"

func TestMakeHealthText(t *testing.T) {
	got := MakeHealthText("ok")
	expected := "[green][✔] ok"
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestMakeWarnText(t *testing.T) {
	got := MakeWarnText("fail")
	expected := "[red][✔] fail"
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}
