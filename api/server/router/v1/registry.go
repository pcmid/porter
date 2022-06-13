package v1

import (
	"fmt"

	"github.com/go-chi/chi"
	"github.com/porter-dev/porter/api/server/handlers/registry"
	"github.com/porter-dev/porter/api/server/shared"
	"github.com/porter-dev/porter/api/server/shared/config"
	"github.com/porter-dev/porter/api/server/shared/router"
	"github.com/porter-dev/porter/api/types"
)

// swagger:parameters getRegistry deleteRegistry createRegistryRepository listRegistryRepositories listRegistryImages
type registryPathParams struct {
	// The project id
	// in: path
	// required: true
	// minimum: 1
	ProjectID uint `json:"project_id"`

	// The registry id
	// in: path
	// required: true
	// minimum: 1
	RegistryID uint `json:"registry_id"`
}

func NewV1RegistryScopedRegisterer(children ...*router.Registerer) *router.Registerer {
	return &router.Registerer{
		GetRoutes: GetV1RegistryScopedRoutes,
		Children:  children,
	}
}

func GetV1RegistryScopedRoutes(
	r chi.Router,
	config *config.Config,
	basePath *types.Path,
	factory shared.APIEndpointFactory,
	children ...*router.Registerer,
) []*router.Route {
	routes, projPath := getV1RegistryRoutes(r, config, basePath, factory)

	if len(children) > 0 {
		r.Route(projPath.RelativePath, func(r chi.Router) {
			for _, child := range children {
				childRoutes := child.GetRoutes(r, config, basePath, factory, child.Children...)

				routes = append(routes, childRoutes...)
			}
		})
	}

	return routes
}

func getV1RegistryRoutes(
	r chi.Router,
	config *config.Config,
	basePath *types.Path,
	factory shared.APIEndpointFactory,
) ([]*router.Route, *types.Path) {
	relPath := "/registries"

	newPath := &types.Path{
		Parent:       basePath,
		RelativePath: relPath,
	}

	var routes []*router.Route

	// POST /api/v1/projects/{project_id}/registries -> registry.NewRegistryCreateHandler
	// swagger:operation POST /api/v1/projects/{project_id}/registries createRegistry
	//
	// Connects a new image registry
	//
	// ---
	// produces:
	// - application/json
	// summary: Connect an image registry
	// tags:
	// - Registries
	// parameters:
	//   - name: project_id
	//   - in: body
	//     name: CreateRegistryRequest
	//     description: The registry to connect
	//     schema:
	//       $ref: '#/definitions/CreateRegistryRequest'
	// responses:
	//   '201':
	//     description: Successfully connected the registry
	//     schema:
	//       $ref: '#/definitions/CreateRegistryResponse'
	//   '403':
	//     description: Forbidden
	createRegistryEndpoint := factory.NewAPIEndpoint(
		&types.APIRequestMetadata{
			Verb:   types.APIVerbCreate,
			Method: types.HTTPVerbPost,
			Path: &types.Path{
				Parent:       basePath,
				RelativePath: relPath,
			},
			Scopes: []types.PermissionScope{
				types.UserScope,
				types.ProjectScope,
			},
		},
	)

	createRegistryHandler := registry.NewRegistryCreateHandler(
		config,
		factory.GetDecoderValidator(),
		factory.GetResultWriter(),
	)

	routes = append(routes, &router.Route{
		Endpoint: createRegistryEndpoint,
		Handler:  createRegistryHandler,
		Router:   r,
	})

	// GET /api/v1/projects/{project_id}/registries/{registry_id} -> registry.NewRegistryGetHandler
	// swagger:operation GET /api/v1/projects/{project_id}/registries/{registry_id} getRegistry
	//
	// Gets an image registry
	//
	// ---
	// produces:
	// - application/json
	// summary: Get an image registry
	// tags:
	// - Registries
	// parameters:
	//   - name: project_id
	//   - name: registry_id
	// responses:
	//   '201':
	//     description: Successfully got the registry
	//     schema:
	//       $ref: '#/definitions/GetRegistryResponse'
	//   '403':
	//     description: Forbidden
	getEndpoint := factory.NewAPIEndpoint(
		&types.APIRequestMetadata{
			Verb:   types.APIVerbGet,
			Method: types.HTTPVerbGet,
			Path: &types.Path{
				Parent:       basePath,
				RelativePath: relPath + "/{registry_id}",
			},
			Scopes: []types.PermissionScope{
				types.UserScope,
				types.ProjectScope,
				types.RegistryScope,
			},
		},
	)

	getHandler := registry.NewRegistryGetHandler(
		config,
		factory.GetResultWriter(),
	)

	routes = append(routes, &router.Route{
		Endpoint: getEndpoint,
		Handler:  getHandler,
		Router:   r,
	})

	// GET /api/v1/projects/{project_id}/registries -> registry.NewRegistryListHandler
	// swagger:operation GET /api/v1/projects/{project_id}/registries listRegistries
	//
	// Lists registries
	//
	// ---
	// produces:
	// - application/json
	// summary: List image registries
	// tags:
	// - Registries
	// parameters:
	//   - name: project_id
	//   - name: registry_id
	// responses:
	//   '200':
	//     description: Successfully listed registries
	//     schema:
	//       $ref: '#/definitions/ListRegistriesResponse'
	//   '403':
	//     description: Forbidden
	listRegistriesEndpoint := factory.NewAPIEndpoint(
		&types.APIRequestMetadata{
			Verb:   types.APIVerbList,
			Method: types.HTTPVerbGet,
			Path: &types.Path{
				Parent:       basePath,
				RelativePath: relPath,
			},
			Scopes: []types.PermissionScope{
				types.UserScope,
				types.ProjectScope,
			},
		},
	)

	listRegistriesHandler := registry.NewRegistryListHandler(
		config,
		factory.GetResultWriter(),
	)

	routes = append(routes, &router.Route{
		Endpoint: listRegistriesEndpoint,
		Handler:  listRegistriesHandler,
		Router:   r,
	})

	// DELETE /api/v1/projects/{project_id}/registries/{registry_id} -> registry.NewRegistryDeleteHandler
	// swagger:operation DELETE /api/v1/projects/{project_id}/registries/{registry_id} deleteRegistry
	//
	// Deletes an image registry.
	//
	// ---
	// produces:
	// - application/json
	// summary: Disconnect image registry
	// tags:
	// - Registries
	// parameters:
	//   - name: project_id
	//   - name: registry_id
	// responses:
	//   '200':
	//     description: Successfully disconnected image registry
	//   '403':
	//     description: Forbidden
	deleteEndpoint := factory.NewAPIEndpoint(
		&types.APIRequestMetadata{
			Verb:   types.APIVerbDelete,
			Method: types.HTTPVerbDelete,
			Path: &types.Path{
				Parent:       basePath,
				RelativePath: relPath + "/{registry_id}",
			},
			Scopes: []types.PermissionScope{
				types.UserScope,
				types.ProjectScope,
				types.RegistryScope,
			},
		},
	)

	deleteHandler := registry.NewRegistryDeleteHandler(
		config,
		factory.GetResultWriter(),
	)

	routes = append(routes, &router.Route{
		Endpoint: deleteEndpoint,
		Handler:  deleteHandler,
		Router:   r,
	})

	// POST /api/v1/projects/{project_id}/registries/{registry_id}/repositories -> registry.NewRegistryCreateRepositoryHandler
	// swagger:operation POST /api/v1/projects/{project_id}/registries/{registry_id}/repositories createRegistryRepository
	//
	// Creates an image repository inside the registry specified by `registry_id`. This method **only** creates repositories for ECR-integrated
	// repositories.
	//
	// ---
	// produces:
	// - application/json
	// summary: Create image repository
	// tags:
	// - Registries
	// parameters:
	//   - name: project_id
	//   - name: registry_id
	//   - in: body
	//     name: CreateRepositoryRequest
	//     description: The repository to create
	//     schema:
	//       $ref: '#/definitions/CreateRegistryRepositoryRequest'
	// responses:
	//   '201':
	//     description: Successfully created the image repository
	//   '403':
	//     description: Forbidden
	createRepositoryEndpoint := factory.NewAPIEndpoint(
		&types.APIRequestMetadata{
			Verb:   types.APIVerbCreate,
			Method: types.HTTPVerbPost,
			Path: &types.Path{
				Parent:       basePath,
				RelativePath: relPath + "/{registry_id}/repositories",
			},
			Scopes: []types.PermissionScope{
				types.UserScope,
				types.ProjectScope,
				types.RegistryScope,
			},
		},
	)

	createRepositoryHandler := registry.NewRegistryCreateRepositoryHandler(
		config,
		factory.GetDecoderValidator(),
		factory.GetResultWriter(),
	)

	routes = append(routes, &router.Route{
		Endpoint: createRepositoryEndpoint,
		Handler:  createRepositoryHandler,
		Router:   r,
	})

	// GET /api/v1/projects/{project_id}/registries/{registry_id}/repositories -> registry.NewRegistryListRepositoriesHandler
	// swagger:operation GET /api/v1/projects/{project_id}/registries/{registry_id}/repositories listRegistryRepositories
	//
	// Lists image repositories inside the image registry given by `registry_id`
	//
	// ---
	// produces:
	// - application/json
	// summary: List image repositories
	// tags:
	// - Registries
	// parameters:
	//   - name: project_id
	//   - name: registry_id
	// responses:
	//   '200':
	//     description: Successfully listed image repositories
	//     schema:
	//       $ref: '#/definitions/ListRegistryRepositoriesResponse'
	//   '403':
	//     description: Forbidden
	listRepositoriesEndpoint := factory.NewAPIEndpoint(
		&types.APIRequestMetadata{
			Verb:   types.APIVerbList,
			Method: types.HTTPVerbGet,
			Path: &types.Path{
				Parent:       basePath,
				RelativePath: relPath + "/{registry_id}/repositories",
			},
			Scopes: []types.PermissionScope{
				types.UserScope,
				types.ProjectScope,
				types.RegistryScope,
			},
		},
	)

	listRepositoriesHandler := registry.NewRegistryListRepositoriesHandler(
		config,
		factory.GetResultWriter(),
	)

	routes = append(routes, &router.Route{
		Endpoint: listRepositoriesEndpoint,
		Handler:  listRepositoriesHandler,
		Router:   r,
	})

	// GET /api/v1/projects/{project_id}/registries/{registry_id}/repositories/* -> registry.NewRegistryListImagesHandler
	// swagger:operation GET /api/v1/projects/{project_id}/registries/{registry_id}/repositories/{repository} listRegistryImages
	//
	// Lists all images in an image repository.
	//
	// ---
	// produces:
	// - application/json
	// summary: List images
	// tags:
	// - Registries
	// parameters:
	//   - name: project_id
	//   - name: registry_id
	//   - name: repository
	//     in: path
	//     description: the image repository name
	//     type: string
	//     required: true
	// responses:
	//   '200':
	//     description: Successfully listed images
	//     schema:
	//       $ref: '#/definitions/ListImagesResponse'
	//   '403':
	//     description: Forbidden
	listImagesEndpoint := factory.NewAPIEndpoint(
		&types.APIRequestMetadata{
			Verb:   types.APIVerbList,
			Method: types.HTTPVerbGet,
			Path: &types.Path{
				Parent: basePath,
				RelativePath: fmt.Sprintf(
					"%s/{registry_id}/repositories/%s",
					relPath,
					types.URLParamWildcard,
				),
			},
			Scopes: []types.PermissionScope{
				types.UserScope,
				types.ProjectScope,
				types.RegistryScope,
			},
		},
	)

	listImagesHandler := registry.NewRegistryListImagesHandler(
		config,
		factory.GetResultWriter(),
	)

	routes = append(routes, &router.Route{
		Endpoint: listImagesEndpoint,
		Handler:  listImagesHandler,
		Router:   r,
	})

	return routes, newPath
}
