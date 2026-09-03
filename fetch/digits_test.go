package fetch

import "testing"

func TestParseDigitTXT(t *testing.T) {
	body := []byte("2026001 2026-01-01 1 2 3 123 0 0\n2026002 2026-01-02 9 8 7 987 0 0\n")
	draws, err := ParseDigitTXT(body, 3)
	if err != nil || len(draws) != 2 {
		t.Fatalf("draws=%d err=%v", len(draws), err)
	}
	if draws[1].Digits[2] != 7 {
		t.Fatalf("unexpected digits: %+v", draws[1])
	}
	body5 := []byte("2026001 2026-01-01 1 2 3 4 5 12345 0\n")
	draws5, err := ParseDigitTXT(body5, 5)
	if err != nil || len(draws5) != 1 || draws5[0].Digits[4] != 5 {
		t.Fatalf("p5 parse failed: %+v err=%v", draws5, err)
	}
}
