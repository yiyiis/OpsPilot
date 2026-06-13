/*
=== AI Agent 概念：文档加载（File Loading）===

文档加载是知识索引 Pipeline 的第一步。将磁盘上的文件（.md、.txt）
读取并转换为 Eino 框架的 document.Source 对象，供下游分块处理。

核心原理：
  - FileLoader 读取文件原始内容
  - 输出为 Eino 的 document.Loader 接口类型
  - 下游节点（MarkdownSplitter）接收加载后的文档

本文件的角色：
  创建 FileLoader 实例，作为 DAG 图中的第一个节点。

关联文件：
  - orchestration.go — 将本加载器加入 DAG 图
  - transformer.go — 下游节点，接收加载后的文档进行分块
*/
package knowledge_index_pipeline

import (
	"context"

	"github.com/cloudwego/eino-ext/components/document/loader/file"
	"github.com/cloudwego/eino/components/document"
)

// newLoader 创建文件加载器
//
// 【AI 概念】文档加载
// FileLoader 使用 Eino 扩展库（eino-ext）提供的文件加载组件。
// 默认配置（FileLoaderConfig{}）支持常见文件格式。
//
// 输入：document.Source（包含文件路径等信息）
// 输出：document.Loader 接口（可被图节点使用）
func newLoader(ctx context.Context) (ldr document.Loader, err error) {
	// 使用默认配置创建文件加载器
	// TODO: 可以在这里自定义配置（如支持的文件类型、编码等）
	config := &file.FileLoaderConfig{}
	ldr, err = file.NewFileLoader(ctx, config)
	if err != nil {
		return nil, err
	}
	return ldr, nil
}
