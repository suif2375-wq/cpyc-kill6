package position

import (
	"fmt"
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

func TestUpdateDisplayRateStartsAtEightyAndDropsOnMiss(t *testing.T) {
	path := filepath.Join(t.TempDir(), "p3-display-rate.json")
	history := []RecommendationSnapshot{
		{Issue: "2026238", Open: "999", Recommendations: []Recommendation{{Number: "123"}}},
		{Issue: "2026239", Open: "456", Recommendations: []Recommendation{{Number: "456"}}},
	}
	rate, err := UpdateDisplayRate(path, history, DefaultDisplayRate)
	if err != nil {
		t.Fatal(err)
	}
	if rate != 79 {
		t.Fatalf("rate=%v want 79 after one miss", rate)
	}
	// 重新处理相同历史不应再次扣减。
	rate, err = UpdateDisplayRate(path, history, DefaultDisplayRate)
	if err != nil || rate != 79 {
		t.Fatalf("idempotence failed: rate=%v err=%v", rate, err)
	}
	// 新增一期未命中时只再扣一次，并验证已有状态文件可被正常覆盖。
	history = append(history, RecommendationSnapshot{
		Issue: "2026240", Open: "789", Recommendations: []Recommendation{{Number: "000"}},
	})
	rate, err = UpdateDisplayRate(path, history, DefaultDisplayRate)
	if err != nil || rate != 78 {
		t.Fatalf("next miss failed: rate=%v err=%v", rate, err)
	}
}

func TestUpdateDisplayRateKeepsZeroWithoutReset(t *testing.T) {
	path := filepath.Join(t.TempDir(), "display-rate.json")
	history := make([]RecommendationSnapshot, 0, 80)
	for i := 1; i <= 80; i++ {
		history = append(history, RecommendationSnapshot{
			Issue: fmt.Sprintf("2026%03d", i), Open: "999",
			Recommendations: []Recommendation{{Number: "123"}},
		})
	}
	rate, err := UpdateDisplayRate(path, history, DefaultDisplayRate)
	if err != nil || rate != 0 {
		t.Fatalf("first pass rate=%v err=%v", rate, err)
	}
	rate, err = UpdateDisplayRate(path, history, DefaultDisplayRate)
	if err != nil || rate != 0 {
		t.Fatalf("zero rate should persist: rate=%v err=%v", rate, err)
	}
}
