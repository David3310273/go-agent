package core

type Storage interface {
	Save(data any) *Diagnostic
	Search(data any) (any, *Diagnostic)
	Insert(data any) *Diagnostic
	Delete(data any) *Diagnostic
}
