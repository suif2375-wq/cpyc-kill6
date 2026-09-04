// Package position 提供排列3/排列5通用的逐位杀号与滚动回测。
//
// 设计目标不是追求漂亮的历史数字，而是让每一期预测都只看到开奖前的
// 数据。V9局部公式作为稳定主杀号；跨期规律/频率模型只在经过验证的候选
// 场景中参与推荐组合排序，降低单一规律失效时的回撤。
package position

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"fc3d-kill6/data"
	"fc3d-kill6/engine"
	"fc3d-kill6/engine/pattern"
)

// Model 是逐位评分模型。
type Model string

const (
	ModelFrequency  Model = "近期频率"
	ModelTransition Model = "转移概率"
	ModelBlend      Model = "自适应融合"
	ModelV9Local    Model = "V9局部公式"
	ModelPattern    Model = "跨期规律"
)

// Prediction 是下一期逐位杀号结果。
type Prediction struct {
	Kills  [][]int
	Scores [][]float64
	Models []Model
}

// Row 是回测明细。
type Row struct {
	Issue string
	Date  string
	Open  string
	Kills [][]int
	Hits  []bool
	AllOK bool
}

// PositionStat 是单个位次统计。
type PositionStat struct {
	Position int
	N        int
	Hit      int
	Rate     float64
	Baseline float64
	Model    Model
}

// Recommendation 是一组可直接参考的排列3/排列5组合。
// 组合先经过各个位杀号过滤，再按位置评分、和值、跨度和奇偶结构排序；
// 仅用于生成候选，不表示确定性预测。
type Recommendation struct {
	Rank    int
	Number  string
	Digits  []int
	Score   float64
	Sum     int
	Reasons []string
}

// RecommendationSnapshot 是某一期开奖前生成的10组推荐记录。
// Open 用于页面查询时展示该期实际开奖号码，推荐本身仍只来自该期之前的数据。
type RecommendationSnapshot struct {
	Issue           string
	Date            string
	Open            string
	Recommendations []Recommendation
}

type recommendationCandidate struct {
	digits []int
	raw    float64
	sum    int
	span   int
	odd    int
}

// Result 是排列3/排列5页面所需的完整结果。
type Result struct {
	Positions             int
	KillCount             int
	Window                int
	Total                 int
	Latest                data.DigitDraw
	Prediction            Prediction
	Stats                 []PositionStat
	AllN                  int
	AllHit                int
	AllRate               float64
	BaselineAll           float64
	RecentN               int
	RecentHit             int
	RecentRate            float64
	RecentStats           []PositionStat
	Recommendations       []Recommendation
	RecommendationHistory []RecommendationSnapshot
	Rows                  []Row
	Trend                 []float64
}

// Predict 计算下一期各位排除数字。draws 必须按时间正序。
func Predict(draws []data.DigitDraw, killCount, window int) Prediction {
	positions := inferPositions(draws)
	if positions == 0 {
		return Prediction{}
	}
	if killCount <= 0 || killCount >= 10 {
		killCount = 2
	}
	if window <= 0 {
		window = 120
	}
	out := Prediction{
		Kills:  make([][]int, positions),
		Scores: make([][]float64, positions),
		Models: make([]Model, positions),
	}
	for p := 0; p < positions; p++ {
		model := selectModel(draws, p, killCount, window)
		kills, scores := predictPosition(draws, p, model, killCount, window)
		out.Models[p] = model
		out.Scores[p] = scores
		out.Kills[p] = kills
	}
	return out
}

// Backtest 对历史数据做严格滚动回测。每个时点的模型选择只使用该时点之前
// 的开奖数据，避免把未来信息泄漏进预测。
func Backtest(draws []data.DigitDraw, killCount, window int) *Result {
	positions := inferPositions(draws)
	res := &Result{Positions: positions, KillCount: killCount, Window: window, Total: len(draws)}
	if positions == 0 || len(draws) < 2 {
		return res
	}
	if killCount <= 0 || killCount >= 10 {
		killCount = 2
		res.KillCount = killCount
	}
	if window <= 0 {
		window = 120
		res.Window = window
	}

	rows := make([]Row, 0, len(draws)-1)
	start := 1
	if len(draws) > 60 {
		start = 30 // 忽略最初极短样本，避免把冷启动当作模型表现
	}
	statN := make([]int, positions)
	statHit := make([]int, positions)
	allN, allHit := 0, 0
	var activeModels []Model
	for t := 1; t < len(draws); t++ {
		history := draws[:t]
		// 兼容扩展模型的低频刷新机制；当前排列3/排列5主模型固定为V9，
		// 推荐组合另行使用跨期规律软评分。
		if activeModels == nil || t == 1 || t%100 == 0 {
			activeModels = make([]Model, positions)
			for p := 0; p < positions; p++ {
				activeModels[p] = selectModel(history, p, killCount, window)
			}
		}
		pred := predictWithModels(history, killCount, window, activeModels)
		actual := draws[t]
		hits := make([]bool, positions)
		allOK := true
		for p := 0; p < positions; p++ {
			hits[p] = !contains(pred.Kills[p], actual.Digits[p])
			if !hits[p] {
				allOK = false
			}
			if t >= start {
				statN[p]++
				if hits[p] {
					statHit[p]++
				}
			}
		}
		if t >= start {
			allN++
			if allOK {
				allHit++
			}
		}
		rows = append(rows, Row{Issue: actual.Issue, Date: actual.Date, Open: digitsString(actual.Digits), Kills: clone2D(pred.Kills), Hits: hits, AllOK: allOK})
	}

	latest := draws[len(draws)-1]
	res.Latest = latest
	res.Prediction = Predict(draws, killCount, window)
	res.Recommendations = GenerateRecommendations(draws, res.Prediction, 10)
	res.RecommendationHistory = GenerateRecommendationHistory(draws, killCount, window, 100)
	res.AllN, res.AllHit = allN, allHit
	res.AllRate = pct(allHit, allN)
	res.BaselineAll = math.Pow(1-float64(killCount)/10, float64(positions)) * 100
	for p := 0; p < positions; p++ {
		res.Stats = append(res.Stats, PositionStat{
			Position: p + 1,
			N:        statN[p],
			Hit:      statHit[p],
			Rate:     pct(statHit[p], statN[p]),
			Baseline: (1 - float64(killCount)/10) * 100,
			Model:    res.Prediction.Models[p],
		})
	}
	if len(rows) > 100 {
		rows = rows[len(rows)-100:]
	}
	res.RecentN = len(rows)
	recentHit := 0
	recentPosHit := make([]int, positions)
	for _, row := range rows {
		if row.AllOK {
			recentHit++
		}
		for p, ok := range row.Hits {
			if ok {
				recentPosHit[p]++
			}
		}
	}
	res.RecentHit = recentHit
	res.RecentRate = pct(recentHit, len(rows))
	for p := 0; p < positions; p++ {
		model := ModelV9Local
		if p < len(res.Prediction.Models) {
			model = res.Prediction.Models[p]
		}
		res.RecentStats = append(res.RecentStats, PositionStat{
			Position: p + 1,
			N:        len(rows),
			Hit:      recentPosHit[p],
			Rate:     pct(recentPosHit[p], len(rows)),
			Baseline: (1 - float64(killCount)/10) * 100,
			Model:    model,
		})
	}
	// 页面按最新在前展示。
	for i, j := 0, len(rows)-1; i < j; i, j = i+1, j-1 {
		rows[i], rows[j] = rows[j], rows[i]
	}
	res.Rows = rows
	res.Trend = trend(rows)
	return res
}

// GenerateRecommendationHistory 为最近 limit 个已开奖期生成推荐记录。
// 第 t 期的推荐只使用 draws[:t]，因此不会把该期开奖号码泄漏到推荐中。
// 该记录用于页面按期号查询，不参与主杀号回测统计。
func GenerateRecommendationHistory(draws []data.DigitDraw, killCount, window, limit int) []RecommendationSnapshot {
	if len(draws) < 2 || limit <= 0 {
		return nil
	}
	if limit > len(draws)-1 {
		limit = len(draws) - 1
	}
	start := len(draws) - limit
	if start < 1 {
		start = 1
	}
	out := make([]RecommendationSnapshot, 0, len(draws)-start)
	for t := start; t < len(draws); t++ {
		history := draws[:t]
		pred := Predict(history, killCount, window)
		recs := GenerateRecommendations(history, pred, 10)
		out = append(out, RecommendationSnapshot{
			Issue:           draws[t].Issue,
			Date:            draws[t].Date,
			Open:            digitsString(draws[t].Digits),
			Recommendations: recs,
		})
	}
	// 最新期在前，便于页面打开后直接看到最近记录。
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

// GenerateRecommendations 生成 count 组推荐号码。
// 评分使用全局近期位置概率与低权重结构约束，避免把和值/奇偶等规则
// 变成硬过滤条件；因此不会因为单一规则失效而清空候选池。
func GenerateRecommendations(draws []data.DigitDraw, pred Prediction, count int) []Recommendation {
	positions := inferPositions(draws)
	if positions == 0 || len(pred.Kills) != positions || count <= 0 {
		return nil
	}
	if count > 50 {
		count = 50
	}
	probs := make([][]float64, positions)
	allowed := make([][]int, positions)
	// lottery-analyzer 的跨期规律作为软特征参与号码排序：胆码偏好轻微加分，
	// 毒胆轻微减分。它不改变杀号边界，避免规律短期失效时误杀号码。
	danma := pattern.AnalyzeDigits(pattern.DigitDanma, draws, pattern.DefaultDigitConfig)
	dudan := pattern.AnalyzeDigits(pattern.DigitDudan, draws, pattern.DefaultDigitConfig)
	for p := 0; p < positions; p++ {
		probs[p] = scorePosition(draws, p, ModelBlend, 120)
		for d := 0; d <= 9; d++ {
			if !contains(pred.Kills[p], d) {
				allowed[p] = append(allowed[p], d)
			}
		}
		if len(allowed[p]) == 0 {
			allowed[p] = []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
		}
	}

	winStart := len(draws) - 120
	if winStart < 0 {
		winStart = 0
	}
	var sumTotal, spanTotal, oddTotal int
	for i := winStart; i < len(draws); i++ {
		d := draws[i].Digits
		s := 0
		minV, maxV := 9, 0
		odd := 0
		for _, v := range d {
			s += v
			if v < minV {
				minV = v
			}
			if v > maxV {
				maxV = v
			}
			if v%2 == 1 {
				odd++
			}
		}
		sumTotal += s
		spanTotal += maxV - minV
		oddTotal += odd
	}
	n := len(draws) - winStart
	meanSum := float64(sumTotal) / float64(maxInt(n, 1))
	meanSpan := float64(spanTotal) / float64(maxInt(n, 1))
	meanOdd := float64(oddTotal) / float64(maxInt(n, 1))
	stdSum := recentStd(draws, winStart, meanSum, 0)
	stdSpan := recentStd(draws, winStart, meanSpan, 1)
	if stdSum < 1 {
		stdSum = 4
	}
	if stdSpan < 1 {
		stdSpan = 2
	}

	candidates := make([]recommendationCandidate, 0, 1024)
	current := make([]int, positions)
	var walk func(int)
	walk = func(p int) {
		if p == positions {
			s := 0
			minV, maxV, odd := 9, 0, 0
			logP := 0.0
			for i, d := range current {
				s += d
				if d < minV {
					minV = d
				}
				if d > maxV {
					maxV = d
				}
				if d%2 == 1 {
					odd++
				}
				pval := probs[i][d]
				if pval < 1e-9 {
					pval = 1e-9
				}
				logP += math.Log(pval)
			}
			span := maxV - minV
			sumBonus := math.Exp(-math.Abs(float64(s)-meanSum) / stdSum)
			spanBonus := math.Exp(-math.Abs(float64(span)-meanSpan) / stdSpan)
			oddBonus := math.Exp(-math.Abs(float64(odd)-meanOdd) / 1.5)
			patternBonus := 0.0
			for _, d := range current {
				if danma != nil && contains(danma.Picks, d) {
					patternBonus += 0.018
				}
				if dudan != nil && contains(dudan.Picks, d) {
					patternBonus -= 0.012
				}
			}
			candidates = append(candidates, recommendationCandidate{digits: append([]int(nil), current...), raw: logP + 0.18*sumBonus + 0.08*spanBonus + 0.05*oddBonus + patternBonus, sum: s, span: span, odd: odd})
			return
		}
		for _, d := range allowed[p] {
			current[p] = d
			walk(p + 1)
		}
	}
	walk(0)
	if len(candidates) == 0 {
		return nil
	}
	sort.Slice(candidates, func(i, j int) bool {
		if math.Abs(candidates[i].raw-candidates[j].raw) > 1e-12 {
			return candidates[i].raw > candidates[j].raw
		}
		return digitsString(candidates[i].digits) < digitsString(candidates[j].digits)
	})

	selected := make([]recommendationCandidate, 0, count)
	// 10组推荐不应只是同一组号码的微小变体：排列3至少要求相差2个位，
	// 排列5至少相差3个位；候选不足时再逐级放宽，保证始终尽量生成足量组合。
	minDiff := 2
	if positions >= 5 {
		minDiff = 3
	}
	firstCap := (count + len(allowed[0]) - 1) / len(allowed[0])
	if firstCap < 1 {
		firstCap = 1
	}
	firstCounts := make(map[int]int, len(allowed[0]))
	for len(selected) < count && len(selected) < len(candidates) {
		bestIdx, bestScore := -1, -1e100
		for i, cand := range candidates {
			if containsCandidate(selected, cand.digits) {
				continue
			}
			if len(cand.digits) > 0 && firstCounts[cand.digits[0]] >= firstCap {
				continue
			}
			validDistance := true
			for _, picked := range selected {
				if digitDistance(cand.digits, picked.digits) < minDiff {
					validDistance = false
					break
				}
			}
			if len(selected) > 0 && !validDistance {
				continue
			}
			adjusted := cand.raw
			for _, picked := range selected {
				overlap := len(cand.digits) - digitDistance(cand.digits, picked.digits)
				adjusted -= 0.08 * float64(overlap)
			}
			if adjusted > bestScore {
				bestIdx, bestScore = i, adjusted
			}
		}
		if bestIdx < 0 {
			if firstCap < count {
				firstCap++
				continue
			}
			if minDiff > 1 {
				minDiff--
				continue
			}
			break
		}
		selected = append(selected, candidates[bestIdx])
		if len(candidates[bestIdx].digits) > 0 {
			firstCounts[candidates[bestIdx].digits[0]]++
		}
	}

	topRaw := selected[0].raw
	result := make([]Recommendation, 0, len(selected))
	for i, cand := range selected {
		result = append(result, Recommendation{
			Rank: i + 1, Number: digitsString(cand.digits), Digits: append([]int(nil), cand.digits...),
			Score: recommendationScore(cand.raw, topRaw), Sum: cand.sum,
			Reasons: recommendationReasons(cand, meanSum, meanSpan, meanOdd),
		})
	}
	return result
}

func recentStd(draws []data.DigitDraw, start int, mean float64, mode int) float64 {
	if start >= len(draws) {
		return 0
	}
	var ss float64
	for i := start; i < len(draws); i++ {
		v := 0
		if mode == 0 {
			for _, d := range draws[i].Digits {
				v += d
			}
		} else {
			minV, maxV := 9, 0
			for _, d := range draws[i].Digits {
				if d < minV {
					minV = d
				}
				if d > maxV {
					maxV = d
				}
			}
			v = maxV - minV
		}
		ss += (float64(v) - mean) * (float64(v) - mean)
	}
	return math.Sqrt(ss / float64(len(draws)-start))
}

func recommendationScore(raw, top float64) float64 {
	d := top - raw
	s := 100 - d*35
	if s < 60 {
		s = 60
	}
	if s > 100 {
		s = 100
	}
	return s
}

func recommendationReasons(c recommendationCandidate, meanSum, meanSpan, meanOdd float64) []string {
	reasons := []string{"未命中各位杀号", "位置近期评分"}
	if math.Abs(float64(c.sum)-meanSum) <= 3 {
		reasons = append(reasons, "和值靠近期均值")
	}
	if math.Abs(float64(c.span)-meanSpan) <= 2 {
		reasons = append(reasons, "跨度平衡")
	}
	if math.Abs(float64(c.odd)-meanOdd) <= 1 {
		reasons = append(reasons, "奇偶协调")
	}
	return reasons
}

func containsCandidate(cands []recommendationCandidate, digits []int) bool {
	for _, c := range cands {
		if digitsString(c.digits) == digitsString(digits) {
			return true
		}
	}
	return false
}

func digitDistance(a, b []int) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	diff := 0
	for i := 0; i < n; i++ {
		if a[i] != b[i] {
			diff++
		}
	}
	return diff + int(math.Abs(float64(len(a)-len(b))))
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func predictWithModels(draws []data.DigitDraw, killCount, window int, models []Model) Prediction {
	positions := inferPositions(draws)
	out := Prediction{Kills: make([][]int, positions), Scores: make([][]float64, positions), Models: make([]Model, positions)}
	for p := 0; p < positions; p++ {
		model := ModelBlend
		if p < len(models) {
			model = models[p]
		}
		kills, scores := predictPosition(draws, p, model, killCount, window)
		out.Kills[p], out.Scores[p], out.Models[p] = kills, scores, model
	}
	return out
}

func inferPositions(draws []data.DigitDraw) int {
	if len(draws) == 0 {
		return 0
	}
	return len(draws[0].Digits)
}

func selectModel(draws []data.DigitDraw, pos, killCount, window int) Model {
	// 当前排列3/排列5的主杀号保留经过长期样本外验证更稳的 V9 局部公式。
	// lottery-analyzer 的跨期规律会在推荐组合中作为软特征参与排序，
	// 不直接覆盖主杀号，避免短窗口规律把全量表现拉低。
	if positions := inferPositions(draws); positions == 3 || positions == 5 {
		return ModelV9Local
	}
	if len(draws) < 42 {
		return ModelV9Local
	}
	// 用最近最多 60 期做小型 walk-forward 选择；比超短窗口更不容易把
	// 随机波动误判成优势，融合模型平分时优先。
	valN := 60
	if len(draws)-1 < valN {
		valN = len(draws) - 1
	}
	start := len(draws) - valN
	if start < 20 {
		return ModelBlend
	}
	candidates := []Model{ModelV9Local, ModelPattern, ModelFrequency, ModelTransition, ModelBlend}
	scores := make([]int, len(candidates))
	for t := start; t < len(draws); t++ {
		if t == 0 || pos >= len(draws[t].Digits) {
			continue
		}
		for i, model := range candidates {
			kills, _ := predictPosition(draws[:t], pos, model, killCount, window)
			if !contains(kills, draws[t].Digits[pos]) {
				scores[i]++
			}
		}
	}
	best := 0 // V9模型作为稳定基线
	for i := 0; i < len(scores); i++ {
		if i == best {
			continue
		}
		// 只有在验证窗口内至少多命中 2 期时才切换；避免把一两期
		// 随机波动误判成“提升”，这也是防过拟合的关键护栏。
		if scores[i] >= scores[best]+2 {
			best = i
		}
	}
	return candidates[best]
}

func predictPosition(draws []data.DigitDraw, pos int, model Model, killCount, window int) ([]int, []float64) {
	if model == ModelV9Local {
		kills := v9LocalKills(draws, pos)
		if len(kills) > killCount {
			kills = kills[:killCount]
		}
		scores := make([]float64, 10)
		for i := range scores {
			scores[i] = 1
		}
		for _, k := range kills {
			scores[k] = 0
		}
		return kills, scores
	}
	if model == ModelPattern {
		kills := patternDudanKills(draws, pos, killCount)
		if len(kills) > 0 {
			scores := make([]float64, 10)
			for i := range scores {
				scores[i] = 1
			}
			for _, k := range kills {
				scores[k] = 0
			}
			return kills, scores
		}
		return v9LocalKills(draws, pos), nil
	}
	scores := scorePosition(draws, pos, model, window)
	return lowestDigits(scores, killCount), scores
}

func patternDudanKills(draws []data.DigitDraw, pos, killCount int) []int {
	if len(draws) < 16 || pos < 0 || len(draws[0].Digits) == 0 {
		return nil
	}
	res := pattern.AnalyzeDigits(pattern.DigitDudan, draws, pattern.DefaultDigitConfig)
	if res == nil || len(res.Picks) == 0 {
		return nil
	}
	// 规律引擎聚合的是“下一期不出现”的候选数字；按频次取前 N 个，
	// 再交给当前杀号数量约束，避免把三胆结果直接塞入双杀号卡片。
	if killCount <= 0 || killCount > len(res.Picks) {
		killCount = len(res.Picks)
	}
	return append([]int(nil), res.Picks[:killCount]...)
}

func scorePosition(draws []data.DigitDraw, pos int, model Model, window int) []float64 {
	if window <= 0 {
		window = 120
	}
	if len(draws) == 0 {
		return make([]float64, 10)
	}
	freq := frequencyScores(draws, pos, window)
	if model == ModelFrequency {
		return freq
	}
	trans := transitionScores(draws, pos, window)
	if model == ModelTransition {
		return trans
	}
	out := make([]float64, 10)
	for i := range out {
		out[i] = 0.65*freq[i] + 0.35*trans[i]
	}
	return out
}

// v9LocalKills 把上游 V9 公式作为可验证候选特征迁移到排列3/5：
// 排列3直接使用百/十/个位公式；排列5按自身所在位选择相邻三位上下文。
// 它不是强行覆盖模型，而是交给 selectModel 通过历史 walk-forward 决定是否采用。
func v9LocalKills(draws []data.DigitDraw, pos int) []int {
	if len(draws) == 0 || pos < 0 || pos >= len(draws[len(draws)-1].Digits) {
		return []int{0, 1}
	}
	d := draws[len(draws)-1].Digits
	positions := len(d)
	ctx := [3]int{0, 1, 2}
	slot := pos
	if positions == 3 {
		return v9StatefulKills(draws, pos)
	}
	// 排列5的万/千/百位与排列3的百/十/个位使用同一组前三位开奖数据。
	// 为保证两个彩种页面口径一致，前三位严格复用三位 V9 状态机；只有
	// 十位、个位才使用排列5专属的局部三位上下文。
	if positions >= 5 && pos < 3 {
		return v9StatefulKills(draws, pos)
	}
	if positions >= 5 {
		ctx = [5][3]int{{0, 1, 2}, {0, 1, 2}, {1, 2, 3}, {2, 3, 4}, {2, 3, 4}}[pos]
		slot = [5]int{0, 1, 1, 1, 2}[pos]
	}
	b, s, g := d[ctx[0]], d[ctx[1]], d[ctx[2]]
	var k1, k2 int
	switch slot {
	case 0:
		k1, k2 = engine.KillH(b, s, g), engine.KillH2(b, s, g)
	case 1:
		k1, k2 = engine.KillT(b, s, g), engine.KillT2(b, s, g)
	default:
		k1, k2 = engine.KillO(b, s, g, nil, len(draws)), engine.KillO2(b, s, g)
	}
	return distinctKills(k1, k2, slot, b, s, g)
}

func v9StatefulKills(draws []data.DigitDraw, pos int) []int {
	if len(draws) == 0 || len(draws[len(draws)-1].Digits) < 3 {
		return []int{0, 1}
	}
	st := engine.NewState()
	for i := 1; i < len(draws); i++ {
		prev, actual := draws[i-1].Digits, draws[i].Digits
		st.Next(prev[0], prev[1], prev[2], actual[2])
	}
	last := draws[len(draws)-1].Digits
	b, s, g := last[0], last[1], last[2]
	var k1, k2 int
	switch pos {
	case 0:
		k1, k2 = engine.ApplyFB(engine.KillH(b, s, g), st.PHK, engine.HFb, b, s, g), engine.KillH2(b, s, g)
	case 1:
		k1, k2 = engine.ApplyFB(engine.KillT(b, s, g), st.PTK, engine.TFb, b, s, g), engine.KillT2(b, s, g)
	default:
		k1, k2 = engine.ApplyFB(engine.KillO(b, s, g, st.OFail, len(draws)), st.POK, engine.OFb, b, s, g), engine.KillO2(b, s, g)
	}
	return distinctKills(k1, k2, pos, b, s, g)
}

// distinctKills 保证每个位的双杀号始终是两个不同数字。
// 原始 V9 的两个公式偶尔会给出同一个数字；这里优先使用同一期开奖上下文
// 的其它独立公式作为第二候选，最后才使用确定性的偏移，避免页面出现“9,9”
// 这类无效双杀号，同时不改动 engine 包的 golden 基准。
func distinctKills(k1, k2, slot, b, s, g int) []int {
	if k1 != k2 {
		return []int{k1, k2}
	}
	var alternatives []int
	switch slot {
	case 0:
		alternatives = []int{(k1 + 1) % 10, engine.KillT(b, s, g), engine.KillO(b, s, g, nil, 0), engine.KillH2(b, s, g), (k1 + 3) % 10}
	case 1:
		alternatives = []int{(k1 + 1) % 10, engine.KillH(b, s, g), engine.KillO(b, s, g, nil, 0), engine.KillT2(b, s, g), (k1 + 3) % 10}
	default:
		alternatives = []int{(k1 + 1) % 10, engine.KillH(b, s, g), engine.KillT(b, s, g), engine.KillO2(b, s, g), (k1 + 3) % 10}
	}
	for _, alt := range alternatives {
		if alt != k1 {
			return []int{k1, alt}
		}
	}
	return []int{k1, (k1 + 1) % 10}
}

func frequencyScores(draws []data.DigitDraw, pos, window int) []float64 {
	out := make([]float64, 10)
	for i := range out {
		out[i] = 1 // Laplace smoothing
	}
	start := len(draws) - window
	if start < 0 {
		start = 0
	}
	for i := start; i < len(draws); i++ {
		if pos >= len(draws[i].Digits) {
			continue
		}
		age := len(draws) - 1 - i
		weight := math.Pow(0.96, float64(age))
		out[draws[i].Digits[pos]] += weight
	}
	return normalize(out)
}

func transitionScores(draws []data.DigitDraw, pos, window int) []float64 {
	out := make([]float64, 10)
	for i := range out {
		out[i] = 1
	}
	if len(draws) < 2 {
		return normalize(out)
	}
	start := len(draws) - window
	if start < 1 {
		start = 1
	}
	prev := draws[len(draws)-1].Digits
	if pos >= len(prev) {
		return normalize(out)
	}
	for i := start; i < len(draws); i++ {
		if pos >= len(draws[i].Digits) || pos >= len(draws[i-1].Digits) {
			continue
		}
		if draws[i-1].Digits[pos] != prev[pos] {
			continue
		}
		age := len(draws) - 1 - i
		out[draws[i].Digits[pos]] += math.Pow(0.90, float64(age))
	}
	// 没有足够的同前位样本时，退化为全局近期频率，避免极端尖峰。
	if sum(out) <= 10.000001 {
		return frequencyScores(draws, pos, window)
	}
	return normalize(out)
}

func normalize(a []float64) []float64 {
	t := sum(a)
	if t == 0 {
		return a
	}
	for i := range a {
		a[i] /= t
	}
	return a
}

func lowestDigits(scores []float64, n int) []int {
	idx := make([]int, len(scores))
	for i := range idx {
		idx[i] = i
	}
	sort.Slice(idx, func(i, j int) bool {
		if math.Abs(scores[idx[i]]-scores[idx[j]]) > 1e-12 {
			return scores[idx[i]] < scores[idx[j]]
		}
		return idx[i] < idx[j]
	})
	if n > len(idx) {
		n = len(idx)
	}
	return append([]int(nil), idx[:n]...)
}

func contains(a []int, v int) bool {
	for _, x := range a {
		if x == v {
			return true
		}
	}
	return false
}

func clone2D(in [][]int) [][]int {
	out := make([][]int, len(in))
	for i := range in {
		out[i] = append([]int(nil), in[i]...)
	}
	return out
}

func digitsString(d []int) string {
	var b strings.Builder
	for _, n := range d {
		b.WriteByte(byte('0' + n))
	}
	return b.String()
}

func trend(rows []Row) []float64 {
	if len(rows) == 0 {
		return nil
	}
	out := make([]float64, len(rows))
	good := 0
	for i := len(rows) - 1; i >= 0; i-- {
		if rows[i].AllOK {
			good++
		}
		out[i] = pct(good, len(rows)-i)
	}
	return out
}

func sum(a []float64) float64 {
	t := 0.0
	for _, v := range a {
		t += v
	}
	return t
}

func pct(hit, n int) float64 {
	if n == 0 {
		return 0
	}
	return float64(hit) / float64(n) * 100
}

// String 便于日志输出。
func (m Model) String() string { return string(m) }

// FormatPrediction 返回紧凑的逐位杀号字符串。
func FormatPrediction(p Prediction) string {
	parts := make([]string, len(p.Kills))
	for i, ks := range p.Kills {
		d := make([]string, len(ks))
		for j, n := range ks {
			d[j] = fmt.Sprintf("%d", n)
		}
		parts[i] = strings.Join(d, ",")
	}
	return strings.Join(parts, " | ")
}
