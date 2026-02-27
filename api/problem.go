package api

import (
	"encoding/json"
	"net/http"
)

// Content-Type definido no RFC 9457 para Problem Details
const ProblemContentType = "application/problem+json"

// Problem representa o corpo de erro padronizado RFC 9457 (Problem Details for HTTP APIs).
type Problem struct {
	Type     string        `json:"type,omitempty"`     // URI reference que identifica o tipo do problema
	Title    string        `json:"title"`             // Resumo legível do tipo
	Status   int           `json:"status"`             // Código HTTP
	Detail   string        `json:"detail,omitempty"`   // Explicação específica desta ocorrência
	Instance string        `json:"instance,omitempty"` // URI que identifica a ocorrência (ex: path da requisição)
	Errors   []FieldError  `json:"errors,omitempty"`   // Extensão: erros por campo (validação)
}

// FieldError representa um erro em um campo (validação).
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Write envia uma resposta de erro no formato RFC 9457.
// Use instance vazio para preencher com r.URL.Path quando r não for nil.
func Write(w http.ResponseWriter, r *http.Request, status int, title, detail string) {
	instance := ""
	if r != nil {
		instance = r.URL.Path
	}
	p := Problem{
		Type:     "about:blank",
		Title:    title,
		Status:   status,
		Detail:   detail,
		Instance: instance,
	}
	writeProblem(w, status, p)
}

// WriteValidation envia erro 422 Unprocessable Entity com lista de erros por campo.
func WriteValidation(w http.ResponseWriter, r *http.Request, errors []FieldError) {
	instance := ""
	if r != nil {
		instance = r.URL.Path
	}
	p := Problem{
		Type:     "about:blank",
		Title:    "Validation Failed",
		Status:   http.StatusUnprocessableEntity,
		Detail:   "Um ou mais campos possuem valores inválidos.",
		Instance: instance,
		Errors:   errors,
	}
	writeProblem(w, http.StatusUnprocessableEntity, p)
}

const internalErrDetail = "Ocorreu um erro inesperado. Tente novamente mais tarde."

// WriteInternal envia 500 sem expor detalhes internos (para uso em recover/panic).
func WriteInternal(w http.ResponseWriter, r *http.Request) {
	WriteInternalErr(w, r, nil, false)
}

// WriteInternalErr envia 500. Em devMode (ex.: ENV=development), detail usa err.Error() para facilitar depuração; em produção mantém mensagem genérica.
func WriteInternalErr(w http.ResponseWriter, r *http.Request, err error, devMode bool) {
	detail := internalErrDetail
	if devMode && err != nil && err.Error() != "" {
		detail = err.Error()
	}
	Write(w, r, http.StatusInternalServerError, "Internal Server Error", detail)
}

func writeProblem(w http.ResponseWriter, status int, p Problem) {
	w.Header().Set("Content-Type", ProblemContentType+"; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(p)
}

// Recover é um middleware que recupera panic e retorna 500 no formato RFC 9457.
// Deve ser usado no topo da cadeia de middlewares.
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				WriteInternal(w, r)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
