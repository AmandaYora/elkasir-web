package infrastructure

import "testing"

func TestIsPaidStatus(t *testing.T) {
	paid := []string{"SUCCEEDED", "completed", "Paid", "success"}
	for _, s := range paid {
		if !isPaidStatus(s) {
			t.Errorf("%q seharusnya dianggap lunas", s)
		}
	}
	notPaid := []string{"PENDING", "ACTIVE", "FAILED", "EXPIRED", ""}
	for _, s := range notPaid {
		if isPaidStatus(s) {
			t.Errorf("%q seharusnya TIDAK lunas", s)
		}
	}
}

func TestFirstNonEmpty(t *testing.T) {
	if got := firstNonEmpty("", "  ", "abc", "def"); got != "abc" {
		t.Errorf("firstNonEmpty = %q, want abc", got)
	}
	if got := firstNonEmpty("", ":"); got != "" {
		t.Errorf("firstNonEmpty hanya ':' harus kosong, got %q", got)
	}
}
