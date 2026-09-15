package simple

import (
	"log"

	"github.com/David3310273/go-agent/core"
	knowledgebase "github.com/David3310273/go-agent/knowledgebase"
	storage "github.com/David3310273/go-agent/knowledgebase/storage"
	"github.com/David3310273/go-agent/storage/milvus"
)

// NewSimpleKnowledgeBase creates a MilvusKnowledgebase from config.
// factory function that loads embedder and chunker from knowledgebase config file.
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
		// load knowledgebase config and create embedder and chunker
		kbConfig, err := knowledgebase.LoadConfig(rootPath)
		if err != nil {
			log.Printf("[KnowledgeBase] failed to load config: %v, embedder and chunker will be nil", err)
		}

		var embedder core.Embedder
		var chunker core.Chunker
		if kbConfig != nil {
			// get storage config by domain
			storageConfig := kbConfig.GetStorageConfigByDomain(config.Domain)
			if storageConfig != nil {
				embedder, err = knowledgebase.CreateEmbedder(&storageConfig.Embedder)
				if err != nil {
					log.Printf("[KnowledgeBase] failed to create embedder: %v", err)
				}

				chunker, err = knowledgebase.CreateChunker(&storageConfig.Chunker)
				if err != nil {
					log.Printf("[KnowledgeBase] failed to create chunker: %v", err)
				}
			}
		}

		return storage.NewMilvusKnowledgebase[any](embedder, chunker, config, opts)
	}

	return nil
}
