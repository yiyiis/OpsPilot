/*
=== AI Agent 概念：Milvus 客户端初始化 ===

Milvus 是本项目使用的向量数据库，用于存储和检索知识库的向量数据。
本文件负责初始化 Milvus 客户端，包括自动创建数据库和集合。

核心原理：
  - Milvus 的数据组织：数据库（Database）→ 集合（Collection）→ 字段（Field）
  - 本项目使用 "agent" 数据库、"biz" 集合
  - 集合 Schema 包含 4 个字段：id、vector、content、metadata
  - 自动初始化：首次连接时自动创建不存在的数据库和集合

本文件的角色：
  提供 Milvus 客户端实例。所有需要访问 Milvus 的地方
  （检索器、索引器）都通过此文件获取客户端连接。

初始化流程：
  1. 连接 default 数据库
  2. 检查 "agent" 数据库是否存在，不存在则创建
  3. 连接 "agent" 数据库
  4. 检查 "biz" 集合是否存在，不存在则创建（含 Schema 和索引）
  5. 关闭 default 连接，返回 agent 数据库的客户端

关联文件：
  - utility/common/common.go — 数据库名和集合名的常量定义
  - internal/ai/retriever/retriever.go — 检索器使用客户端
  - internal/ai/indexer/indexer.go — 索引器使用客户端
*/
package client

import (
	"OpsPilot/utility/common"
	"OpsPilot/utility/config"
	"context"
	"fmt"

	cli "github.com/milvus-io/milvus-sdk-go/v2/client"
	"github.com/milvus-io/milvus-sdk-go/v2/entity"
)

// NewMilvusClient 创建并初始化 Milvus 客户端
//
// 【AI 概念】向量数据库的自动初始化
// 与传统数据库不同，向量数据库需要预先定义向量维度和索引类型。
// 本函数在首次运行时自动完成所有初始化工作，确保数据库和集合已就绪。
//
// 返回值：连接到 "agent" 数据库的 Milvus 客户端
func NewMilvusClient(ctx context.Context) (cli.Client, error) {
	// 步骤1：连接 default 数据库（用于管理数据库）
	defaultClient, err := cli.NewClient(ctx, cli.Config{
		Address: config.App.Milvus.Address, // 从 config.yaml 读取
		DBName:  "default",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to default database: %w", err)
	}

	// 步骤2：检查 "agent" 数据库是否存在，不存在则创建
	databases, err := defaultClient.ListDatabases(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list databases: %w", err)
	}
	agentDBExists := false
	for _, db := range databases {
		if db.Name == common.MilvusDBName {
			agentDBExists = true
			break
		}
	}
	if !agentDBExists {
		err = defaultClient.CreateDatabase(ctx, common.MilvusDBName)
		if err != nil {
			return nil, fmt.Errorf("failed to create agent database: %w", err)
		}
	}

	// 步骤3：创建连接到 "agent" 数据库的客户端
	agentClient, err := cli.NewClient(ctx, cli.Config{
		Address: config.App.Milvus.Address,
		DBName:  common.MilvusDBName, // "agent"
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to agent database: %w", err)
	}

	// 步骤4：检查 "biz" 集合是否存在，不存在则创建
	collections, err := agentClient.ListCollections(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}

	bizCollectionExists := false
	for _, collection := range collections {
		if collection.Name == common.MilvusCollectionName {
			bizCollectionExists = true
			break
		}
	}

	if !bizCollectionExists {
		// 创建集合 Schema
		schema := &entity.Schema{
			CollectionName: common.MilvusCollectionName, // "biz"
			Description:    "Business knowledge collection",
			Fields:         fields, // 见下方的 fields 定义
		}

		err = agentClient.CreateCollection(ctx, schema, entity.DefaultShardNumber)
		if err != nil {
			return nil, fmt.Errorf("failed to create biz collection: %w", err)
		}

		// 为各字段创建索引，加速检索
		// id 字段索引（L2 距离）
		idIndex, err := entity.NewIndexAUTOINDEX(entity.L2)
		if err != nil {
			return nil, fmt.Errorf("failed to create id index: %w", err)
		}
		err = agentClient.CreateIndex(ctx, common.MilvusCollectionName, "id", idIndex, false)
		if err != nil {
			return nil, fmt.Errorf("failed to create id index: %w", err)
		}

		// content 字段索引（L2 距离）
		contentIndex, err := entity.NewIndexAUTOINDEX(entity.L2)
		if err != nil {
			return nil, fmt.Errorf("failed to create content index: %w", err)
		}
		err = agentClient.CreateIndex(ctx, common.MilvusCollectionName, "content", contentIndex, false)
		if err != nil {
			return nil, fmt.Errorf("failed to create content index: %w", err)
		}

		// vector 字段索引（COSINE 余弦相似度，适用于浮点向量）
		vectorIndex, err := entity.NewIndexAUTOINDEX(entity.COSINE)
		if err != nil {
			return nil, fmt.Errorf("failed to create vector index: %w", err)
		}
		err = agentClient.CreateIndex(ctx, common.MilvusCollectionName, "vector", vectorIndex, false)
		if err != nil {
			return nil, fmt.Errorf("failed to create vector index: %w", err)
		}
	}

	// 关闭 default 数据库连接（不再需要）
	defaultClient.Close()

	return agentClient, nil
}

// fields 定义 Milvus 集合的字段 Schema
//
// 【AI 概念】向量数据库的 Schema 设计
//
// 集合名："biz"（业务知识库）
//
// 4 个字段：
//   - id: VarChar(256), 主键 — 文档片段的唯一标识（UUID）
//   - vector: FloatVector(3072) — 文档片段的向量表示（float32 格式）
//   - content: VarChar(8192) — 文档片段的原始文本内容
//   - metadata: JSON — 文档片段的元数据（如来源、标题等）
//
// 为什么用 FloatVector？
// 本项目使用 Gemini Embedding 模型生成 3072 维的 float32 向量。
// FloatVector 保留完整的浮点精度，配合 COSINE 距离度量，语义检索效果最佳。
var fields = []*entity.Field{
	{
		Name:     "id",
		DataType: entity.FieldTypeVarChar,
		TypeParams: map[string]string{
			"max_length": "256",
		},
		PrimaryKey: true, // 主键字段
	},
	{
		Name:     "vector",
		DataType: entity.FieldTypeFloatVector, // 浮点向量类型
		TypeParams: map[string]string{
			"dim": "3072", // 向量维度：3072（Gemini Embedding 输出维度）
		},
	},
	{
		Name:     "content",
		DataType: entity.FieldTypeVarChar,
		TypeParams: map[string]string{
			"max_length": "65535",
		},
	},
	{
		Name:     "metadata",
		DataType: entity.FieldTypeJSON,
	},
}
