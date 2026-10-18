package fault

import "errors"

var (
	ErrInvalid   = errors.New("dados inválidos")
	ErrNotFound  = errors.New("registro não encontrado")
	ErrConflict  = errors.New("registro em conflito")
	ErrForbidden = errors.New("sem permissão")
)

// Coded carries a stable client-facing code without exposing database errors.
type Coded struct {
	Kind          error
	Code, Message string
}

func (e *Coded) Error() string { return e.Message }
func (e *Coded) Unwrap() error { return e.Kind }
