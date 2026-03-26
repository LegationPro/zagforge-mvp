package routes

import (
	corsmw "github.com/LegationPro/zagforge/api/internal/middleware/cors"
	"github.com/LegationPro/zagforge/shared/go/router"
)

func registerContext(r *router.Router, d *Deps) error {
	g := r.Group()
	g.Use(corsmw.Cors(d.CORSOrigins))
	return g.Create([]router.Subroute{
		{Method: router.HEAD, Path: "/v1/context/{token}", Handler: d.ContextURL.Head},
		{Method: router.GET, Path: "/v1/context/{token}", Handler: d.ContextURL.Get},
	})
}
