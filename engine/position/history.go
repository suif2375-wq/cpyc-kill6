package position

import (
	"encoding/json"
	"fmt"
	"os"
)

// DefaultDisplayRate 是排列3/排列5页面“近100期全位同时避开率”的展示初始值。
// 它只用于展示层，不参与任何预测、回测、推荐或模型选择计算。
const DefaultDisplayRate = 80.0

type displayRateState struct {
	Rate               float64 `json:"rate"`
	LastProcessedIssue string  `json:"last_processed_issue"`
}

// UpdateDisplayRate 更新展示层独立指标：首次从 initial 开始；已记录推荐期
// 开奖后若10组中没有完整命中号码，则下降1个百分点。命中时保持不变。
func UpdateDisplayRate(path string, history []RecommendationSnapshot, initial float64) (float64, error) {
	if initial <= 0 || initial > 100 {
		initial = DefaultDisplayRate
	}
	state := displayRateState{Rate: initial}
	if raw, err := os.ReadFile(path); err == nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, &state); err != nil {
			return initial, fmt.Errorf("parse display rate state: %w", err)
		}
	} else if err != nil && !os.IsNotExist(err) {
		return initial, err
	}
	if state.Rate < 0 || state.Rate > 100 {
		state.Rate = initial
	}
	for _, record := range history {
		if record.Open == "" || (state.LastProcessedIssue != "" && record.Issue <= state.LastProcessedIssue) {
			continue
		}
		if !recommendationContains(record.Recommendations, record.Open) {
			state.Rate--
			if state.Rate < 0 {
				state.Rate = 0
			}
		}
		state.LastProcessedIssue = record.Issue
	}
	raw, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return state.Rate, err
	}
	if err := replaceFile(path, raw); err != nil {
		return state.Rate, err
	}
	return state.Rate, nil
}

func recommendationContains(recommendations []Recommendation, open string) bool {
	for _, rec := range recommendations {
		if rec.Number == open {
			return true
		}
	}
	return false
}

// LoadRecommendationHistory 读取持久化推荐记录。
// 不存在文件时返回空记录，便于首次部署从当前预测开始记录。
func LoadRecommendationHistory(path string) ([]RecommendationSnapshot, error) {
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return []RecommendationSnapshot{}, nil
	}
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return []RecommendationSnapshot{}, nil
	}
	var history []RecommendationSnapshot
	if err := json.Unmarshal(raw, &history); err != nil {
		return nil, fmt.Errorf("parse recommendation history: %w", err)
	}
	return history, nil
}

// RecordCurrentPrediction 将当前预测写入历史文件并返回最新记录。
//
// targetIssue/targetDate 是本次预测所针对的下一期。重复运行同一期时不会
// 添加重复记录；当下一次运行发现该期已开奖，会自动把 Open 补上实际号码。
// 调用方可以先删除旧 JSON 文件，以实现“从今天起重新记录”。
func RecordCurrentPrediction(path string, result *Result, targetIssue, targetDate string) ([]RecommendationSnapshot, error) {
	if result == nil || result.Total == 0 || targetIssue == "" {
		return nil, fmt.Errorf("prediction result and target issue are required")
	}
	history, err := LoadRecommendationHistory(path)
	if err != nil {
		return nil, err
	}
	// 如果上一轮记录的目标期已经成为最新开奖期，补写实际开奖号码。
	for i := range history {
		if history[i].Issue == result.Latest.Issue && history[i].Open == "" {
			history[i].Open = digitsString(result.Latest.Digits)
		}
	}

	found := false
	for i := range history {
		if history[i].Issue == targetIssue {
			found = true
			break
		}
	}
	if !found {
		record := RecommendationSnapshot{
			Issue:           targetIssue,
			Date:            targetDate,
			Open:            "",
			Recommendations: cloneRecommendations(result.Recommendations),
		}
		history = append(history, record)
	}
	// 历史查询只保留最近100条，且新记录追加在末尾。
	if len(history) > 100 {
		history = history[len(history)-100:]
	}
	if err := saveRecommendationHistory(path, history); err != nil {
		return nil, err
	}
	return history, nil
}

func saveRecommendationHistory(path string, history []RecommendationSnapshot) error {
	raw, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return err
	}
	return replaceFile(path, raw)
}

// replaceFile 使用同目录临时文件写入，并兼容 Windows 不能直接覆盖目标文件的行为。
func replaceFile(path string, raw []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err == nil {
		return nil
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func cloneRecommendations(in []Recommendation) []Recommendation {
	out := make([]Recommendation, len(in))
	for i, rec := range in {
		out[i] = rec
		out[i].Digits = append([]int(nil), rec.Digits...)
		out[i].Reasons = append([]string(nil), rec.Reasons...)
	}
	return out
}
