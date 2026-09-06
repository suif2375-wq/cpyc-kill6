package data

import "testing"

// 排列3从第2004021期起与排列5共同摇奖，开奖号码取排列5的前三位。
// 该测试锁定仓库历史数据的一致性，防止同步源异常污染后续回测。
func TestP3P5SharedDrawHistory(t *testing.T) {
	p3, err := LoadDigitCSV("../p3-history.csv", 3)
	if err != nil {
		t.Fatal(err)
	}
	p5, err := LoadDigitCSV("../p5-history.csv", 5)
	if err != nil {
		t.Fatal(err)
	}
	p3ByIssue := make(map[string]DigitDraw, len(p3))
	for _, draw := range p3 {
		p3ByIssue[draw.Issue] = draw
	}
	checked := 0
	for _, draw5 := range p5 {
		if draw5.Issue < "2004021" {
			continue
		}
		draw3, ok := p3ByIssue[draw5.Issue]
		if !ok {
			continue
		}
		checked++
		if draw3.Date != draw5.Date || draw3.Digits[0] != draw5.Digits[0] || draw3.Digits[1] != draw5.Digits[1] || draw3.Digits[2] != draw5.Digits[2] {
			t.Fatalf("issue %s prefix mismatch: p3=%s %v p5=%s %v", draw5.Issue, draw3.Date, draw3.Digits, draw5.Date, draw5.Digits)
		}
	}
	if checked < 7000 {
		t.Fatalf("shared draw sample unexpectedly small: %d", checked)
	}
}
