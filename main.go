package main

import (
	"OpsPilot/internal/controller/chat"
	"OpsPilot/utility/common"
	"OpsPilot/utility/config"
	"OpsPilot/utility/middleware"

	_ "OpsPilot/docs/swagger"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title           OpsPilot API
// @version         1.0
// @description     AI 驱动的智能运维助手 API
// @host            localhost:6872
// @BasePath        /api
func main() {
	// 初始化配置
	config.Init("manifest/config/config.yaml")
	common.FileDir = config.GetString("file_dir")

	// 创建 Gin 路由
	r := gin.Default()

	// 全局中间件
	r.Use(middleware.CORSMiddleware())

	// Swagger 文档
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// API 路由组
	handler := chat.NewChatHandler()
	api := r.Group("/api")
	{
		api.POST("/chat", handler.Chat)
		api.POST("/chat_stream", handler.ChatStream)
		api.POST("/upload", handler.FileUpload)
		api.POST("/ai_ops", handler.AIOps)
	}

	// 启动服务
	r.Run(":6872")
}
