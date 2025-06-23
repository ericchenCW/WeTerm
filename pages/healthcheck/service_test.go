package healthcheck

import "testing"

func TestServiceHealth_buildColorStatus(t *testing.T) {
	h := ServiceHealth{}
	cases := map[string]string{
		"passing":  "[green]passing",
		"warning":  "[orange]warning",
		"critical": "[red]critical",
		"unknown":  "[white]None",
	}
	for in, expect := range cases {
		if got := h.buildColorStatus(in); got != expect {
			t.Errorf("input %s: expected %s, got %s", in, expect, got)
		}
	}
}

func TestServiceHealth_getServiceFullName(t *testing.T) {
	h := ServiceHealth{}
	tests := map[string][]string{
		"mysql-80":      {"mysql"},
		"mongodb-27017": {"mongodb"},
		"nodeman":       {"nodeman"},
		"job-gateway":   {"job", "gateway"},
		"gse-agent":     {"gse"},
		"bk-cmdb":       {"bk", "cmdb"},
	}
	for input, expect := range tests {
		got := h.getServiceFullName(input)
		if len(got) != len(expect) {
			t.Fatalf("input %s expected %v got %v", input, expect, got)
		}
		for i, v := range expect {
			if got[i] != v {
				t.Errorf("input %s expect %v got %v", input, expect, got)
				break
			}
		}
	}
}
