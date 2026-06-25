package api

import (
	"github.com/danielgtaylor/huma/v2"

	"github.com/BirgerRydback/the-run/backend/internal/auth"
)

// adminMiddlewares is the per-operation middleware list applied to every
// admin-only Huma operation. Centralised so adding additional checks (rate
// limiting, audit, etc.) later means editing one place.
func adminMiddlewares(cfg auth.Config) huma.Middlewares {
	return huma.Middlewares{auth.RequireAdmin(cfg)}
}
