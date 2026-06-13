/*
=== AI Agent 概念：Agent 编排（Graph / DAG 流水线）===

用有向无环图（DAG）编排多个处理步骤，是 Agent 框架的核心编排模式。
每个节点是一个独立的处理单元（检索、模板、推理等），节点之间通过边定义数据流向。

核心原理：
  - Eino 框架用 compose.NewGraph 创建图
  - 用 AddXxxNode 添加节点（Lambda、Retriever、ChatTemplate 等）
  - 用 AddEdge 定义节点间的数据流向
  - 用 Compile 编译为可执行的 Runnable

本文件的角色：
  定义 RAG Chat 的完整 DAG 图拓扑。这是整个 Pipeline 的入口，
  调用 GetChatAgent 即可得到一个可运行的 Agent（构建一次、请求间复用）。

关键数据流（两路并行，在 ChatTemplate 汇聚）：

  START ──→ InputToRag ──→ MilvusRetriever ──┐
    │                                          ├─→ ChatTemplate ──→ ReActAgent ──→ END
    └────→ InputToChat ───────────────────────┘

  - RAG 路径：提取问题文本 → 向量检索相关文档
  - Chat 路径：提取对话上下文（问题 + 历史 + 时间）
  - 两条路径在 ChatTemplate 汇聚（AllPredecessor 模式），组装最终 Prompt

关联文件：
  - lambda_func.go — InputToRag 和 InputToChat 的 Lambda 实现
  - retriever.go — MilvusRetriever 节点的实现
  - prompt.go — ChatTemplate 节点的实现
  - flow.go — ReActAgent 节点的实现
  - types.go — 输入类型 UserMessage 的定义
*/
package chat_pipeline

import (
	"context"
	"sync"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

// chatAgent 缓存编译好的 Chat Agent，进程内只构建一次、所有请求共享复用。
//
// 【性能优化】为什么要复用：
// Agent 构建过程会建立 3 个 MCP 服务器连接（握手 + 列工具）、Milvus 客户端、
// LLM 模型实例等。这些资源在运行期基本不变，若每个请求都重建，每条聊天消息
// 都要重复 3 次 MCP 握手 + Milvus 连接，开销巨大且毫无必要。
//
// 【并发安全】Agent（编译后的 compose.Runnable）本身是无状态的——
// 对话历史通过 UserMessage.History 每次请求单独传入，因此同一个 Runnable
// 可以被多个请求/会话并发 Invoke。这里用 sync.Mutex + 空值检查实现
// “构建一次、失败可重试”的单例：成功后缓存，失败则下次请求重新构建。
var (
	chatAgentMu sync.Mutex
	chatAgent   compose.Runnable[*UserMessage, *schema.Message]
)

// GetChatAgent 获取（必要时首次构建）复用的 Chat Agent。
//
// 首次调用时构建一次 DAG 图并编译；之后所有请求直接返回缓存的 Runnable。
// 传入的 ctx 仅用于首次构建；后续请求的 Invoke 仍使用各自的请求 context。
func GetChatAgent(ctx context.Context) (compose.Runnable[*UserMessage, *schema.Message], error) {
	chatAgentMu.Lock()
	defer chatAgentMu.Unlock()
	if chatAgent != nil {
		return chatAgent, nil
	}
	r, err := buildChatAgent(ctx)
	if err != nil {
		return nil, err
	}
	chatAgent = r
	return chatAgent, nil
}

// buildChatAgent 构建 RAG Chat Agent 的 DAG 图并编译为可执行对象
//
// 【AI 概念】图编排（Graph Orchestration）
// 将复杂任务拆分为多个独立节点，用 DAG 定义执行顺序和数据流。
// 好处：节点可独立开发和测试，数据流清晰可见。
//
// 返回值：compose.Runnable[*UserMessage, *schema.Message]
//   - 输入：UserMessage（用户问题 + 对话历史）
//   - 输出：*schema.Message（Agent 的回答）
func buildChatAgent(ctx context.Context) (r compose.Runnable[*UserMessage, *schema.Message], err error) {
	// 定义 5 个图节点的名称常量
	const (
		InputToRag      = "InputToRag"      // 数据转换节点：提取问题文本用于 RAG 检索
		ChatTemplate    = "ChatTemplate"    // Prompt 模板节点：组装系统 Prompt + RAG 文档 + 对话历史
		ReactAgent      = "ReactAgent"      // ReAct Agent 节点：LLM 推理 + 工具调用
		MilvusRetriever = "MilvusRetriever" // 向量检索节点：从 Milvus 检索相关文档
		InputToChat     = "InputToChat"     // 数据转换节点：提取对话上下文
	)

	// 创建一个新的 DAG 图
	// 泛型参数：输入类型 *UserMessage，输出类型 *schema.Message
	g := compose.NewGraph[*UserMessage, *schema.Message]()

	// === 添加节点 ===

	// 节点1：InputToRag — Lambda 节点，从 UserMessage 提取问题文本
	// InvokableLambdaWithOption 将普通函数包装为图可用的 Lambda 节点
	_ = g.AddLambdaNode(InputToRag, compose.InvokableLambdaWithOption(newInputToRagLambda), compose.WithNodeName("UserMessageToRag"))

	// 节点2：ChatTemplate — Prompt 模板节点，将 RAG 文档和对话上下文组装为最终 Prompt
	chatTemplateKeyOfChatTemplate, err := newChatTemplate(ctx)
	if err != nil {
		return nil, err
	}
	_ = g.AddChatTemplateNode(ChatTemplate, chatTemplateKeyOfChatTemplate)

	// 节点3：ReActAgent — 核心推理节点，LLM + 工具调用的 ReAct 循环
	reactAgentKeyOfLambda, err := newReactAgentLambda(ctx)
	if err != nil {
		return nil, err
	}
	_ = g.AddLambdaNode(ReactAgent, reactAgentKeyOfLambda, compose.WithNodeName("ReActAgent"))

	// 节点4：MilvusRetriever — 向量检索节点，从 Milvus 知识库检索相关文档
	milvusRetrieverKeyOfRetriever, err := newRetriever(ctx)
	if err != nil {
		return nil, err
	}
	// WithOutputKey("documents") 将检索结果标记为 "documents" 键
	// 这样 ChatTemplate 中的 {documents} 占位符就能匹配到检索结果
	_ = g.AddRetrieverNode(MilvusRetriever, milvusRetrieverKeyOfRetriever, compose.WithOutputKey("documents"))

	// 节点5：InputToChat — Lambda 节点，从 UserMessage 提取对话上下文（问题+历史+时间）
	_ = g.AddLambdaNode(InputToChat, compose.InvokableLambdaWithOption(newInputToChatLambda), compose.WithNodeName("UserMessageToChat"))

	// === 定义边（数据流向）===
	//
	// 图结构：
	//   START ──→ InputToRag ──→ MilvusRetriever ──┐
	//     │                                          ├─→ ChatTemplate ──→ ReActAgent ──→ END
	//     └────→ InputToChat ───────────────────────┘

	// START 节点同时触发两条分支（并行执行）
	_ = g.AddEdge(compose.START, InputToRag)  // 分支1：RAG 检索路径
	_ = g.AddEdge(compose.START, InputToChat)  // 分支2：对话上下文路径

	// ReActAgent 是最终节点，输出到 END
	_ = g.AddEdge(ReactAgent, compose.END)

	// RAG 路径：问题文本 → 向量检索
	_ = g.AddEdge(InputToRag, MilvusRetriever)

	// 两条路径汇聚到 ChatTemplate
	_ = g.AddEdge(MilvusRetriever, ChatTemplate) // RAG 检索结果 → Prompt 模板
	_ = g.AddEdge(InputToChat, ChatTemplate)      // 对话上下文 → Prompt 模板

	// Prompt 模板 → ReAct Agent 推理
	_ = g.AddEdge(ChatTemplate, ReactAgent)

	// === 编译图 ===
	// WithNodeTriggerMode(compose.AllPredecessor)：
	//   节点在【所有前驱节点】都完成后才触发。
	//   这对 ChatTemplate 很关键——它必须等 RAG 检索和对话上下文都就绪才能组装 Prompt。
	r, err = g.Compile(ctx, compose.WithGraphName("ChatAgent"), compose.WithNodeTriggerMode(compose.AllPredecessor))
	if err != nil {
		return nil, err
	}
	return r, err
}
