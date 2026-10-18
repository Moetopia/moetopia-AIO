package httpserver

import (
	"net/http"
	"strconv"

	"github.com/danielgtaylor/huma/v2"
)

func init() {
	factory := huma.NewError
	huma.NewError = func(status int, message string, details ...error) huma.StatusError {
		original := factory(status, message, details...)
		model, ok := original.(*huma.ErrorModel)
		if !ok {
			return original
		}
		for _, detail := range model.Errors {
			detail.Value = nil
		}
		code := "http_" + strconv.Itoa(status)
		if status == http.StatusTooManyRequests {
			code = "rate_limited"
		}
		return &AccountProblem{ErrorModel: *model, Code: code}
	}
}
