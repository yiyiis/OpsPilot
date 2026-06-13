package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
	"github.com/subosito/gotenv"
)

// C 是全局 Viper 实例，在 main.go 中通过 Init() 初始化
var C *viper.Viper

// Init 初始化配置：
//  1. 加载 .env 文件到环境变量（必须存在）
//  2. 读取 config.yaml
//  3. 将 ${ENV_VAR} 占位符替换为实际环境变量值
func Init(configPath string) {
	// 加载 .env（必须存在）
	if err := gotenv.Load(); err != nil {
		panic(fmt.Errorf(".env 文件加载失败，请确保项目根目录存在 .env 文件: %w", err))
	}

	C = viper.New()
	C.SetConfigFile(configPath)

	if err := C.ReadInConfig(); err != nil {
		panic(fmt.Errorf("读取配置文件失败: %w", err))
	}

	// 遍历所有配置项，将 ${ENV_VAR} 替换为环境变量值
	interpolateEnv(C)
}

// interpolateEnv 将 Viper 中所有值的 ${ENV_VAR} 占位符替换为环境变量
func interpolateEnv(v *viper.Viper) {
	for _, key := range v.AllKeys() {
		if val := v.GetString(key); val != "" {
			v.Set(key, expandEnv(val))
		}
	}
}

// expandEnv 替换字符串中的 ${VAR} 为环境变量值
func expandEnv(s string) string {
	return os.Expand(s, func(key string) string {
		return os.Getenv(key)
	})
}

// GetString 读取字符串配置
func GetString(key string) string {
	return expandEnv(C.GetString(key))
}

// GetInt 读取整数配置
func GetInt(key string) int {
	return C.GetInt(key)
}

// UnmarshalKey 将配置子树解码到结构体
// 用于读取复杂的嵌套配置（如 mcp_servers 列表）
func UnmarshalKey(key string, rawVal any) error {
	return C.UnmarshalKey(key, rawVal)
}
