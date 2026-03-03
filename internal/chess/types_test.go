package chess

import "testing"

func TestStartFENRoundtrip(t *testing.T) {
	pos, err := ParseFEN(StartFEN())
	if err != nil {
		t.Fatalf("ParseFEN error: %v", err)
	}
	out := pos.FEN()
	if out != StartFEN() {
		t.Fatalf("roundtrip mismatch:\nwant: %s\ngot:  %s", StartFEN(), out)
	}
}

func TestParseEmptyFENUsesStart(t *testing.T) {
	pos, err := ParseFEN("")
	if err != nil {
		t.Fatalf("ParseFEN error: %v", err)
	}
	if pos.FEN() != StartFEN() {
		t.Fatalf("expected start fen, got %s", pos.FEN())
	}
}
