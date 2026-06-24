package api

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"

	"github.com/BirgerRydback/the-run/backend/internal/store"
)

func Register(mux *http.ServeMux, s store.Store) huma.API {
	config := huma.DefaultConfig("The Run API", "0.0.1")
	api := humago.New(mux, config)

	registerHello(api)
	registerRegistrations(api, s)

	return api
}
