package component

import (
	"strings"
	"testing"
)

func TestProgressBarUpdate(t *testing.T) {
	pb := NewProgressBar(100)
	pb.UpdateProgressBar(10)
	got := pb.GetProgressBarInstance().GetText(false)
	expected := "[green:]" + strings.Repeat("|", 10) + "[white:]" + strings.Repeat(" ", 90)
	if got != expected {
		t.Errorf("unexpected progress bar text: %q", got)
	}
}
