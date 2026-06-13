package main

import (
	"context"
	"fmt"
	"log"

	"OpsPilot/internal/ai/embedder"
	"OpsPilot/utility/client"
	"OpsPilot/utility/common"
	"OpsPilot/utility/config"

	"github.com/cloudwego/eino-ext/components/indexer/milvus"
	"github.com/milvus-io/milvus-sdk-go/v2/entity"
)

func main() {
	config.Init("manifest/config/config.yaml")
	ctx := context.Background()
	cli, err := client.NewMilvusClient(ctx)
	if err != nil {
		log.Fatal(err)
	}

	cli.DropCollection(ctx, "biz")

	eb, err := embedder.NewEmbedder(ctx)
	if err != nil {
		log.Fatal(err)
	}

	fields := []*entity.Field{
		{Name: "id", DataType: entity.FieldTypeVarChar, TypeParams: map[string]string{"max_length": "255"}, PrimaryKey: true},
		{Name: "vector", DataType: entity.FieldTypeFloatVector, TypeParams: map[string]string{"dim": "3072"}},
		{Name: "content", DataType: entity.FieldTypeVarChar, TypeParams: map[string]string{"max_length": "8192"}},
		{Name: "metadata", DataType: entity.FieldTypeJSON},
	}

	_, err = milvus.NewIndexer(ctx, &milvus.IndexerConfig{
		Client:     cli,
		Collection: common.MilvusCollectionName,
		Fields:     fields,
		Embedding:  eb,
	})
	if err != nil {
		coll, descErr := cli.DescribeCollection(ctx, "biz")
		if descErr == nil {
			fmt.Printf("Milvus fields (%d):\n", len(coll.Schema.Fields))
			for i, f := range coll.Schema.Fields {
				fmt.Printf("  [%d] Name=%-10s DataType=%d\n", i, f.Name, f.DataType)
			}
			fmt.Printf("Defined fields (%d):\n", len(fields))
			for i, f := range fields {
				fmt.Printf("  [%d] Name=%-10s DataType=%d\n", i, f.Name, f.DataType)
			}
		}
		log.Fatalf("Error: %v", err)
	}
	fmt.Println("Success!")
}
