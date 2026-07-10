package metrics

import (
	"awesome/internal/inf/evaluation/config"
	"testing"
	"time"
)

func TestNewMetricsCalculator(t *testing.T) {
	configs := config.DefaultMetricConfigs()
	calculator := NewMetricsCalculator(configs)

	if calculator == nil {
		t.Fatal("Expected calculator to be created, but got nil")
	}

	if calculator.configs == nil {
		t.Error("Expected configs map to be initialized")
	}

	// Verify default configs are loaded
	if len(calculator.configs) != len(configs) {
		t.Errorf("Expected %d configs, got %d", len(configs), len(calculator.configs))
	}
}

func TestMetricsCalculator_CalculateMetrics(t *testing.T) {
	calculator := NewMetricsCalculator(config.DefaultMetricConfigs())

	// Test with binary classification data
	predictions := []interface{}{true, true, false, true, false, false, true, false}
	labels := []interface{}{true, true, false, false, false, true, true, false}

	results, err := calculator.CalculateMetrics(predictions, labels)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify basic metrics
	if results["total_predictions"] != 8 {
		t.Errorf("Expected total_predictions to be 8, got %f", results["total_predictions"])
	}

	// Verify confusion matrix values
	if results["true_positives"] != 3 {
		t.Errorf("Expected true_positives to be 3, got %f", results["true_positives"])
	}

	if results["false_positives"] != 1 {
		t.Errorf("Expected false_positives to be 1, got %f", results["false_positives"])
	}

	if results["true_negatives"] != 3 {
		t.Errorf("Expected true_negatives to be 3, got %f", results["true_negatives"])
	}

	if results["false_negatives"] != 1 {
		t.Errorf("Expected false_negatives to be 1, got %f", results["false_negatives"])
	}

	// Verify accuracy: (TP + TN) / Total = (3 + 3) / 8 = 0.75
	expectedAccuracy := 0.75
	if results["accuracy"] != expectedAccuracy {
		t.Errorf("Expected accuracy to be %f, got %f", expectedAccuracy, results["accuracy"])
	}

	// Verify precision: TP / (TP + FP) = 3 / (3 + 1) = 0.75
	expectedPrecision := 0.75
	if results["precision"] != expectedPrecision {
		t.Errorf("Expected precision to be %f, got %f", expectedPrecision, results["precision"])
	}

	// Verify recall: TP / (TP + FN) = 3 / (3 + 1) = 0.75
	expectedRecall := 0.75
	if results["recall"] != expectedRecall {
		t.Errorf("Expected recall to be %f, got %f", expectedRecall, results["recall"])
	}

	// Verify F1 score: 2 * (precision * recall) / (precision + recall) = 0.75
	expectedF1 := 0.75
	if results["f1_score"] != expectedF1 {
		t.Errorf("Expected f1_score to be %f, got %f", expectedF1, results["f1_score"])
	}
}

func TestMetricsCalculator_CalculateMetrics_EmptyInput(t *testing.T) {
	calculator := NewMetricsCalculator(config.DefaultMetricConfigs())

	predictions := []interface{}{}
	labels := []interface{}{}

	results, err := calculator.CalculateMetrics(predictions, labels)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if results["total_predictions"] != 0 {
		t.Errorf("Expected total_predictions to be 0, got %f", results["total_predictions"])
	}

	if _, exists := results["accuracy"]; exists {
		t.Error("Expected accuracy to not exist for empty input")
	}
}

func TestMetricsCalculator_CalculateLatencyMetrics(t *testing.T) {
	calculator := NewMetricsCalculator(config.DefaultMetricConfigs())

	latencies := []time.Duration{
		10 * time.Millisecond,
		20 * time.Millisecond,
		30 * time.Millisecond,
		40 * time.Millisecond,
		50 * time.Millisecond,
		60 * time.Millisecond,
		70 * time.Millisecond,
		80 * time.Millisecond,
		90 * time.Millisecond,
		100 * time.Millisecond,
	}

	results := calculator.CalculateLatencyMetrics(latencies)

	if len(results) == 0 {
		t.Error("Expected latency metrics to be calculated")
	}

	// Verify min and max
	if results["min_latency_ms"] != 10.0 {
		t.Errorf("Expected min_latency_ms to be 10.0, got %f", results["min_latency_ms"])
	}

	if results["max_latency_ms"] != 100.0 {
		t.Errorf("Expected max_latency_ms to be 100.0, got %f", results["max_latency_ms"])
	}

	// Verify median (should be around 55ms for 10 values)
	if results["median_latency_ms"] < 50 || results["median_latency_ms"] > 60 {
		t.Errorf("Expected median_latency_ms to be around 55, got %f", results["median_latency_ms"])
	}

	// Verify percentiles are in correct order
	if results["p50_latency_ms"] > results["p90_latency_ms"] {
		t.Error("Expected p50 to be less than or equal to p90")
	}

	if results["p90_latency_ms"] > results["p95_latency_ms"] {
		t.Error("Expected p90 to be less than or equal to p95")
	}

	if results["p95_latency_ms"] > results["p99_latency_ms"] {
		t.Error("Expected p95 to be less than or equal to p99")
	}
}

func TestMetricsCalculator_CalculateLatencyMetrics_EmptyInput(t *testing.T) {
	calculator := NewMetricsCalculator(config.DefaultMetricConfigs())

	latencies := []time.Duration{}
	results := calculator.CalculateLatencyMetrics(latencies)

	if len(results) != 0 {
		t.Error("Expected empty results for empty input")
	}
}

func TestMetricsCalculator_CalculateThroughput(t *testing.T) {
	calculator := NewMetricsCalculator(config.DefaultMetricConfigs())

	// Test normal case
	requests := int64(1000)
	duration := 10 * time.Second
	throughput := calculator.CalculateThroughput(requests, duration)

	expectedThroughput := 100.0 // 1000 requests / 10 seconds
	if throughput != expectedThroughput {
		t.Errorf("Expected throughput to be %f, got %f", expectedThroughput, throughput)
	}

	// Test zero duration
	throughputZero := calculator.CalculateThroughput(requests, 0)
	if throughputZero != 0 {
		t.Errorf("Expected throughput to be 0 for zero duration, got %f", throughputZero)
	}
}

func TestMetricsCalculator_CalculateAvailability(t *testing.T) {
	calculator := NewMetricsCalculator(config.DefaultMetricConfigs())

	// Test normal case
	successCount := int64(999)
	totalCount := int64(1000)
	availability := calculator.CalculateAvailability(successCount, totalCount)

	expectedAvailability := 99.9 // 999/1000 * 100
	if availability != expectedAvailability {
		t.Errorf("Expected availability to be %f, got %f", expectedAvailability, availability)
	}

	// Test perfect availability
	availability = calculator.CalculateAvailability(1000, 1000)
	if availability != 100.0 {
		t.Errorf("Expected availability to be 100.0, got %f", availability)
	}

	// Test zero total count
	availability = calculator.CalculateAvailability(0, 0)
	if availability != 0 {
		t.Errorf("Expected availability to be 0 for zero total count, got %f", availability)
	}
}

func TestMetricsCalculator_CalculateWeightedScore(t *testing.T) {
	configs := []*config.MetricConfig{
		{Name: "accuracy", DisplayName: "准确率", Weight: 1.0, Enabled: true, Threshold: 0.9},
		{Name: "precision", DisplayName: "精确率", Weight: 0.8, Enabled: true, Threshold: 0.85},
		{Name: "recall", DisplayName: "召回率", Weight: 0.8, Enabled: true, Threshold: 0.85},
	}

	calculator := NewMetricsCalculator(configs)

	metrics := map[string]float64{
		"accuracy":  0.95,
		"precision": 0.93,
		"recall":    0.92,
	}

	overallScore, scores := calculator.CalculateWeightedScore(metrics)

	if overallScore <= 0 {
		t.Error("Expected overall score to be positive")
	}

	if len(scores) != 3 {
		t.Errorf("Expected 3 individual scores, got %d", len(scores))
	}

	for _, score := range scores {
		if score < 0 || score > 100 {
			t.Errorf("Expected score to be between 0 and 100, got %f", score)
		}
	}
}

func TestMetricsCalculator_CalculateWeightedScore_WithDisabledMetrics(t *testing.T) {
	configs := []*config.MetricConfig{
		{Name: "accuracy", DisplayName: "准确率", Weight: 1.0, Enabled: true, Threshold: 0.9},
		{Name: "precision", DisplayName: "精确率", Weight: 0.8, Enabled: false, Threshold: 0.85},
	}

	calculator := NewMetricsCalculator(configs)

	metrics := map[string]float64{
		"accuracy":  0.95,
		"precision": 0.93,
	}

	overallScore, scores := calculator.CalculateWeightedScore(metrics)

	// Only accuracy should be included
	if _, exists := scores["precision"]; exists {
		t.Error("Expected disabled metric to not be included in scores")
	}

	if _, exists := scores["accuracy"]; !exists {
		t.Error("Expected enabled metric to be included in scores")
	}

	_ = overallScore
}

func TestMetricsCalculator_EvaluateMetric(t *testing.T) {
	configs := []*config.MetricConfig{
		{Name: "accuracy", DisplayName: "准确率", Weight: 1.0, Enabled: true, Threshold: 0.9},
		{Name: "latency_p95", DisplayName: "P95 延迟", Weight: 0.7, Enabled: true, Threshold: 500},
	}

	calculator := NewMetricsCalculator(configs)

	// Test passing metric
	result := calculator.EvaluateMetric("accuracy", 0.95)
	if result.Name != "accuracy" {
		t.Errorf("Expected metric name 'accuracy', got %s", result.Name)
	}

	if result.Value != 0.95 {
		t.Errorf("Expected metric value 0.95, got %f", result.Value)
	}

	if !result.Passed {
		t.Error("Expected accuracy metric to pass")
	}

	// Test failing metric
	result = calculator.EvaluateMetric("accuracy", 0.8)
	if result.Passed {
		t.Error("Expected accuracy metric to fail")
	}

	// Test latency metric (lower is better)
	result = calculator.EvaluateMetric("latency_p95", 400)
	if !result.Passed {
		t.Error("Expected latency metric to pass (400 < 500)")
	}

	result = calculator.EvaluateMetric("latency_p95", 600)
	if result.Passed {
		t.Error("Expected latency metric to fail (600 > 500)")
	}
}

func TestMetricsCalculator_EvaluateMetric_UnknownMetric(t *testing.T) {
	calculator := NewMetricsCalculator(config.DefaultMetricConfigs())

	result := calculator.EvaluateMetric("unknown_metric", 123.45)

	if result.Name != "unknown_metric" {
		t.Errorf("Expected metric name 'unknown_metric', got %s", result.Name)
	}

	if result.Value != 123.45 {
		t.Errorf("Expected metric value 123.45, got %f", result.Value)
	}

	if result.Weight != 0 {
		t.Errorf("Expected weight to be 0 for unknown metric, got %f", result.Weight)
	}
}

func TestMetricsCalculator_AddCustomMetric(t *testing.T) {
	calculator := NewMetricsCalculator(config.DefaultMetricConfigs())

	customConfig := &config.MetricConfig{
		Name:        "custom_metric",
		DisplayName: "自定义指标",
		Weight:      0.5,
		Enabled:     true,
		Threshold:   0.8,
		Formula:     "custom_formula",
	}

	calculator.AddCustomMetric(customConfig)

	cfg, exists := calculator.GetMetricConfig("custom_metric")
	if !exists {
		t.Error("Expected custom metric to be added")
	}

	if cfg.DisplayName != customConfig.DisplayName {
		t.Errorf("Expected display name %s, got %s", customConfig.DisplayName, cfg.DisplayName)
	}

	if cfg.Weight != customConfig.Weight {
		t.Errorf("Expected weight %f, got %f", customConfig.Weight, cfg.Weight)
	}
}

func TestMetricsCalculator_UpdateMetricWeight(t *testing.T) {
	calculator := NewMetricsCalculator(config.DefaultMetricConfigs())

	// Update existing metric
	err := calculator.UpdateMetricWeight("accuracy", 0.5)
	if err != nil {
		t.Fatalf("Unexpected error updating metric weight: %v", err)
	}

	cfg, exists := calculator.GetMetricConfig("accuracy")
	if !exists {
		t.Error("Expected accuracy metric to exist")
	}

	if cfg.Weight != 0.5 {
		t.Errorf("Expected weight to be 0.5, got %f", cfg.Weight)
	}

	// Try to update non-existent metric
	err = calculator.UpdateMetricWeight("non_existent", 0.5)
	if err == nil {
		t.Error("Expected error when updating non-existent metric")
	}
}

func TestMetricsCalculator_GetAllMetricConfigs(t *testing.T) {
	configs := config.DefaultMetricConfigs()
	calculator := NewMetricsCalculator(configs)

	allConfigs := calculator.GetAllMetricConfigs()

	if len(allConfigs) != len(configs) {
		t.Errorf("Expected %d configs, got %d", len(configs), len(allConfigs))
	}
}

func TestMetricsCalculator_Mean(t *testing.T) {
	calculator := NewMetricsCalculator(config.DefaultMetricConfigs())

	values := []float64{10, 20, 30, 40, 50}
	mean := calculator.mean(values)

	expectedMean := 30.0
	if mean != expectedMean {
		t.Errorf("Expected mean to be %f, got %f", expectedMean, mean)
	}

	// Test empty slice
	mean = calculator.mean([]float64{})
	if mean != 0 {
		t.Errorf("Expected mean of empty slice to be 0, got %f", mean)
	}
}

func TestMetricsCalculator_StdDev(t *testing.T) {
	calculator := NewMetricsCalculator(config.DefaultMetricConfigs())

	values := []float64{10, 20, 30, 40, 50}
	stdDev := calculator.stdDev(values)

	// Expected std dev for [10, 20, 30, 40, 50] is sqrt(200) ≈ 14.14
	if stdDev < 14 || stdDev > 15 {
		t.Errorf("Expected std dev to be around 14.14, got %f", stdDev)
	}

	// Test empty slice
	stdDev = calculator.stdDev([]float64{})
	if stdDev != 0 {
		t.Errorf("Expected std dev of empty slice to be 0, got %f", stdDev)
	}
}

func TestMetricsCalculator_Percentile(t *testing.T) {
	calculator := NewMetricsCalculator(config.DefaultMetricConfigs())

	sortedValues := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	p50 := calculator.percentile(sortedValues, 50)
	if p50 < 5 || p50 > 6 {
		t.Errorf("Expected p50 to be around 5.5, got %f", p50)
	}

	p90 := calculator.percentile(sortedValues, 90)
	if p90 < 9 || p90 > 10 {
		t.Errorf("Expected p90 to be around 9.1, got %f", p90)
	}

	p99 := calculator.percentile(sortedValues, 99)
	if p99 < 9 || p99 > 10 {
		t.Errorf("Expected p99 to be close to 10, got %f", p99)
	}

	// Test empty slice
	p := calculator.percentile([]float64{}, 50)
	if p != 0 {
		t.Errorf("Expected percentile of empty slice to be 0, got %f", p)
	}
}

func TestMetricsCalculator_ToBool(t *testing.T) {
	calculator := NewMetricsCalculator(config.DefaultMetricConfigs())

	tests := []struct {
		input    interface{}
		expected bool
	}{
		{true, true},
		{false, false},
		{int(1), true},
		{int(0), false},
		{int64(1), true},
		{int64(0), false},
		{float64(1.5), true},
		{float64(0.0), false},
		{"true", true},
		{"1", true},
		{"yes", true},
		{"false", false},
		{"0", false},
		{"no", false},
		{"", false},
		{nil, false},
	}

	for _, test := range tests {
		result := calculator.toBool(test.input)
		if result != test.expected {
			t.Errorf("toBool(%v) [type: %T] = %v, expected %v", test.input, test.input, result, test.expected)
		}
	}
}
