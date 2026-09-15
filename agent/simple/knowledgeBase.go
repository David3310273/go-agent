package simple

import (
	"log"

	"github.com/David3310273/go-agent/core"
	knowledgebase "github.com/David3310273/go-agent/knowledgebase"
	storage "github.com/David3310273/go-agent/knowledgebase/storage"
	"github.com/David3310273/go-agent/storage/milvus"
)

// NewSimpleKnowledgeBase creates a MilvusKnowledgebase from config.
// [auto-added] factory function that loads embedder from knowledgebase config file.
// rootPath should be set in config.RootPath before calling this function.
func NewSimpleKnowledgeBase(config core.KnowledgeBaseConfig) *storage.MilvusKnowledgebase[any] {
	// inject root path from config
	rootPath := config.RootPath
	opts := milvus.MilvusOptions{
		DBName:        config.StorageOptions["dbName"],
		Collection:    config.StorageOptions["collection"],
		PartitionName: config.StorageOptions["partitionName"],
	}

	// factory logic to create kb using config
	if config.StorageOptions["type"] == storage.MilvusKnowledgeBaseStorageType {
		// load knowledgebase config and create embedder
		kbConfig, err := knowledgebase.LoadConfig(rootPath)
		if err != nil {
			log.Printf("[KnowledgeBase] failed to load config: %v, embedder will be nil", err)
		}

		var embedder core.Embedder
		if kbConfig != nil {
			embedder, err = knowledgebase.CreateEmbedder(&kbConfig.Milvus.Embedder)
			if err != nil {
				log.Printf("[KnowledgeBase] failed to create embedder: %v", err)
			}
		}

		return storage.NewMilvusKnowledgebase[any](embedder, config, opts)
	}

	return nil
}
