package report

import (
	"fmt"
	"html"
	"math"
	"strings"
	"time"

	"fc3d-kill6/engine/position"
)

// ExtendWithDigits 在上游福彩3D/双色球静态页面中注入排列3、排列5页签。
// 这样保留上游页面的成熟样式和双色球功能，同时让新增玩法独立演进。
func ExtendWithDigits(base string, p3, p5 *position.Result) string {
	if base == "" {
		return base
	}
	css := digitCSS()
	if i := strings.Index(base, "</style>"); i >= 0 {
		base = base[:i] + css + base[i:]
	}
	base = strings.Replace(base, "<title>福彩3D 杀码 + 双色球 数据参考</title>", "<title>福彩3D + 排列3 + 排列5 + 双色球 数据参考</title>", 1)
	base = strings.Replace(base, "福彩3D + 双色球 · 数据参考", "福彩3D + 排列3/5 + 双色球 · 数据参考", 1)
	base = strings.Replace(base, "福彩3D + 双色球 · 每日自动更新", "福彩3D + 排列3/5 + 双色球 · 每日自动更新", 1)
	base = updateDigitFooterDates(base, p3, p5)
	base = strings.Replace(base, `<input type="radio" name="lot" id="tab-ssq">`, `<input type="radio" name="lot" id="tab-ssq">
    <input type="radio" name="lot" id="tab-p3">
    <input type="radio" name="lot" id="tab-p5">`, 1)
	base = strings.Replace(base, `<label class="tab-btn" for="tab-ssq"><span class="tab-ico">球</span>双色球</label>`, `<label class="tab-btn" for="tab-ssq"><span class="tab-ico">球</span>双色球</label>
      <label class="tab-btn" for="tab-p3"><span class="tab-ico">P3</span>排列3</label>
      <label class="tab-btn" for="tab-p5"><span class="tab-ico">P5</span>排列5</label>`, 1)
	marker := `<div class="tab-pane" id="pane-ssq">`
	insert := renderDigitPane("排列3", "p3", "三位结构 · V9 双引擎局部公式与滚动回测", p3) +
		renderDigitPane("排列5", "p5", "五位数字独立建模 · 逐位杀号 + 严格 walk-forward 回测", p5)
	if strings.Contains(base, marker) {
		base = strings.Replace(base, marker, insert+"\n"+marker, 1)
	}
	return base
}

func updateDigitFooterDates(base string, p3, p5 *position.Result) string {
	prefix := `<span class="foot-meta">`
	start := strings.Index(base, prefix)
	if start < 0 {
		return base
	}
	endRel := strings.Index(base[start+len(prefix):], `</span>`)
	if endRel < 0 {
		return base
	}
	end := start + len(prefix) + endRel
	content := base[start+len(prefix) : end]
	dates := make([]string, 0, 2)
	if p3 != nil && p3.Latest.Date != "" {
		dates = append(dates, "排列3 数据截止 "+p3.Latest.Date)
	}
	if p5 != nil && p5.Latest.Date != "" {
		dates = append(dates, "排列5 数据截止 "+p5.Latest.Date)
	}
	if len(dates) == 0 {
		return base
	}
	marker := " · 每日开奖后自动更新"
	if strings.Contains(content, marker) {
		content = strings.Replace(content, marker, " · "+strings.Join(dates, " · ")+marker, 1)
	} else {
		content += " · " + strings.Join(dates, " · ")
	}
	return base[:start+len(prefix)] + content + base[end:]
}

func renderDigitPane(title, key, desc string, res *position.Result) string {
	var b strings.Builder
	b.WriteString(`<div class="tab-pane" id="pane-` + key + `"><main class="digit-main">`)
	b.WriteString(`<section class="digit-hero"><div class="digit-kicker"><span class="digit-tag">自适应融合</span><span>` + html.EscapeString(desc) + `</span></div>`)
	b.WriteString(`<h1>` + html.EscapeString(title) + ` ` + map[string]string{"p3": "三位", "p5": "五位"}[key] + `杀码参考</h1>`)
	if res == nil || res.Total == 0 {
		b.WriteString(`<p class="digit-muted">暂无历史数据。运行程序后会自动同步 17500.cn 的完整历史并生成回测。</p></section></main></div>`)
		return b.String()
	}
	next := nextIssue(res.Latest.Issue, res.Latest.Date)
	b.WriteString(`<p class="digit-sub">下一期参考 · 第 <strong>` + html.EscapeString(next) + `</strong> 期。每个位排除 ` + fmt.Sprintf("%d", res.KillCount) + ` 个数字，结果仅基于开奖前历史数据。</p>`)
	b.WriteString(`<div class="digit-note">杀码是排除数字，不是预测开奖号码。主杀号使用经过长期滚动验证的 V9 局部公式；lottery-analyzer 的跨期胆码/毒胆规律作为推荐组合的软评分特征，并同步显示随机基线。</div>`)
	b.WriteString(`<div class="digit-pred-grid">`)
	labels := []string{"万位", "千位", "百位", "十位", "个位"}
	for i, ks := range res.Prediction.Kills {
		label := fmt.Sprintf("第%d位", i+1)
		if i < len(labels) && res.Positions == 5 {
			label = labels[i]
		} else if res.Positions == 3 {
			label = []string{"百位", "十位", "个位"}[i]
		}
		b.WriteString(`<div class="digit-pred-card"><div class="digit-pred-head"><span>` + label + ` · ` + fmt.Sprintf("%d杀", len(ks)) + `</span><small>` + html.EscapeString(res.Prediction.Models[i].String()) + `</small></div><div class="digit-pred-num">` + joinDigits(ks) + `</div><div class="digit-pred-foot">排除后保留 ` + fmt.Sprintf("%d", 10-len(ks)) + ` 个数字</div></div>`)
	}
	b.WriteString(`</div></section>`)

	b.WriteString(`<section class="digit-section digit-recommend-section"><div class="digit-section-head"><h2>10组推荐号码</h2><span>杀号过滤 · 位置评分 · 和值/跨度结构排序</span></div><div class="digit-recommend-grid">`)
	for _, rec := range res.Recommendations {
		b.WriteString(`<div class="digit-recommend-card"><div class="digit-recommend-rank">` + fmt.Sprintf("%02d", rec.Rank) + `</div><div class="digit-recommend-number">` + html.EscapeString(rec.Number) + `</div><div class="digit-recommend-meta">和值 ` + fmt.Sprintf("%d", rec.Sum) + ` · 评分 ` + fmt.Sprintf("%.1f", rec.Score) + `</div><div class="digit-recommend-reasons">` + html.EscapeString(strings.Join(rec.Reasons, " · ")) + `</div></div>`)
	}
	b.WriteString(`</div><p class="digit-disclaimer">推荐组合用于覆盖不同的高分结构，不代表确定性结果；每组均先排除当前各个位杀号数字。</p></section>`)

	b.WriteString(`<section class="digit-section"><div class="digit-section-head"><h2>滚动回测</h2><span>近` + fmt.Sprintf("%d", res.RecentN) + `期 · 全量 ` + fmt.Sprintf("%.1f", res.AllRate) + `% · 随机基线 ` + fmt.Sprintf("%.1f", res.BaselineAll) + `%</span></div>`)
	b.WriteString(`<div class="digit-summary"><div class="digit-summary-main"><span>近` + fmt.Sprintf("%d", res.RecentN) + `期全位同时避开率</span><strong>` + fmt.Sprintf("%.1f%%", res.RecentRate) + `</strong><em>` + fmt.Sprintf("较随机基线 %+.1fpp · 全量 %.1f%%", res.RecentRate-res.BaselineAll, res.AllRate) + `</em><div class="digit-bar"><i style="width:` + fmt.Sprintf("%.1f", clamp(res.RecentRate)) + `%"></i></div></div><div class="digit-stats">`)
	stats := res.RecentStats
	if len(stats) == 0 {
		stats = res.Stats
	}
	for _, st := range stats {
		label := fmt.Sprintf("第%d位", st.Position)
		if res.Positions == 5 {
			label = labels[st.Position-1]
		} else if res.Positions == 3 {
			label = []string{"百位", "十位", "个位"}[st.Position-1]
		}
		b.WriteString(`<div class="digit-stat"><span>` + label + `</span><strong>` + fmt.Sprintf("%.1f%%", st.Rate) + `</strong><small>` + html.EscapeString(st.Model.String()) + ` · 基线 ` + fmt.Sprintf("%.1f%%", st.Baseline) + `</small></div>`)
	}
	b.WriteString(`</div></div></section>`)

	b.WriteString(`<section class="digit-section"><div class="digit-section-head"><h2>表现趋势</h2><span>最新在左 · 近100期累计全位避开率</span></div><div class="digit-chart">` + digitTrendSVG(res.Trend) + `</div></section>`)
	b.WriteString(`<section class="digit-section"><div class="digit-section-head"><h2>近100期回测明细</h2><span>每行只使用该期之前的数据</span></div><div class="digit-table-wrap"><table class="digit-table"><thead><tr><th>期号</th><th>日期</th><th>开奖</th><th>各位杀号</th><th>结果</th></tr></thead><tbody>`)
	for _, row := range res.Rows {
		b.WriteString(`<tr><td>` + html.EscapeString(row.Issue) + `</td><td>` + html.EscapeString(row.Date) + `</td><td class="digit-open">` + html.EscapeString(row.Open) + `</td><td>` + html.EscapeString(position.FormatPrediction(position.Prediction{Kills: row.Kills})) + `</td><td class="` + boolClass(row.AllOK) + `">` + boolText(row.AllOK) + `</td></tr>`)
	}
	b.WriteString(`</tbody></table></div><p class="digit-disclaimer">彩票开奖结果具有随机性，回测结果不代表未来收益。该页面仅作历史数据研究参考。</p></section></main></div>`)
	return b.String()
}

func joinDigits(a []int) string {
	parts := make([]string, len(a))
	for i, n := range a {
		parts[i] = fmt.Sprintf("%d", n)
	}
	return strings.Join(parts, ", ")
}

func boolClass(ok bool) string {
	if ok {
		return "digit-ok"
	}
	return "digit-fail"
}

func boolText(ok bool) string {
	if ok {
		return "全位命中"
	}
	return "未全中"
}

func nextIssue(issue, date string) string {
	if len(issue) < 7 {
		return "待更新"
	}
	var year, seq int
	if _, err := fmt.Sscanf(issue[:4], "%d", &year); err != nil {
		return "待更新"
	}
	if _, err := fmt.Sscanf(issue[4:], "%d", &seq); err != nil {
		return "待更新"
	}
	if d, err := time.Parse("2006-01-02", date); err == nil && d.Month() == 12 && d.Day() == 31 {
		return fmt.Sprintf("%d001", year+1)
	}
	return fmt.Sprintf("%d%03d", year, seq+1)
}

func clamp(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}

func digitTrendSVG(values []float64) string {
	if len(values) == 0 {
		return `<div class="digit-muted">数据不足</div>`
	}
	const w, h = 760.0, 210.0
	left, right, top, bottom := 18.0, 50.0, 20.0, 28.0
	plotW, plotH := w-left-right, h-top-bottom
	minY, maxY := 0.0, 100.0
	point := func(i int, v float64) string {
		x := left
		if len(values) > 1 {
			x += float64(i) * plotW / float64(len(values)-1)
		}
		y := top + (maxY-clamp(v))/(maxY-minY)*plotH
		return fmt.Sprintf("%.1f,%.1f", x, y)
	}
	pts := make([]string, len(values))
	for i, v := range values {
		pts[i] = point(i, v)
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf(`<svg viewBox="0 0 %.0f %.0f" role="img" aria-label="逐期全位避开率趋势"><line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="#334155" stroke-dasharray="4 4"/><text x="%.1f" y="%.1f" fill="#94A3B8" font-size="11">随机基线</text><polyline points="%s" fill="none" stroke="#22D3EE" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"/>`, w, h, left, top+plotH*0.5, w-right, top+plotH*0.5, w-right-64, top+plotH*0.5-6, strings.Join(pts, " ")))
	v := values[0]
	last := strings.Split(pts[0], ",")
	b.WriteString(fmt.Sprintf(`<circle cx="%s" cy="%s" r="4" fill="#34D399"/><text x="%s" y="%s" fill="#34D399" font-size="13" font-weight="700">%.1f%%</text></svg>`, last[0], last[1], last[0], fmt.Sprintf("%.1f", math.Max(12, parseFloat(last[1])-9)), v))
	return b.String()
}

func parseFloat(s string) float64 {
	var v float64
	_, _ = fmt.Sscanf(s, "%f", &v)
	return v
}

func digitCSS() string {
	return `<style>
#tab-p3:checked~.tab-bar .tab-btn[for="tab-p3"],#tab-p5:checked~.tab-bar .tab-btn[for="tab-p5"]{color:#67E8F9;border-color:rgba(34,211,238,.55);background:rgba(23,71,97,.5)}
#tab-p3:checked~#pane-p3,#tab-p5:checked~#pane-p5{display:block}
.digit-main{max-width:100%}.digit-hero{padding:34px 0 42px}.digit-kicker{display:flex;align-items:center;gap:12px;color:#64748B;font-size:12px}.digit-tag{padding:4px 10px;border-radius:999px;background:rgba(34,211,238,.12);border:1px solid rgba(34,211,238,.4);color:#67E8F9;font-weight:700}.digit-hero h1{margin-top:14px;font-size:40px}.digit-sub{margin-top:12px;color:#94A3B8;line-height:1.7}.digit-sub strong{color:#67E8F9}.digit-note{max-width:780px;margin-top:16px;padding:12px 16px;border-radius:12px;background:rgba(52,211,153,.07);border:1px solid rgba(52,211,153,.28);color:#94A3B8;font-size:12px;line-height:1.7}.digit-pred-grid{display:grid;grid-template-columns:repeat(5,minmax(0,1fr));gap:14px;margin-top:24px}.digit-pred-card{padding:18px 16px;border-radius:16px;background:linear-gradient(180deg,rgba(23,71,97,.75),rgba(13,20,41,1));border:1px solid rgba(34,211,238,.32);min-height:150px;display:flex;flex-direction:column;justify-content:space-between}.digit-pred-head{display:flex;justify-content:space-between;gap:8px;color:#67E8F9;font-size:12px}.digit-pred-head small{color:#94A3B8;font-size:10px}.digit-pred-num{font:800 34px/1.1 var(--font-num);text-align:center;color:#22D3EE;letter-spacing:1px}.digit-pred-foot{font-size:10px;color:#64748B;text-align:center}.digit-section{padding:0 0 34px}.digit-section-head{display:flex;justify-content:space-between;align-items:end;margin-bottom:14px}.digit-section-head h2{font-size:22px}.digit-section-head span{font-size:11px;color:#64748B}.digit-summary{display:grid;grid-template-columns:320px 1fr;gap:16px}.digit-summary-main,.digit-stat{padding:18px;border-radius:14px;background:#121A2B;border:1px solid rgba(148,163,184,.12)}.digit-summary-main{display:flex;flex-direction:column;gap:10px}.digit-summary-main span{color:#6EE7B7;font-size:13px}.digit-summary-main strong{font:800 40px var(--font-num);color:#34D399}.digit-summary-main em{font-size:11px;color:#94A3B8;font-style:normal}.digit-bar{height:6px;border-radius:3px;background:#1E293B;overflow:hidden}.digit-bar i{display:block;height:100%;border-radius:3px;background:linear-gradient(90deg,#34D399,#22D3EE)}.digit-stats{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:12px}.digit-stat{display:flex;flex-direction:column;gap:7px}.digit-stat span{font-size:12px;color:#94A3B8}.digit-stat strong{font:800 26px var(--font-num);color:#67E8F9}.digit-stat small{font-size:10px;color:#64748B}.digit-chart{padding:12px;border-radius:14px;background:#0D1424;border:1px solid rgba(148,163,184,.12);overflow:auto}.digit-chart svg{display:block;width:100%;min-width:620px;height:210px}.digit-table-wrap{overflow:auto;border-radius:14px;border:1px solid rgba(148,163,184,.12)}.digit-table{width:100%;border-collapse:collapse;min-width:680px;background:#0D1424}.digit-table th,.digit-table td{padding:10px 12px;border-bottom:1px solid rgba(148,163,184,.1);text-align:left;font-size:11px}.digit-table th{color:#94A3B8;background:#121A2B;font-weight:600}.digit-table td{color:#CBD5E1}.digit-table .digit-open{font:700 14px var(--font-num);color:#E8EEF9;letter-spacing:1px}.digit-table .digit-ok{color:#34D399;font-weight:700}.digit-table .digit-fail{color:#F87171}.digit-disclaimer,.digit-muted{margin-top:12px;color:#64748B;font-size:11px;line-height:1.6}@media(max-width:900px){.digit-pred-grid{grid-template-columns:repeat(3,minmax(0,1fr))}.digit-summary{grid-template-columns:1fr}.digit-stats{grid-template-columns:repeat(2,minmax(0,1fr))}}@media(max-width:560px){.digit-pred-grid{grid-template-columns:repeat(2,minmax(0,1fr))}.digit-hero h1{font-size:30px}.digit-section-head{display:block}.digit-section-head span{display:block;margin-top:6px}.digit-stats{grid-template-columns:1fr}}
 .digit-recommend-section{margin-top:4px}.digit-recommend-grid{display:grid;grid-template-columns:repeat(5,minmax(0,1fr));gap:12px}.digit-recommend-card{position:relative;padding:16px 14px;border-radius:14px;background:linear-gradient(145deg,rgba(30,41,59,.9),rgba(15,23,42,.95));border:1px solid rgba(96,165,250,.24);min-height:138px;display:flex;flex-direction:column;justify-content:center;align-items:center;gap:7px}.digit-recommend-card:nth-child(-n+3){border-color:rgba(52,211,153,.4);box-shadow:0 10px 24px -14px rgba(52,211,153,.65)}.digit-recommend-rank{position:absolute;left:10px;top:8px;color:#64748B;font:700 11px var(--font-num)}.digit-recommend-number{font:800 27px var(--font-num);letter-spacing:2px;color:#E8EEF9}.digit-recommend-meta{font:600 11px var(--font-num);color:#67E8F9}.digit-recommend-reasons{font-size:10px;line-height:1.5;text-align:center;color:#94A3B8}.digit-recommend-card:nth-child(3n+2) .digit-recommend-number{color:#A78BFA}.digit-recommend-card:nth-child(3n+3) .digit-recommend-number{color:#FBBF24}@media(max-width:900px){.digit-recommend-grid{grid-template-columns:repeat(3,minmax(0,1fr))}}@media(max-width:560px){.digit-recommend-grid{grid-template-columns:repeat(2,minmax(0,1fr))}.digit-recommend-number{font-size:23px}}
</style>`
}
