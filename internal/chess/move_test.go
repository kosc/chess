package chess

import "testing"

func TestParseUCI(t *testing.T) {
	m, err := ParseUCI("e2e4")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if m.From.String() != "e2" || m.To.String() != "e4" {
		t.Fatalf("got %v", m)
	}

	mp, err := ParseUCI("e7e8q")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if mp.Flags&FlagPromotion == 0 || mp.Promo != Queen {
		t.Fatalf("expected promotion to queen: %+v", mp)
	}
}
