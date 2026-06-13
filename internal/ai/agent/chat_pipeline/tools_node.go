/*
=== AI Agent 概念：Agent 工具注册 ===

Agent 的"工具"是 LLM 可以调用的外部函数。本文件定义了一个
DuckDuckGo 网络搜索工具，但当前【未启用】——在 flow.go 中被注释掉了。

核心原理：
  - 工具需要实现 Eino 的 tool.BaseTool 接口
  - 工具的 Schema 描述会被发送给 LLM，让 LLM 了解工具的用途和参数
  - LLM 自主决定是否调用工具

本文件的角色：
  提供 DuckDuckGo 搜索工具的定义。当前作为未启用的工具示例存在。
  可参考此文件了解如何定义新的 Agent 工具。

状态：⚠️ 未启用（dead code）
  flow.go 中的调用已被注释掉：
    // searchTool, err := newSearchTool(ctx)

关联文件：
  - flow.go — 工具注册入口（搜索工具的调用已被注释）
  - internal/ai/tools/ — 已启用的工具实现
*/
package chat_pipeline

import (
	"context"

	"github.com/cloudwego/eino-ext/components/tool/duckduckgo/v2"
	"github.com/cloudwego/eino/components/tool"
)

// newSearchTool 创建 DuckDuckGo 文本搜索工具
//
// 【AI 概念】工具定义方式
// DuckDuckGo 工具来自 Eino 的扩展库（eino-ext），
// 它封装了搜索 API，LLM 可以通过调用此工具获取网络搜索结果。
//
// 当前未启用。如果需要启用，在 flow.go 中取消注释相关代码即可。
func newSearchTool(ctx context.Context) (bt tool.BaseTool, err error) {
	config := &duckduckgo.Config{}
	bt, err = duckduckgo.NewTextSearchTool(ctx, config)
	if err != nil {
		return nil, err
	}
	return bt, nil
}
