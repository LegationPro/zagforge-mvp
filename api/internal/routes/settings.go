package routes

import "github.com/LegationPro/zagforge/shared/go/router"

func settingsSubroutes(d *Deps) []router.Subroute {
	return []router.Subroute{
		{Method: router.GET, Path: "/api/v1/repos/{repoID}/context-tokens", Handler: d.CtxTokens.List},
		{Method: router.POST, Path: "/api/v1/repos/{repoID}/context-tokens", Handler: d.CtxTokens.Create},
		{Method: router.DELETE, Path: "/api/v1/repos/{repoID}/context-tokens/{tokenID}", Handler: d.CtxTokens.Delete},
		{Method: router.GET, Path: "/api/v1/orgs/settings/ai-keys", Handler: d.AIKeys.List},
		{Method: router.PUT, Path: "/api/v1/orgs/settings/ai-keys", Handler: d.AIKeys.Upsert},
		{Method: router.DELETE, Path: "/api/v1/orgs/settings/ai-keys/{provider}", Handler: d.AIKeys.Delete},
		{Method: router.POST, Path: "/api/v1/repos/{repoID}/query", Handler: d.Query.Query},
	}
}
