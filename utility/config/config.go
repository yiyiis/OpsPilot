package config

import (
	"fmt"
	"os"
	"regexp"

	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/viper"
	"github.com/subosito/gotenv"
)

// App 是全局配置实例，在 main.go 中通过 Init() 初始化并填充。
// 业务代码统一通过 config.App.Xxx.Yyy 访问，避免散落的字符串 key
// （config.GetString）带来的拼写错误与类型不安全。
var App *AppConfig

// AppConfig 对应 manifest/config/config.yaml 的整体结构。
// Init 时通过 mapstructure 一次性解码。
type AppConfig struct {
	Server          ServerConfig      `mapstructure:"server"`
	Milvus          MilvusConfig      `mapstructure:"milvus"`
	Prometheus      PrometheusConfig  `mapstructure:"prometheus"`
	Logger          LoggerConfig      `mapstructure:"logger"`
	GLMThinkChat    ModelConfig       `mapstructure:"glm_think_chat_model"`
	GLMQuickChat    ModelConfig       `mapstructure:"glm_quick_chat_model"`
	GeminiEmbedding ModelConfig       `mapstructure:"gemini_embedding_model"`
	FileDir         string            `mapstructure:"file_dir"`
	MCPServers      []MCPServerConfig `mapstructure:"mcp_servers"`
}

// ServerConfig HTTP 服务配置
type ServerConfig struct {
	Address     string `mapstructure:"address"`
	OpenapiPath string `mapstructure:"openapiPath"`
	SwaggerPath string `mapstructure:"swaggerPath"`
}

// MilvusConfig Milvus 向量数据库连接配置
type MilvusConfig struct {
	Address string `mapstructure:"address"`
}

// PrometheusConfig Prometheus 监控连接配置
type PrometheusConfig struct {
	BaseURL string `mapstructure:"base_url"`
}

// LoggerConfig 日志配置
type LoggerConfig struct {
	Level  string `mapstructure:"level"`
	Stdout bool   `mapstructure:"stdout"`
}

// ModelConfig 通用模型配置。
// GLM Think / GLM Quick / Gemini Embedding 三处模型配置共用此结构。
type ModelConfig struct {
	Model   string `mapstructure:"model"`
	APIKey  string `mapstructure:"api_key"`
	BaseURL string `mapstructure:"base_url"`
}

// MCPServerConfig 对应 config.yaml 中 mcp_servers 下的单个服务器配置。
// connection_headers 中的 ${VAR} 已在 Init 时替换为环境变量真实值。
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

// Init 初始化配置：
//  1. 加载 .env 文件到环境变量
//  2. 读取 config.yaml
//  3. 递归替换所有值（含嵌套 map/list）中的 ${ENV_VAR} 占位符
//  4. 通过 mapstructure 一次性解码到全局 App
func Init(configPath string) {
	// 加载 .env（必须存在）
	if err := gotenv.Load(); err != nil {
		panic(fmt.Errorf(".env 文件加载失败，请确保项目根目录存在 .env 文件: %w", err))
	}

	v := viper.New()
	v.SetConfigFile(configPath)
	if err := v.ReadInConfig(); err != nil {
		panic(fmt.Errorf("读取配置文件失败: %w", err))
	}

	// 递归替换 ${VAR}，得到已展开的纯 map。
	// 刻意不使用 viper.Unmarshal，而是对替换后的 map 直接 mapstructure.Decode，
	// 避免 viper Set/Unmarshal 内部时序对占位符替换的影响。
	resolved := expandAny(v.AllSettings()).(map[string]any)

	App = &AppConfig{}
	if err := mapstructure.Decode(resolved, App); err != nil {
		panic(fmt.Errorf("配置解码失败: %w", err))
	}
}

// envRefRegex 仅匹配 ${VAR} 形式的占位符（大写字母开头，可含数字和下划线）。
// 刻意不匹配 $VAR 这种裸形式，避免配置值中的 $（如密码 "p@ss$word"）被误当作变量展开。
var envRefRegex = regexp.MustCompile(`\$\{([A-Z][A-Z0-9_]*)\}`)

// expandEnv 替换字符串中的 ${VAR} 为环境变量值
func expandEnv(s string) string {
	return envRefRegex.ReplaceAllStringFunc(s, func(match string) string {
		envKey := match[2 : len(match)-1] // 去掉 ${ 和 }
		return os.Getenv(envKey)
	})
}

// expandAny 递归展开配置值中的 ${VAR}：
//   - string：直接展开
//   - map[string]any：递归处理每个 value
//   - []any：递归处理每个元素
//   - 其它类型（int/bool 等）：原样返回
func expandAny(v any) any {
	switch val := v.(type) {
	case string:
		return expandEnv(val)
	case map[string]any:
		out := make(map[string]any, len(val))
		for k, vv := range val {
			out[k] = expandAny(vv)
		}
		return out
	case []any:
		out := make([]any, len(val))
		for i, item := range val {
			out[i] = expandAny(item)
		}
		return out
	default:
		return v
	}
}
