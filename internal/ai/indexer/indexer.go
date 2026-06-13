package indexer

import (
	embedder2 "OpsPilot/internal/ai/embedder"
	"OpsPilot/utility/client"
	"OpsPilot/utility/common"
	"context"
	"github.com/bytedance/sonic"
	"github.com/cloudwego/eino-ext/components/indexer/milvus"
	"github.com/cloudwego/eino/schema"
	"github.com/milvus-io/milvus-sdk-go/v2/entity"
)

// floatVectorSchema 是支持 FloatVector 的自定义行结构
// Eino 默认的 defaultSchema 使用 []byte（BinaryVector），
// 这里用 []float32 对应 FloatVector 字段类型
type floatVectorSchema struct {
	ID       string  `json:"id" milvus:"name:id"`
	Content  string  `json:"content" milvus:"name:content"`
	Vector   []float32 `json:"vector" milvus:"name:vector"`
	Metadata []byte  `json:"metadata" milvus:"name:metadata"`
}

func NewMilvusIndexer(ctx context.Context) (*milvus.Indexer, error) {
	cli, err := client.NewMilvusClient(ctx)
	if err != nil {
		return nil, err
	}
	eb, err := embedder2.NewEmbedder(ctx)
	if err != nil {
		return nil, err
	}
	config := &milvus.IndexerConfig{
		Client:            cli,
		Collection:        common.MilvusCollectionName,
		Fields:            fields,
		Embedding:         eb,
		MetricType:        milvus.COSINE, // FloatVector 使用余弦相似度
		DocumentConverter: floatVectorDocumentConverter,
	}
	indexer, err := milvus.NewIndexer(ctx, config)
	if err != nil {
		return nil, err
	}
	return indexer, nil
}

// floatVectorDocumentConverter 将文档和向量转换为 Milvus 可插入的行
// 将 float64 向量转为 float32，匹配 FloatVector 字段类型
func floatVectorDocumentConverter(ctx context.Context, docs []*schema.Document, vectors [][]float64) ([]interface{}, error) {
	rows := make([]interface{}, 0, len(docs))

	for i, doc := range docs {
		metadata, err := sonic.Marshal(doc.MetaData)
		if err != nil {
			return nil, err
		}

		// float64 → float32
		vec := make([]float32, len(vectors[i]))
		for j, v := range vectors[i] {
			vec[j] = float32(v)
		}

		// 如果文档没有 ID，使用 JSON 序列化的元数据中的 id，或留空让 Milvus 生成
		id := doc.ID
		if id == "" {
			if idVal, ok := doc.MetaData["id"]; ok {
				id, _ = idVal.(string)
			}
		}

		rows = append(rows, &floatVectorSchema{
			ID:       id,
			Content:  doc.Content,
			Vector:   vec,
			Metadata: metadata,
		})
	}
	return rows, nil
}

var fields = []*entity.Field{
	{
		Name:     "id",
		DataType: entity.FieldTypeVarChar,
		TypeParams: map[string]string{
			"max_length": "255",
		},
		PrimaryKey: true,
	},
	{
		Name:     "vector", // 确保字段名匹配
		DataType: entity.FieldTypeFloatVector,
		TypeParams: map[string]string{
			"dim": "3072",
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
