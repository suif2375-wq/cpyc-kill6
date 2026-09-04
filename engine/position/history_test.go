package position

import (
	"path/filepath"
	"testing"

	"fc3d-kill6/data"
)

func TestRecordCurrentPredictionStartsFreshAndIsIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "p3-recommend-history.json")
	result := &Result{
		Total:           10,
		Latest:          data.DigitDraw{Issue: "2026237", Date: "2026-09-04", Digits: []int{9, 5, 0}},
		Recommendations: []Recommendation{{Rank: 1, Number: "123", Digits: []int{1, 2, 3}}},
	}
	history, err := RecordCurrentPrediction(path, result, "2026238", result.Latest.Date)
	if err != nil || len(history) != 1 || history[0].Issue != "2026238" {
		t.Fatalf("first record failed: history=%+v err=%v", history, err)
	}
	if history[0].Open != "" {
		t.Fatalf("future target should not have open result: %+v", history[0])
	}
	history, err = RecordCurrentPrediction(path, result, "2026238", result.Latest.Date)
	if err != nil || len(history) != 1 {
		t.Fatalf("duplicate target should be idempotent: history=%+v err=%v", history, err)
	}

	result2 := &Result{Total: 11, Latest: data.DigitDraw{Issue: "2026238", Date: "2026-09-05", Digits: []int{1, 2, 3}}, Recommendations: []Recommendation{{Rank: 1, Number: "456", Digits: []int{4, 5, 6}}}}
	history, err = RecordCurrentPrediction(path, result2, "2026239", result2.Latest.Date)
	if err != nil || len(history) != 2 {
		t.Fatalf("second record failed: history=%+v err=%v", history, err)
	}
	if history[0].Open != "123" || history[1].Open != "" {
		t.Fatalf("actual result backfill failed: history=%+v", history)
	}
}
