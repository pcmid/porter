package stack

import (
	"fmt"
	"net/http"

	"github.com/porter-dev/porter/api/server/authz"
	"github.com/porter-dev/porter/api/server/handlers"
	"github.com/porter-dev/porter/api/server/shared"
	"github.com/porter-dev/porter/api/server/shared/apierrors"
	"github.com/porter-dev/porter/api/server/shared/config"
	"github.com/porter-dev/porter/api/types"
	"github.com/porter-dev/porter/internal/encryption"
	"github.com/porter-dev/porter/internal/models"
)

type StackCreateHandler struct {
	handlers.PorterHandlerReadWriter
	authz.KubernetesAgentGetter
}

func NewStackCreateHandler(
	config *config.Config,
	reader shared.RequestDecoderValidator,
	writer shared.ResultWriter,
) *StackCreateHandler {
	return &StackCreateHandler{
		PorterHandlerReadWriter: handlers.NewDefaultPorterHandler(config, reader, writer),
		KubernetesAgentGetter:   authz.NewOutOfClusterAgentGetter(config),
	}
}

func (p *StackCreateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	proj, _ := r.Context().Value(types.ProjectScope).(*models.Project)
	cluster, _ := r.Context().Value(types.ClusterScope).(*models.Cluster)
	namespace, _ := r.Context().Value(types.NamespaceScope).(string)

	req := &types.CreateStackRequest{}

	if ok := p.DecodeAndValidate(w, r, req); !ok {
		return
	}

	uid, err := encryption.GenerateRandomBytes(16)

	if err != nil {
		p.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	sourceConfigs, err := getSourceConfigModels(req.SourceConfigs)

	if err != nil {
		p.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	resources, err := getResourceModels(req.AppResources, sourceConfigs)

	if err != nil {
		p.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	// write stack to the database with creating status
	stack := &models.Stack{
		ProjectID: proj.ID,
		ClusterID: cluster.ID,
		Namespace: namespace,
		Name:      req.Name,
		UID:       uid,
		Revisions: []models.StackRevision{
			{
				RevisionNumber: 1,
				Status:         string(types.StackRevisionStatusDeploying),
				SourceConfigs:  sourceConfigs,
				Resources:      resources,
			},
		},
	}

	stack, err = p.Repo().Stack().CreateStack(stack)

	if err != nil {
		p.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	// apply all app resources
	registries, err := p.Repo().Registry().ListRegistriesByProjectID(cluster.ProjectID)

	if err != nil {
		p.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	helmAgent, err := p.GetHelmAgent(r, cluster, "")

	if err != nil {
		p.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	for _, appResource := range req.AppResources {
		err = applyAppResource(&applyAppResourceOpts{
			config:     p.Config(),
			projectID:  proj.ID,
			namespace:  namespace,
			cluster:    cluster,
			registries: registries,
			helmAgent:  helmAgent,
			request:    appResource,
		})

		if err != nil {
			// TODO: mark stack with error
			p.HandleAPIError(w, r, apierrors.NewErrInternal(err))
			return
		}
	}

	// update stack revision status
	revision := &stack.Revisions[0]
	revision.Status = string(types.StackRevisionStatusDeployed)

	revision, err = p.Repo().Stack().UpdateStackRevision(revision)

	if err != nil {
		p.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	// read the stack again to get the latest revision info
	stack, err = p.Repo().Stack().ReadStackByStringID(proj.ID, stack.UID)

	if err != nil {
		p.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	w.WriteHeader(http.StatusCreated)
	p.WriteResult(w, r, stack.ToStackType())
}

func getSourceConfigModels(sourceConfigs []*types.CreateStackSourceConfigRequest) ([]models.StackSourceConfig, error) {
	res := make([]models.StackSourceConfig, 0)

	// for now, only write source configs which are deployed as a docker image
	// TODO: add parsing/writes for git-based sources
	for _, sourceConfig := range sourceConfigs {
		if sourceConfig.StackSourceConfigBuild == nil {
			uid, err := encryption.GenerateRandomBytes(16)

			if err != nil {
				return nil, err
			}

			res = append(res, models.StackSourceConfig{
				UID:          uid,
				Name:         sourceConfig.Name,
				ImageRepoURI: sourceConfig.ImageRepoURI,
				ImageTag:     sourceConfig.ImageTag,
			})
		}
	}

	return res, nil
}

func getResourceModels(appResources []*types.CreateStackAppResourceRequest, sourceConfigs []models.StackSourceConfig) ([]models.StackResource, error) {
	res := make([]models.StackResource, 0)

	for _, appResource := range appResources {
		uid, err := encryption.GenerateRandomBytes(16)

		if err != nil {
			return nil, err
		}

		var linkedSourceConfigUID string

		for _, sourceConfig := range sourceConfigs {
			if sourceConfig.Name == appResource.SourceConfigName {
				linkedSourceConfigUID = sourceConfig.UID
			}
		}

		if linkedSourceConfigUID == "" {
			return nil, fmt.Errorf("source config %s does not exist in source config list", appResource.SourceConfigName)
		}

		res = append(res, models.StackResource{
			Name:                 appResource.Name,
			UID:                  uid,
			StackSourceConfigUID: linkedSourceConfigUID,
			TemplateRepoURL:      appResource.TemplateRepoURL,
			TemplateName:         appResource.TemplateName,
			TemplateVersion:      appResource.TemplateVersion,
			HelmRevisionID:       1,
		})
	}

	return res, nil
}