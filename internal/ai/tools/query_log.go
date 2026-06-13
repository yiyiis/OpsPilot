/*
=== AI Agent 概念：多 MCP 服务器工具集成 ===

MCP（Model Context Protocol）是一种标准化的工具集成协议。
通过 MCP，Agent 可以连接到多个外部工具服务器，动态获取可用工具列表，
而无需在代码中硬编码每个工具的实现。

核心原理：
  - 支持三种传输方式：stdio、SSE、Streamable HTTP（符合 Anthropic MCP 规范）
  - 从配置文件读取多个 MCP 服务器定义
  - 逐一连接每个服务器，获取工具列表并汇总
  - 单个服务器失败不影响其他服务器

本文件的角色：
  通过 MCP 协议连接多个外部服务：
    1. 本地日志文件查询服务（SSE）
    2. 智谱联网搜索服务（Streamable HTTP）
    3. 智谱网页读取服务（Streamable HTTP）

关键数据流：
  1. 读取 config.yaml 中的 mcp_servers 配置列表
  2. 对每个服务器：创建客户端 → 连接 → 初始化 → 获取工具
  3. 汇总所有工具为 []tool.BaseTool 返回

关联文件：
  - manifest/config/config.yaml — mcp_servers 配置
  - internal/ai/mcp/log_reader_server/ — 本地日志 MCP Server
  - flow.go — 将 MCP 工具注册到 ReAct Agent
  - executor.go — 将 MCP 工具注册到 Plan-Execute-Replan 执行器
*/
package tools

import (
	"context"
	"log"
	"os"
	"regexp"

	"OpsPilot/utility/config"

	e_mcp "github.com/cloudwego/eino-ext/components/tool/mcp"
	"github.com/cloudwego/eino/components/tool"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
)

// MCPServerConfig 对应 config.yaml 中 mcp_servers 下的单个服务器配置
type MCPServerConfig struct {
	Name              string            `mapstructure:"name"`
	Description       string            `mapstructure:"description"`
	ConnectionURL     string            `mapstructure:"connection_url"`
	ConnectionType    string            `mapstructure:"connection_type"`    // "stdio"、"sse" 或 "streamable_http"
	Command           string            `mapstructure:"command"`            // stdio 模式下的可执行文件路径
	Args              []string          `mapstructure:"args"`               // stdio 模式下的命令行参数
	Env               []string          `mapstructure:"env"`                // stdio 模式下的额外环境变量
	ConnectionHeaders map[string]string `mapstructure:"connection_headers"` // SSE / Streamable HTTP 自定义请求头
	Enabled           bool              `mapstructure:"enabled"`
}

// GetAllMcpTools 连接所有配置的 MCP 服务器，汇总返回工具列表
//
// 【AI 概念】多 MCP 服务器集成
//
// 支持同时连接多个 MCP 服务器，每个服务器可能提供多个工具。
// 例如：日志查询服务器提供 query_log，智谱搜索服务提供 web_search 等。
//
// 工作流程：
//  1. 从配置读取 mcp_servers 列表
//  2. 遍历每个启用的服务器
//  3. 根据连接类型（stdio / sse / streamable_http）创建客户端
//  4. 连接 → 初始化 → 获取工具
//  5. 汇总所有工具返回
//
// 容错机制：单个服务器连接失败只记录警告，不影响其他服务器
func GetAllMcpTools() ([]tool.BaseTool, error) {
	// 读取 MCP 服务器配置列表
	var servers []MCPServerConfig
	if err := config.UnmarshalKey("mcp_servers", &servers); err != nil {
		log.Printf("[MCP] 读取 mcp_servers 配置失败: %v", err)
		return []tool.BaseTool{}, err
	}

	ctx := context.Background()
	var allTools []tool.BaseTool

	for _, srv := range servers {
		if !srv.Enabled {
			log.Printf("[MCP] 跳过已禁用的服务器: %s", srv.Name)
			continue
		}

		log.Printf("[MCP] 正在连接服务器: %s (%s)", srv.Name, srv.ConnectionType)

		// 创建 MCP 客户端（stdio 模式下已自动启动，无需后续 Start）
		cli, err := createMcpClient(srv)
		if err != nil {
			log.Printf("[MCP] 创建客户端失败 [%s]: %v", srv.Name, err)
			continue
		}

		// stdio 模式在 NewStdioMCPClient 中已自动 Start，跳过手动启动
		if srv.ConnectionType != "stdio" {
			if err = cli.Start(ctx); err != nil {
				log.Printf("[MCP] 连接失败 [%s]: %v", srv.Name, err)
				continue
			}
		}

		// 发送初始化请求
		initRequest := mcp.InitializeRequest{}
		initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
		initRequest.Params.ClientInfo = mcp.Implementation{
			Name:    "OpsPilot-client",
			Version: "1.0.0",
		}
		if _, err = cli.Initialize(ctx, initRequest); err != nil {
			log.Printf("[MCP] 初始化失败 [%s]: %v", srv.Name, err)
			continue
		}

		// 获取工具列表
		mcpTools, err := e_mcp.GetTools(ctx, &e_mcp.Config{Cli: cli})
		if err != nil {
			log.Printf("[MCP] 获取工具失败 [%s]: %v", srv.Name, err)
			continue
		}

		log.Printf("[MCP] 连接成功 [%s]，获取到 %d 个工具", srv.Name, len(mcpTools))
		allTools = append(allTools, mcpTools...)
	}

	if len(allTools) == 0 {
		log.Printf("[MCP] 警告：未从任何 MCP 服务器获取到工具")
	}

	return allTools, nil
}

// createMcpClient 根据配置创建对应类型的 MCP 客户端
func createMcpClient(srv MCPServerConfig) (*client.Client, error) {
	switch srv.ConnectionType {
	case "stdio":
		// stdio 模式：启动子进程，通过 stdin/stdout 通信
		// 环境变量：继承当前进程 + 额外配置的 env
		env := os.Environ()
		env = append(env, srv.Env...)
		return client.NewStdioMCPClient(srv.Command, env, srv.Args...)

	case "streamable_http":
		headers := resolveHeaders(srv.ConnectionHeaders)
		return client.NewStreamableHttpClient(
			srv.ConnectionURL,
			transport.WithHTTPHeaders(headers),
		)

	case "sse":
		// SSE 也支持自定义请求头（用于认证等）
		opts := []transport.ClientOption{}
		if len(srv.ConnectionHeaders) > 0 {
			opts = append(opts, client.WithHeaders(resolveHeaders(srv.ConnectionHeaders)))
		}
		return client.NewSSEMCPClient(srv.ConnectionURL, opts...)

	default:
		log.Printf("[MCP] 未知的连接类型 [%s]: %s，默认使用 SSE", srv.Name, srv.ConnectionType)
		return client.NewSSEMCPClient(srv.ConnectionURL)
	}
}

// resolveHeaders 精确替换请求头中的 ${ENV_VAR} 为环境变量值
// 仅匹配 ${...} 格式，不会误伤 $HOME 等 shell 变量或 JWT 中的字面量
var envRefRegex = regexp.MustCompile(`\$\{([A-Z][A-Z0-9_]*)\}`)

func resolveHeaders(headers map[string]string) map[string]string {
	resolved := make(map[string]string, len(headers))
	for key, value := range headers {
		resolved[key] = envRefRegex.ReplaceAllStringFunc(value, func(match string) string {
			envKey := match[2 : len(match)-1] // 去掉 ${ 和 }
			envVal := os.Getenv(envKey)
			if envVal == "" {
				log.Printf("[MCP] 警告：环境变量 %s 未设置", envKey)
			}
			return envVal
		})
	}
	return resolved
}
