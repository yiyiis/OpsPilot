package chat

// 请求/响应结构体（原 GoFrame api/chat/v1/chat.go 中的定义，去除 g.Meta 标签）

type ChatReq struct {
	Id       string `json:"Id"`
	Question string `json:"Question"`
}

type ChatRes struct {
	Answer string `json:"answer"`
}

type ChatStreamReq struct {
	Id       string `json:"Id"`
	Question string `json:"Question"`
}

type FileUploadRes struct {
	FileName string `json:"fileName"`
	FilePath string `json:"filePath"`
	FileSize int64  `json:"fileSize"`
}

type AIOpsReq struct {
	Query string `json:"query"`
}

type AIOpsRes struct {
	Result string   `json:"result"`
	Detail []string `json:"detail"`
}
