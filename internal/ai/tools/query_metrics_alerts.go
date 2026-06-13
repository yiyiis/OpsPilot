/*
=== AI Agent 概念：Prometheus 告警查询工具 ===

让 LLM 能够查询 Prometheus 监控系统的活跃告警。
展示了较复杂的工具设计：多层嵌套数据结构、数据去重、时间计算。

核心原理：
  - 调用 Prometheus HTTP API 获取活跃告警列表
  - 对原始告警数据做简化和去重（相同 alertname 只保留第一个）
  - 计算告警持续时间（从激活时间到当前）
  - 格式化为简洁的 JSON 返回给 LLM

本文件的角色：
  提供 Prometheus 告警查询工具，LLM 可以在分析告警时调用。

⚠️ 当前状态：HTTP 调用被禁用（直接返回空结果）
  queryPrometheusAlerts 函数开头有 return 语句，实际 HTTP 调用代码被跳过。
  需要启动本地 Prometheus 容器后，删除那个 return 才能正常工作。

数据结构层次：
  PrometheusAlert  → 原始 Prometheus 告警格式（Labels, Annotations, State...）
  PrometheusAlertsResult → Prometheus API 响应包装
  SimplifiedAlert → 简化后的告警（去除冗余字段，增加计算字段 Duration）
  PrometheusAlertsOutput → 最终工具输出（Success + Alerts + Message）

关联文件：
  - executor.go — Plan-Execute-Replan 执行器使用此工具
  - flow.go — ReAct Agent 也使用此工具
*/
package tools

import (
	"OpsPilot/utility/config"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// PrometheusAlert 原始 Prometheus 告警数据结构
//
// 【AI 概念】为什么需要多层嵌套结构？
// 外部 API 的原始响应通常包含大量字段，不适合直接发给 LLM。
// 需要：原始结构 → 简化结构 → 工具输出结构，逐步精炼。
type PrometheusAlert struct {
	Labels      map[string]string `json:"labels"`      // 告警标签（如 alertname、severity 等）
	Annotations map[string]string `json:"annotations"` // 告警注解（如 description、summary 等）
	State       string            `json:"state"`       // 告警状态（firing、pending 等）
	ActiveAt    string            `json:"activeAt"`    // 激活时间（RFC3339 格式）
	Value       string            `json:"value"`       // 告警触发时的指标值
}

// PrometheusAlertsResult Prometheus API 原始响应结构
type PrometheusAlertsResult struct {
	Status string `json:"status"`
	Data   struct {
		Alerts []PrometheusAlert `json:"alerts"`
	} `json:"data"`
	Error     string `json:"error,omitempty"`
	ErrorType string `json:"errorType,omitempty"`
}

// SimplifiedAlert 简化后的告警信息
//
// 【AI 概念】数据简化（给 LLM 的高质量信息）
// 原始 Prometheus 告警有很多 LLM 不需要的字段。
// SimplifiedAlert 只保留 LLM 做推理所需的关键信息，
// 并新增了 Duration（持续时间）这个计算字段。
type SimplifiedAlert struct {
	AlertName   string `json:"alert_name" jsonschema:"description=告警名称，从 Prometheus 告警的 labels.alertname 字段提取"`
	Description string `json:"description" jsonschema:"description=告警描述信息，从 Prometheus 告警的 annotations.description 字段提取"`
	State       string `json:"state" jsonschema:"description=告警状态，通常为 'firing'（触发中）或 'pending'（待触发）"`
	ActiveAt    string `json:"active_at" jsonschema:"description=告警激活时间，RFC3339 格式的时间戳，例如 '2025-10-29T08:48:42.496134755Z'"`
	Duration    string `json:"duration" jsonschema:"description=告警持续时间，从激活时间到当前时间的时长，格式如 '2h30m15s'、'30m15s' 或 '15s'"`
}

// PrometheusAlertsOutput 告警查询的最终工具输出
//
// 【AI 概念】工具输出结构设计
// 包含 Success 标志、数据列表和状态消息。
// 这种统一的输出格式让 LLM 可以一致地处理成功和失败情况。
type PrometheusAlertsOutput struct {
	Success bool              `json:"success" jsonschema:"description=查询是否成功"`
	Alerts  []SimplifiedAlert `json:"alerts,omitempty" jsonschema:"description=活动告警列表，每个告警包含名称、描述、状态、激活时间和持续时间。相同 alertname 的告警只保留第一个"`
	Message string            `json:"message,omitempty" jsonschema:"description=操作结果的状态消息"`
	Error   string            `json:"error,omitempty" jsonschema:"description=如果查询失败，包含错误信息"`
}

// queryPrometheusAlerts 查询 Prometheus 活跃告警
//
// ⚠️ 当前被禁用：函数开头直接 return 空结果
// 实际 HTTP 调用代码在 return 语句之后（dead code）
// 需要启动 Prometheus 容器后删除那个 return 才能工作
func queryPrometheusAlerts() (PrometheusAlertsResult, error) {
	// 开关：直接返回空结果（跳过下面的实际 HTTP 调用）
	return PrometheusAlertsResult{}, nil

	// 下面的代码当前不会执行（Prometheus 地址从 config.yaml 读取）
	baseURL := config.App.Prometheus.BaseURL
	apiURL := fmt.Sprintf("%s/api/v1/alerts", baseURL)

	log.Printf("Querying Prometheus alerts: %s", apiURL)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	var result PrometheusAlertsResult

	resp, err := client.Get(apiURL)
	if err != nil {
		return result, fmt.Errorf("failed to query Prometheus alerts: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return result, fmt.Errorf("failed to read response: %v", err)
	}

	if err = json.Unmarshal(body, &result); err != nil {
		return result, fmt.Errorf("failed to parse response: %v", err)
	}

	return result, nil
}

// calculateDuration 计算从 activeAt 到现在的持续时间
//
// 将 RFC3339Nano 时间戳转为人类可读的时长格式（如 "2h30m15s"）
func calculateDuration(activeAtStr string) string {
	activeAt, err := time.Parse(time.RFC3339Nano, activeAtStr)
	if err != nil {
		return "unknown"
	}

	duration := time.Since(activeAt)

	hours := int(duration.Hours())
	minutes := int(duration.Minutes()) % 60
	seconds := int(duration.Seconds()) % 60

	if hours > 0 {
		return fmt.Sprintf("%dh%dm%ds", hours, minutes, seconds)
	} else if minutes > 0 {
		return fmt.Sprintf("%dm%ds", minutes, seconds)
	} else {
		return fmt.Sprintf("%ds", seconds)
	}
}

// NewPrometheusAlertsQueryTool 创建 Prometheus 告警查询工具
//
// 【AI 概念】复杂工具的处理流程
// 1. 调用 Prometheus API 获取原始告警
// 2. 按 alertname 去重（相同名称的告警只保留第一个）
// 3. 简化数据结构（提取关键字段，计算持续时间）
// 4. 格式化为 JSON 返回
//
// 输入：无需参数（查询所有活跃告警）
// 输出：JSON 格式的 SimplifiedAlert 列表
func NewPrometheusAlertsQueryTool() tool.InvokableTool {
	t, err := utils.InferOptionableTool(
		"query_prometheus_alerts",
		"Query active alerts from Prometheus alerting system. This tool retrieves all currently active/firing alerts including their labels, annotations, state, and values. Use this tool when you need to check what alerts are currently firing, investigate alert conditions, or monitor alert status.",
		func(ctx context.Context, input *struct{}, opts ...tool.Option) (output string, err error) {
			log.Printf("Querying Prometheus active alerts")

			// 步骤1：调用 Prometheus API
			result, err := queryPrometheusAlerts()
			if err != nil {
				alertsOut := PrometheusAlertsOutput{
					Success: false,
					Error:   err.Error(),
					Message: "Failed to query Prometheus alerts",
				}
				jsonBytes, _ := json.MarshalIndent(alertsOut, "", "  ")
				return string(jsonBytes), err
			}

			// 步骤2：转换为简化格式，按 alertname 去重
			seenAlertNames := make(map[string]bool)
			simplifiedAlerts := make([]SimplifiedAlert, 0)
			for _, alert := range result.Data.Alerts {
				alertName := alert.Labels["alertname"]

				// 相同 alertname 只保留第一个（去重）
				if seenAlertNames[alertName] {
					continue
				}
				seenAlertNames[alertName] = true

				// 提取关键字段并计算持续时间
				simplified := SimplifiedAlert{
					AlertName:   alertName,
					Description: alert.Annotations["description"],
					State:       alert.State,
					ActiveAt:    alert.ActiveAt,
					Duration:    calculateDuration(alert.ActiveAt),
				}
				simplifiedAlerts = append(simplifiedAlerts, simplified)
			}

			// 步骤3：构建成功响应
			alertsOut := PrometheusAlertsOutput{
				Success: true,
				Alerts:  simplifiedAlerts,
				Message: fmt.Sprintf("Successfully retrieved %d active alerts", len(simplifiedAlerts)),
			}

			// 步骤4：格式化为 JSON 返回
			jsonBytes, err := json.MarshalIndent(alertsOut, "", "  ")
			if err != nil {
				log.Printf("Error marshaling alerts result to JSON: %v", err)
				return "", err
			}

			log.Printf("Prometheus alerts query completed: %d alerts found", len(simplifiedAlerts))
			return string(jsonBytes), nil
		})
	if err != nil {
		log.Fatal(err)
	}
	return t
}
