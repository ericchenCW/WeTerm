package healthcheck

import "testing"

func TestConsulHealth_buildColorStatus(t *testing.T) {
	h := ConsulHealth{}
	cases := map[int]string{
		0:  "[white]None",
		1:  "[green]Alive",
		2:  "[orange]Leaving",
		3:  "[red]Left",
		4:  "[red]Failed",
		99: "[white]None",
	}
	for in, expect := range cases {
		if got := h.buildColorStatus(in); got != expect {
			t.Errorf("status %d expected %s, got %s", in, expect, got)
		}
	}
}
