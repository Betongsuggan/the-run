package api

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"

	"github.com/BirgerRydback/the-run/backend/internal/audit"
	"github.com/BirgerRydback/the-run/backend/internal/auth"
	"github.com/BirgerRydback/the-run/backend/internal/email"
	"github.com/BirgerRydback/the-run/backend/internal/store"
)

func Register(
	mux *http.ServeMux,
	s store.Store,
	authCfg auth.Config,
	sender email.Sender,
	renderer *email.Renderer,
	recorder audit.Recorder,
) huma.API {
	config := huma.DefaultConfig("The Run API", "0.0.1")
	api := humago.New(mux, config)

	registerHello(api)
	registerAuth(api, s, authCfg)
	registerRegistrations(api, s, renderer)
	registerEvents(api, s, authCfg)
	registerRaces(api, s, authCfg)
	registerRunners(api, s, authCfg)
	registerAdminRegistrations(api, s, authCfg, recorder)
	registerResults(api, s, authCfg)
	registerGuardianConsent(api, s)
	registerDSR(api, s, authCfg, renderer, recorder)
	registerPolicies(api, s)
	registerAdminPolicies(api, s, authCfg, recorder)
	registerAdminEmailTemplates(api, s, authCfg, recorder)
	registerAdminUsers(api, s, authCfg, renderer, recorder)

	return api
}
