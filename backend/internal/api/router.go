package api

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
)

func Register(mux *http.ServeMux) huma.API {
	config := huma.DefaultConfig("The Run API", "0.0.1")
	api := humago.New(mux, config)

	registerHello(api)
	registerRegistrations(api)

	return api
}
