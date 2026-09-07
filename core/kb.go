package core

// TODO: support local kb in simple agent
type KnowledgeBaseManger interface {
	SetStorage(storage Storage)
	SearchKnowledgebase(keyword string) string
	InsertKnowledgebase(data any) *Diagnostic
	DeleteKnowledgebase(data any) *Diagnostic
	UpdateKnowledgebase(data any) *Diagnostic
}
