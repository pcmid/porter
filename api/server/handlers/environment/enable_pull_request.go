package environment

import (
	"bytes"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/go-github/v41/github"
	"github.com/porter-dev/porter/api/server/authz"
	"github.com/porter-dev/porter/api/server/handlers"
	"github.com/porter-dev/porter/api/server/shared"
	"github.com/porter-dev/porter/api/server/shared/apierrors"
	"github.com/porter-dev/porter/api/server/shared/config"
	"github.com/porter-dev/porter/api/types"
	"github.com/porter-dev/porter/internal/models"
)

type EnablePullRequestHandler struct {
	handlers.PorterHandlerReadWriter
	authz.KubernetesAgentGetter
}

func NewEnablePullRequestHandler(
	config *config.Config,
	decoderValidator shared.RequestDecoderValidator,
	writer shared.ResultWriter,
) *EnablePullRequestHandler {
	return &EnablePullRequestHandler{
		PorterHandlerReadWriter: handlers.NewDefaultPorterHandler(config, decoderValidator, writer),
		KubernetesAgentGetter:   authz.NewOutOfClusterAgentGetter(config),
	}
}

func (c *EnablePullRequestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cluster, _ := r.Context().Value(types.ClusterScope).(*models.Cluster)
	request := &types.PullRequest{}

	if ok := c.DecodeAndValidate(w, r, request); !ok {
		return
	}

	env, err := c.Repo().Environment().ReadEnvironmentByOwnerRepoName(request.RepoOwner, request.RepoName)

	if err != nil {
		c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	client, err := getGithubClientFromEnvironment(c.Config(), env)

	if err != nil {
		c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	ghResp, err := client.Actions.CreateWorkflowDispatchEventByFileName(
		r.Context(), env.GitRepoOwner, env.GitRepoName, fmt.Sprintf("porter_%s_env.yml", env.Name),
		github.CreateWorkflowDispatchEventRequest{
			Ref: request.BranchFrom,
			Inputs: map[string]interface{}{
				"pr_number":      strconv.FormatUint(uint64(request.Number), 10),
				"pr_title":       request.Title,
				"pr_branch_from": request.BranchFrom,
				"pr_branch_into": request.BranchInto,
			},
		},
	)

	buf := new(bytes.Buffer)
	buf.ReadFrom(ghResp.Body)
	responseString := buf.String()

	fmt.Println(responseString)

	if ghResp.StatusCode == 404 {
		c.HandleAPIError(w, r, apierrors.NewErrPassThroughToClient(fmt.Errorf("workflow file not found"), 404))
		return
	}

	if err != nil {
		c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	// FIXME: hardcoded namespace
	namespace := fmt.Sprintf("pr-%d-%s", request.Number, strings.ReplaceAll(env.GitRepoName, "_", "-"))

	// create the deployment
	depl, err := c.Repo().Environment().CreateDeployment(&models.Deployment{
		EnvironmentID: env.ID,
		Namespace:     namespace,
		Status:        types.DeploymentStatusCreating,
		PullRequestID: request.Number,
		// GHDeploymentID: ghDeployment.GetID(),
		RepoOwner:    request.RepoOwner,
		RepoName:     request.RepoName,
		PRName:       request.Title,
		PRBranchFrom: request.BranchFrom,
		PRBranchInto: request.BranchInto,
	})

	if err != nil {
		c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	// create the backing namespace
	agent, err := c.GetAgent(r, cluster, "")

	if err != nil {
		c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	_, err = agent.CreateNamespace(depl.Namespace)

	if err != nil {
		c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

}
