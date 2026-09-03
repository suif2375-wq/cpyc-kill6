package pattern

import (
	"fmt"
	"math/rand"
	"testing"

	"fc3d-kill6/data"
)

// synth 生成合成数据：可指定注入一条稳定规律，验证引擎能挖出来。
func synth(n int, seed int64, inject func(i int) Draw) []Draw {
	r := rand.New(rand.NewSource(seed))
	ds := make([]Draw, n)
	for i := 0; i < n; i++ {
		if inject != nil {
			d := inject(i)
			if d.Issue != "" {
				ds[i] = d
				continue
			}
		}
		ds[i] = Draw{
			Issue: itoa(2026000 + i),
			B:     r.Intn(10), S: r.Intn(10), G: r.Intn(10),
		}
	}
	return ds
}

func TestAnalyzeDigitsSupportsP5(t *testing.T) {
	draws := make([]data.DigitDraw, 0, 40)
	for i := 0; i < 40; i++ {
		draws = append(draws, data.DigitDraw{Issue: fmt.Sprintf("2026%03d", i+1), Digits: []int{i % 10, (i + 1) % 10, (i + 2) % 10, (i + 3) % 10, (i + 4) % 10}})
	}
	res := AnalyzeDigits(DigitDudan, draws, DefaultDigitConfig)
	if res == nil {
		t.Fatal("nil analysis")
	}
	for _, h := range res.Hits {
		if len(h.Values) != DefaultDigitConfig.CombSize {
			t.Fatalf("unexpected combination width: %+v", h)
		}
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

// TestBuildCombosCount 默认参数组合数 = C(9,3) = 84
func TestBuildCombosCount(t *testing.T) {
	combos := buildCombos(3, 3)
	if len(combos) != 84 {
		t.Fatalf("组合数 = %d, 期望 84", len(combos))
	}
	// 第一个组合应为 1_1|1_2|1_3（迭代序）
	if comboKey(combos[0]) != "1_1|1_2|1_3" {
		t.Fatalf("首个组合 = %q, 期望 1_1|1_2|1_3", comboKey(combos[0]))
	}
	// key 无重复
	seen := map[string]bool{}
	for _, c := range combos {
		k := comboKey(c)
		if seen[k] {
			t.Fatalf("重复组合 %s", k)
		}
		seen[k] = true
	}
}

// TestAnalyzeTooSmall 数据不足时返回空结果而非 panic
func TestAnalyzeTooSmall(t *testing.T) {
	ds := synth(10, 1, nil)
	for _, k := range Kinds {
		a := Analyze(k, ds, Default)
		if a.HitCount != 0 || len(a.Picks) != 0 {
			t.Fatalf("%s 数据不足应返回空: %+v", k, a)
		}
	}
}

// TestDanmaDetectInjectedPattern 注入一条胆码规律：
// 最近 3 期中第 1 期的百位数字，一定会在下一期出现（验证 Danma 能挖出包含它的规律）。
func TestDanmaDetectInjectedPattern(t *testing.T) {
	// 构造：每 4 期一组，前 3 期第 1 期百位 = 7，第 4 期验证期必含 7
	// 规律路径 "1_1"（第 1 期百位）→ 验证期含 7
	ds := make([]Draw, 0, 60)
	period := 0
	val := 7
	for g := 0; g < 15; g++ {
		for i := 0; i < 3; i++ {
			period++
			ds = append(ds, Draw{Issue: itoa(period), B: val + i, S: (val + i + 1) % 10, G: (val + i + 2) % 10})
		}
		period++
		ds = append(ds, Draw{Issue: itoa(period), B: val, S: 1, G: 2}) // 验证期必含 7
	}
	// 最新一期之后再补一组随机（预测窗口需要 3 期）
	ds = append(ds, Draw{Issue: itoa(period + 1), B: 9, S: 8, G: 7})
	ds = append(ds, Draw{Issue: itoa(period + 2), B: 7, S: 6, G: 5})
	ds = append(ds, Draw{Issue: itoa(period + 3), B: 4, S: 3, G: 2})

	a := Analyze(Danma, ds, Default)
	if a.HitCount == 0 {
		t.Fatalf("应挖到至少一条胆码规律, 实际 0 条")
	}
	found := false
	for _, p := range a.Picks {
		if p == val {
			found = true
		}
	}
	if !found {
		t.Fatalf("胆码推荐应包含注入数字 7, 实际 %v", a.Picks)
	}
	// 所有规律 MaxCons >= MinConsecutive
	for _, h := range a.Hits {
		if h.MaxCons < Default.MinConsecutive {
			t.Fatalf("规律 %s MaxCons=%d < MinConsecutive=%d", h.Path, h.MaxCons, Default.MinConsecutive)
		}
	}
}

// TestDudanDetectInjectedPattern 注入毒胆规律：
// 最近 3 期中第 1 期的百位数字 = 7，验证期绝不含 7。
func TestDudanDetectInjectedPattern(t *testing.T) {
	ds := make([]Draw, 0, 60)
	period := 0
	val := 7
	for g := 0; g < 15; g++ {
		for i := 0; i < 3; i++ {
			period++
			ds = append(ds, Draw{Issue: itoa(period), B: val + i, S: (val + i + 1) % 10, G: (val + i + 2) % 10})
		}
		period++
		ds = append(ds, Draw{Issue: itoa(period), B: (val + 1) % 10, S: 1, G: 2}) // 验证期不含 7
	}
	ds = append(ds, Draw{Issue: itoa(period + 1), B: val, S: 9, G: 8}) // 预测窗口第1期百位=7
	ds = append(ds, Draw{Issue: itoa(period + 2), B: 6, S: 5, G: 4})
	ds = append(ds, Draw{Issue: itoa(period + 3), B: 3, S: 2, G: 1})

	a := Analyze(Dudan, ds, Default)
	if a.HitCount == 0 {
		t.Fatalf("应挖到至少一条毒胆规律, 实际 0 条")
	}
	// 注入的杀号 7 必须出现在至少一条命中规律的预测中
	found := false
	for _, h := range a.Hits {
		for _, v := range h.Next {
			if v == val {
				found = true
			}
		}
	}
	if !found {
		t.Fatalf("毒胆规律应包含注入数字 7, 实际规律: %v", a.Hits)
	}
	// 所有规律 MaxCons >= MinConsecutive
	for _, h := range a.Hits {
		if h.MaxCons < Default.MinConsecutive {
			t.Fatalf("规律 %s MaxCons=%d < MinConsecutive=%d", h.Path, h.MaxCons, Default.MinConsecutive)
		}
	}
}

// TestSumTail 和尾分析：全部数字相加尾数恒定 5 的历史 → 杀百十和尾应输出 5。
func TestSumTail(t *testing.T) {
	// 构造：每 4 期一组，前 3 期数字总和尾 = 5，验证期 (百+十)尾 ≠ 5
	ds := make([]Draw, 0, 60)
	period := 0
	for g := 0; g < 15; g++ {
		for i := 0; i < 3; i++ {
			period++
			ds = append(ds, Draw{Issue: itoa(period), B: 1, S: 2, G: 2}) // 1+2+2=5
		}
		period++
		ds = append(ds, Draw{Issue: itoa(period), B: 9, S: 0, G: 9}) // (9+0)%10=9 ≠ 5
	}
	ds = append(ds, Draw{Issue: itoa(period + 1), B: 1, S: 2, G: 2})
	ds = append(ds, Draw{Issue: itoa(period + 2), B: 1, S: 2, G: 2})
	ds = append(ds, Draw{Issue: itoa(period + 3), B: 1, S: 2, G: 2})

	a := Analyze(SumBT, ds, Default)
	if a.HitCount == 0 {
		t.Fatalf("应挖到和尾规律, 实际 0 条")
	}
	if len(a.Picks) == 0 || a.Picks[0] != 5 {
		t.Fatalf("和尾推荐应为 5, 实际 %v", a.Picks)
	}
}

// TestBacktestRates 回测输出合理性：命中率在 [0,100]，窗口数正确。
func TestBacktestRates(t *testing.T) {
	r := rand.New(rand.NewSource(42))
	ds := make([]Draw, 500)
	for i := range ds {
		ds[i] = Draw{Issue: itoa(2026000 + i), B: r.Intn(10), S: r.Intn(10), G: r.Intn(10)}
	}
	res := Backtest(ds, Default, 100)
	if len(res.Stats) != 5 {
		t.Fatalf("stats 数量 = %d, 期望 5", len(res.Stats))
	}
	for _, s := range res.Stats {
		if s.N != 100 {
			t.Fatalf("%s 窗口 = %d, 期望 100", s.Kind, s.N)
		}
		if s.Rate < 0 || s.Rate > 100 {
			t.Fatalf("%s 命中率越界: %.1f", s.Kind, s.Rate)
		}
		if s.FullN <= 0 {
			t.Fatalf("%s 全量期数 = %d", s.Kind, s.FullN)
		}
		// 全量回测率也应合理
		if s.FullRate < 0 || s.FullRate > 100 {
			t.Fatalf("%s 全量命中率越界: %.1f", s.Kind, s.FullRate)
		}
	}
	// 和尾基线应 ≈ 90%（1 个杀尾）
	for _, k := range []Kind{SumBH, SumBT, SumTO} {
		var s KindStats
		for _, st := range res.Stats {
			if st.Kind == k {
				s = st
			}
		}
		if s.Base < 85 || s.Base > 95 {
			t.Fatalf("%s 基线异常: %.1f (期望≈90)", k, s.Base)
		}
	}
}

// TestDeterministic 相同输入两次分析结果一致
func TestDeterministic(t *testing.T) {
	r := rand.New(rand.NewSource(7))
	ds := make([]Draw, 300)
	for i := range ds {
		ds[i] = Draw{Issue: itoa(2026000 + i), B: r.Intn(10), S: r.Intn(10), G: r.Intn(10)}
	}
	for _, k := range Kinds {
		a1 := Analyze(k, ds, Default)
		a2 := Analyze(k, ds, Default)
		if a1.HitCount != a2.HitCount {
			t.Fatalf("%s 两次结果不一致: %d vs %d", k, a1.HitCount, a2.HitCount)
		}
		if len(a1.Picks) != len(a2.Picks) {
			t.Fatalf("%s Picks 不一致", k)
		}
		for i := range a1.Picks {
			if a1.Picks[i] != a2.Picks[i] {
				t.Fatalf("%s Picks 不一致: %v vs %v", k, a1.Picks, a2.Picks)
			}
		}
	}
}

// TestPicksBase 基线数值抽查（0-1 小数，展示时乘 100）
func TestPicksBase(t *testing.T) {
	if got := picksBase(Danma, []int{1, 2, 3}); got < 0.65 || got > 0.66 {
		t.Fatalf("胆码3个基线 = %.4f, 期望≈0.657", got)
	}
	if got := picksBase(Dudan, []int{1, 2, 3}); got < 0.34 || got > 0.35 {
		t.Fatalf("毒胆3个基线 = %.4f, 期望≈0.343", got)
	}
	if got := picksBase(SumBT, []int{5}); got != 0.9 {
		t.Fatalf("和尾1个基线 = %.4f, 期望 0.9", got)
	}
}
