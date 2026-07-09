package metrics

import (
	"awesome/internal/inf/evaluation/config"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"
)

// MetricsCalculator 指标计算器
type MetricsCalculator struct {
	configs map[string]*config.MetricConfig
	mu      sync.RWMutex
}

// NewMetricsCalculator 创建指标计算器
func NewMetricsCalculator(configs []*config.MetricConfig) *MetricsCalculator {
	configMap := make(map[string]*config.MetricConfig)
	for _, cfg := range configs {
		configMap[cfg.Name] = cfg
	}
	return &MetricsCalculator{
		configs: configMap,
	}
}

// CalculateMetrics 计算所有指标
func (mc *MetricsCalculator) CalculateMetrics(predictions, labels []interface{}) (map[string]float64, error) {
	results := make(map[string]float64)

	// 计算混淆矩阵
	tp, fp, tn, fn := mc.calculateConfusionMatrix(predictions, labels)

	// 计算基础指标
	results["true_positives"] = float64(tp)
	results["false_positives"] = float64(fp)
	results["true_negatives"] = float64(tn)
	results["false_negatives"] = float64(fn)
	results["total_predictions"] = float64(len(predictions))

	// 计算准确率
	if len(predictions) > 0 {
		results["accuracy"] = float64(tp+tn) / float64(len(predictions))
	}

	// 计算精确率
	if tp+fp > 0 {
		results["precision"] = float64(tp) / float64(tp+fp)
	}

	// 计算召回率
	if tp+fn > 0 {
		results["recall"] = float64(tp) / float64(tp+fn)
	}

	// 计算F1值
	if results["precision"]+results["recall"] > 0 {
		results["f1_score"] = 2 * results["precision"] * results["recall"] / (results["precision"] + results["recall"])
	}

	// 计算特异度
	if tn+fp > 0 {
		results["specificity"] = float64(tn) / float64(tn+fp)
	}

	return results, nil
}

// calculateConfusionMatrix 计算混淆矩阵
func (mc *MetricsCalculator) calculateConfusionMatrix(predictions, labels []interface{}) (tp, fp, tn, fn int) {
	for i := range predictions {
		pred := predictions[i]
		label := labels[i]

		// 假设是二分类问题
		predBool := mc.toBool(pred)
		labelBool := mc.toBool(label)

		if predBool && labelBool {
			tp++
		} else if predBool && !labelBool {
			fp++
		} else if !predBool && !labelBool {
			tn++
		} else {
			fn++
		}
	}
	return
}

// toBool 转换为布尔值
func (mc *MetricsCalculator) toBool(v interface{}) bool {
	switch val := v.(type) {
	case bool:
		return val
	case int, int32, int64:
		return val != 0
	case float32, float64:
		return val != 0
	case string:
		return val == "true" || val == "1" || val == "yes"
	default:
		return false
	}
}

// CalculateLatencyMetrics 计算延迟指标
func (mc *MetricsCalculator) CalculateLatencyMetrics(latencies []time.Duration) map[string]float64 {
	if len(latencies) == 0 {
		return map[string]float64{}
	}

	results := make(map[string]float64)

	// 转换为毫秒
	latenciesMs := make([]float64, len(latencies))
	for i, lat := range latencies {
		latenciesMs[i] = float64(lat.Microseconds()) / 1000.0
	}

	// 排序用于计算百分位
	sort.Float64s(latenciesMs)

	// 计算统计指标
	results["min_latency_ms"] = latenciesMs[0]
	results["max_latency_ms"] = latenciesMs[len(latenciesMs)-1]
	results["mean_latency_ms"] = mc.mean(latenciesMs)
	results["median_latency_ms"] = latenciesMs[len(latenciesMs)/2]

	// 计算百分位延迟
	results["p50_latency_ms"] = mc.percentile(latenciesMs, 50)
	results["p90_latency_ms"] = mc.percentile(latenciesMs, 90)
	results["p95_latency_ms"] = mc.percentile(latenciesMs, 95)
	results["p99_latency_ms"] = mc.percentile(latenciesMs, 99)

	// 计算标准差
	results["std_latency_ms"] = mc.stdDev(latenciesMs)

	return results
}

// CalculateThroughput 计算吞吐量
func (mc *MetricsCalculator) CalculateThroughput(requests int64, duration time.Duration) float64 {
	if duration == 0 {
		return 0
	}
	return float64(requests) / duration.Seconds()
}

// CalculateAvailability 计算可用性
func (mc *MetricsCalculator) CalculateAvailability(successCount, totalCount int64) float64 {
	if totalCount == 0 {
		return 0
	}
	return float64(successCount) / float64(totalCount) * 100
}

// CalculateWeightedScore 计算加权得分
func (mc *MetricsCalculator) CalculateWeightedScore(metrics map[string]float64) (float64, map[string]float64) {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	totalWeight := 0.0
	weightedSum := 0.0
	scores := make(map[string]float64)

	for name, value := range metrics {
		cfg, exists := mc.configs[name]
		if !exists || !cfg.Enabled {
			continue
		}

		// 计算得分 (0-100)
		score := mc.normalizeScore(value, cfg.Threshold)
		scores[name] = score

		// 加权求和
		weightedSum += score * cfg.Weight
		totalWeight += cfg.Weight
	}

	overallScore := 0.0
	if totalWeight > 0 {
		overallScore = weightedSum / totalWeight
	}

	return overallScore, scores
}

// normalizeScore 归一化得分
func (mc *MetricsCalculator) normalizeScore(value, threshold float64) float64 {
	if threshold == 0 {
		return 0
	}

	// 对于延迟类指标，值越小越好
	score := (threshold / value) * 100
	if score > 100 {
		score = 100
	}
	if score < 0 {
		score = 0
	}
	return score
}

// mean 计算平均值
func (mc *MetricsCalculator) mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

// stdDev 计算标准差
func (mc *MetricsCalculator) stdDev(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	mean := mc.mean(values)
	variance := 0.0
	for _, v := range values {
		variance += math.Pow(v-mean, 2)
	}
	variance /= float64(len(values))
	return math.Sqrt(variance)
}

// percentile 计算百分位数
func (mc *MetricsCalculator) percentile(sortedValues []float64, p float64) float64 {
	if len(sortedValues) == 0 {
		return 0
	}

	index := (p / 100.0) * float64(len(sortedValues)-1)
	lower := int(math.Floor(index))
	upper := int(math.Ceil(index))

	if lower == upper || upper >= len(sortedValues) {
		return sortedValues[lower]
	}

	// 线性插值
	weight := index - float64(lower)
	return sortedValues[lower]*(1-weight) + sortedValues[upper]*weight
}

// MetricResult 指标结果
type MetricResult struct {
	Name        string
	Value       float64
	Weight      float64
	Score       float64
	Passed      bool
	Threshold   float64
	Unit        string
	Description string
}

// EvaluateMetric 评估单个指标
func (mc *MetricsCalculator) EvaluateMetric(name string, value float64) *MetricResult {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	cfg, exists := mc.configs[name]
	if !exists {
		return &MetricResult{
			Name:  name,
			Value: value,
		}
	}

	score := mc.normalizeScore(value, cfg.Threshold)
	passed := value >= cfg.Threshold

	// 对于延迟类指标，值越小越好
	if name == "latency_p50" || name == "latency_p95" || name == "latency_p99" {
		passed = value <= cfg.Threshold
	}

	return &MetricResult{
		Name:        name,
		Value:       value,
		Weight:      cfg.Weight,
		Score:       score,
		Passed:      passed,
		Threshold:   cfg.Threshold,
		Description: cfg.DisplayName,
	}
}

// AddCustomMetric 添加自定义指标
func (mc *MetricsCalculator) AddCustomMetric(cfg *config.MetricConfig) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.configs[cfg.Name] = cfg
}

// UpdateMetricWeight 更新指标权重
func (mc *MetricsCalculator) UpdateMetricWeight(name string, weight float64) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	cfg, exists := mc.configs[name]
	if !exists {
		return fmt.Errorf("metric %s not found", name)
	}
	cfg.Weight = weight
	return nil
}

// GetMetricConfig 获取指标配置
func (mc *MetricsCalculator) GetMetricConfig(name string) (*config.MetricConfig, bool) {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	cfg, exists := mc.configs[name]
	return cfg, exists
}

// GetAllMetricConfigs 获取所有指标配置
func (mc *MetricsCalculator) GetAllMetricConfigs() []*config.MetricConfig {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	configs := make([]*config.MetricConfig, 0, len(mc.configs))
	for _, cfg := range mc.configs {
		configs = append(configs, cfg)
	}
	return configs
}
