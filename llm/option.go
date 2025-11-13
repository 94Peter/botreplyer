package llm

type ConversationMgrOption func(mgr *conversationMgr)

func WithConversationMemoryMongo(baseUrl, dbName, collectionName string) ConversationMgrOption {
	return func(mgr *conversationMgr) {
		mgr.mongoMemory = &MemoryMongoCfg{
			ConnectionURL:  baseUrl,
			DatabaseName:   dbName,
			CollectionName: collectionName,
		}
	}
}
