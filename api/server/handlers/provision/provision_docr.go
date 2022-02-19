package provision

import (
	"encoding/json"
	"net/http"

	"github.com/porter-dev/porter/api/server/handlers"
	"github.com/porter-dev/porter/api/server/shared"
	"github.com/porter-dev/porter/api/server/shared/apierrors"
	"github.com/porter-dev/porter/api/server/shared/config"
	"github.com/porter-dev/porter/api/types"
	"github.com/porter-dev/porter/internal/analytics"
	"github.com/porter-dev/porter/internal/kubernetes/provisioner"
	"github.com/porter-dev/porter/internal/kubernetes/provisioner/do/docr"
	"github.com/porter-dev/porter/internal/models"
	"github.com/porter-dev/porter/internal/repository"
	"gorm.io/gorm"
)

type ProvisionDOCRHandler struct {
	handlers.PorterHandlerReadWriter
}

func NewProvisionDOCRHandler(
	config *config.Config,
	decoderValidator shared.RequestDecoderValidator,
	writer shared.ResultWriter,
) *ProvisionDOCRHandler {
	return &ProvisionDOCRHandler{
		PorterHandlerReadWriter: handlers.NewDefaultPorterHandler(config, decoderValidator, writer),
	}
}

func (c *ProvisionDOCRHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// read the user and project from context
	user, _ := r.Context().Value(types.UserScope).(*models.User)
	proj, _ := r.Context().Value(types.ProjectScope).(*models.Project)

	request := &types.CreateDOCRInfraRequest{
		ProjectID: proj.ID,
	}

	if ok := c.DecodeAndValidate(w, r, request); !ok {
		return
	}

	// get the DO integration, to check that integration exists and belongs to the project
	doInt, err := c.Repo().OAuthIntegration().ReadOAuthIntegration(proj.ID, request.DOIntegrationID)

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.HandleAPIError(w, r, apierrors.NewErrForbidden(err))
		} else {
			c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		}

		return
	}

	suffix, err := repository.GenerateRandomBytes(6)

	if err != nil {
		c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	lastApplied, err := json.Marshal(request)

	// parse infra last applied into DOCR config
	if err != nil {
		c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	infra := &models.Infra{
		Kind:            types.InfraDOCR,
		ProjectID:       proj.ID,
		Suffix:          suffix,
		Status:          types.StatusCreating,
		DOIntegrationID: request.DOIntegrationID,
		CreatedByUserID: user.ID,
		LastApplied:     lastApplied,
	}

	// handle write to the database
	infra, err = c.Repo().Infra().CreateInfra(infra)

	if err != nil {
		c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	opts, err := GetSharedProvisionerOpts(c.Config(), infra)

	if err != nil {
		c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	vaultToken := ""

	if c.Config().CredentialBackend != nil {
		vaultToken, err = c.Config().CredentialBackend.CreateOAuthToken(doInt)

		if err != nil {
			c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
			return
		}
	}

	opts.CredentialExchange.VaultToken = vaultToken
	opts.DOCR = &docr.Conf{
		DOCRName:             request.DOCRName,
		DOCRSubscriptionTier: request.DOCRSubscriptionTier,
	}

	opts.OperationKind = provisioner.Apply

	err = c.Config().ProvisionerAgent.Provision(opts)

	if err != nil {
		infra.Status = types.StatusError
		infra, _ = c.Repo().Infra().UpdateInfra(infra)
		c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	c.Config().AnalyticsClient.Track(analytics.RegistryProvisioningStartTrack(
		&analytics.RegistryProvisioningStartTrackOpts{
			ProjectScopedTrackOpts: analytics.GetProjectScopedTrackOpts(user.ID, proj.ID),
			RegistryType:           types.InfraDOCR,
			InfraID:                infra.ID,
		},
	))

	c.WriteResult(w, r, infra.ToInfraType())
}