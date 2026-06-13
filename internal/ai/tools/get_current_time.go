/*
=== AI Agent 概念：基础工具（Tool Use 入门示例）===

这是项目中最简单的 Agent 工具，适合作为学习 Tool Use 的入口。
它不需要任何输入参数，直接返回当前时间的多种格式。

核心原理：
  - 工具定义 = 工具名 + 描述 + 输入 Schema + 处理函数
  - LLM 通过工具描述了解工具的用途，自主决定是否调用
  - 工具的输入/输出 Schema 使用 jsonschema 标签定义（给 LLM 看的函数签名）
  - InferOptionableTool 自动从 Go 结构体生成 Tool Schema

本文件的角色：
  提供一个最简单的工具示例，展示 Tool Use 的标准定义模式。

工具定义模式（本项目通用）：
  1. 定义 Input 结构体（jsonschema 标签描述参数）
  2. 定义 Output 结构体（jsonschema 标签描述返回值）
  3. 使用 utils.InferOptionableTool 包装处理函数
  4. 返回 tool.InvokableTool 接口

关联文件：
  - query_internal_docs.go — 工具内部使用 RAG 的进阶示例
  - query_log.go — 使用 MCP 协议的外部工具示例
*/
package tools

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// GetCurrentTimeInput 获取当前时间的输入参数（无需输入）
//
// 【AI 概念】Tool Input Schema
// 这个结构体定义了工具的输入参数。jsonschema 标签中的 description
// 会被发送给 LLM，帮助它理解每个参数的含义。
// 即使没有参数，也需要定义一个空结构体——LLM 需要知道"这个工具不需要参数"。
type GetCurrentTimeInput struct {
	// 无需输入参数
}

// GetCurrentTimeOutput 获取当前时间的输出结果
//
// 【AI 概念】Tool Output Schema
// 输出结构体的 jsonschema 标签告诉 LLM 工具会返回什么数据。
// LLM 根据这些描述来理解和利用工具返回的结果。
type GetCurrentTimeOutput struct {
	Success      bool   `json:"success" jsonschema:"description=Indicates whether the time retrieval was successful"`
	Seconds      int64  `json:"seconds" jsonschema:"description=Current Unix timestamp in seconds since epoch (1970-01-01 00:00:00 UTC)"`
	Milliseconds int64  `json:"milliseconds" jsonschema:"description=Current Unix timestamp in milliseconds since epoch (1970-01-01 00:00:00 UTC)"`
	Microseconds int64  `json:"microseconds" jsonschema:"description=Current Unix timestamp in microseconds since epoch (1970-01-01 00:00:00 UTC)"`
	Timestamp    string `json:"timestamp" jsonschema:"description=Human-readable timestamp in format 'YYYY-MM-DD HH:MM:SS.microseconds'"`
	Message      string `json:"message" jsonschema:"description=Status message describing the operation result"`
}

// NewGetCurrentTimeTool 创建获取当前时间的工具
//
// 【AI 概念】InferOptionableTool 的工作方式
//
// 这个函数做了三件事：
//   1. 参数1："get_current_time" — 工具名称（LLM 看到的函数名）
//   2. 参数2：工具描述（LLM 看到的功能说明，帮助它决定何时调用）
//   3. 参数3：处理函数 — LLM 决定调用时，实际执行的 Go 函数
//
// InferOptionableTool 会：
//   - 从 Input 结构体的 jsonschema 标签自动生成 JSON Schema
//   - 将工具描述和 Schema 注册到 LLM 的工具列表中
//   - LLM 调用工具时，自动将 JSON 参数反序列化为 Input 结构体
func NewGetCurrentTimeTool() tool.InvokableTool {
	t, err := utils.InferOptionableTool(
		"get_current_time", // 工具名：LLM 看到的是这个名称
		"Get current system time in multiple formats. Returns the current time in seconds (Unix timestamp), milliseconds, and microseconds. Use this tool when you need to retrieve current system time for logging, timing operations, or timestamping events.",
		func(ctx context.Context, input *GetCurrentTimeInput, opts ...tool.Option) (output string, err error) {
			// 获取当前时间
			now := time.Now()

			// 计算各种时间格式
			seconds := now.Unix()                                 // 秒级时间戳
			milliseconds := now.UnixMilli()                       // 毫秒级时间戳
			microseconds := now.UnixMicro()                       // 微秒级时间戳
			timestamp := now.Format("2006-01-02 15:04:05.000000") // 可读格式

			log.Printf("Getting current time: %s", timestamp)

			// 构建输出
			timeOutput := GetCurrentTimeOutput{
				Success:      true,
				Seconds:      seconds,
				Milliseconds: milliseconds,
				Microseconds: microseconds,
				Timestamp:    timestamp,
				Message:      "Current time retrieved successfully",
			}

			// 工具返回值必须是 JSON 字符串
			// LLM 接收 JSON 字符串后自行解析和理解
			jsonBytes, err := json.MarshalIndent(timeOutput, "", "  ")
			if err != nil {
				log.Printf("Error marshaling result to JSON: %v", err)
				return "", err
			}

			log.Printf("Current time: Seconds=%d, Milliseconds=%d, Microseconds=%d", seconds, milliseconds, microseconds)
			return string(jsonBytes), nil
		})

	if err != nil {
		log.Fatal(err)
	}

	return t
}
