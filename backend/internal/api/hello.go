package api

import (
	"context"
	"time"

	"github.com/danielgtaylor/huma/v2"
)

type HelloOutput struct {
	Body struct {
		Message   string `json:"message" example:"Hello from the-run" doc:"Greeting message"`
		Timestamp string `json:"timestamp" format:"date-time" doc:"Server-side timestamp (RFC3339)"`
	}
}

func registerHello(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "get-hello",
		Method:      "GET",
		Path:        "/hello",
		Summary:     "Greeting",
		Description: "Returns a hello message and the current server time.",
		Tags:        []string{"hello"},
	}, func(ctx context.Context, _ *struct{}) (*HelloOutput, error) {
		out := &HelloOutput{}
		out.Body.Message = "Hello from the-run"
		out.Body.Timestamp = time.Now().UTC().Format(time.RFC3339)
		return out, nil
	})
}
