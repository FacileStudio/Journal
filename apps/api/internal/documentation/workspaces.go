package docs

var errNoWorkspace = Error{Status: 404, Code: "not_found", Description: "the workspace parameter names a workspace the caller does not belong to"}

var workspacesModule = Module{
	Name: "workspaces",
	Description: "Every log entry, API key, saved query, alert rule and source map belongs to one workspace, and an account " +
		"reads only the workspaces it is a member of. Every session route below accepts workspace, a comma-separated list " +
		"of workspace ids, to narrow the read; without it a read covers every workspace of the caller. Keys, alert rules " +
		"and source maps are managed by the workspace's owners. A new account gets a personal workspace on its first request.",
	Routes: []Route{
		{
			Method:       "GET",
			Path:         "/workspaces",
			Summary:      "List the caller's workspaces",
			Description:  "Ordered by id. role is owner or member; personal marks the account's own workspace; plan_exempt workspaces are the instance operators' and carry no plan limit.",
			Auth:         "bearer",
			ResponseBody: `{"workspaces":[{"id":int64,"name":string,"role":string,"personal":bool,"plan_exempt":bool}]}`,
			Errors:       []Error{errUnauth, errNoWorkspace, errSessionRate},
		},
	},
}
