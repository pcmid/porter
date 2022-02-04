package credentials

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/porter-dev/porter/api/server/shared"
	"github.com/porter-dev/porter/api/server/shared/apierrors"
	"github.com/porter-dev/porter/ee/api/types"
	"github.com/porter-dev/porter/ee/integrations/vault"
	"github.com/porter-dev/porter/internal/models"
	"github.com/porter-dev/porter/internal/repository/credentials"
	"github.com/porter-dev/porter/internal/repository/gorm"
	"golang.org/x/crypto/bcrypt"

	"github.com/porter-dev/porter/provisioner/server/config"
)

type CredentialsGetHandler struct {
	config       *config.Config
	resultWriter shared.ResultWriter
}

func NewCredentialsGetHandler(
	config *config.Config,
) http.Handler {
	return &CredentialsGetHandler{
		config:       config,
		resultWriter: shared.NewDefaultResultWriter(config.Logger, config.Alerter),
	}
}

func (c *CredentialsGetHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// read the request to get the token id and hashed token
	req := &types.CredentialsExchangeRequest{}

	// populate the request from the headers
	req.CredExchangeToken = r.Header.Get("X-Porter-Token")
	tokID, err := strconv.ParseUint(r.Header.Get("X-Porter-Token-ID"), 10, 64)

	if err != nil {
		apierrors.HandleAPIError(c.config.Logger, c.config.Alerter, w, r, apierrors.NewErrForbidden(err), true)
		return
	}

	req.CredExchangeID = uint(tokID)
	req.VaultToken = r.Header.Get("X-Vault-Token")

	// read the access token in the header, check against DB
	ceToken, err := c.config.Repo.CredentialsExchangeToken().ReadCredentialsExchangeToken(req.CredExchangeID)

	if err != nil {
		apierrors.HandleAPIError(c.config.Logger, c.config.Alerter, w, r, apierrors.NewErrForbidden(err), true)
		return
	}

	// TODO: verify hashed token!!
	if valid, err := verifyToken(req.CredExchangeToken, ceToken); !valid {
		apierrors.HandleAPIError(c.config.Logger, c.config.Alerter, w, r, apierrors.NewErrForbidden(err), true)
		return
	}

	resp := &types.CredentialsExchangeResponse{}
	repo := c.config.Repo

	// if the request contains a vault token, use that vault token to construct a new repository
	// that will query vault using the passed in token
	if req.VaultToken != "" {
		// read the vault token in the header, create new vault client with this token
		conf := c.config.DBConf
		vaultClient := vault.NewClient(conf.VaultServerURL, req.VaultToken, conf.VaultPrefix)

		var key [32]byte

		for i, b := range []byte(conf.EncryptionKey) {
			key[i] = b
		}

		// use this vault client for the repo
		repo = gorm.NewRepository(c.config.DB, &key, vaultClient)
	}

	if ceToken.DOCredentialID != 0 {
		doInt, err := repo.OAuthIntegration().ReadOAuthIntegration(ceToken.ProjectID, ceToken.DOCredentialID)

		if err != nil {
			apierrors.HandleAPIError(c.config.Logger, c.config.Alerter, w, r, apierrors.NewErrForbidden(err), true)
			return
		}

		resp.DO = &credentials.OAuthCredential{
			ClientID:     doInt.ClientID,
			AccessToken:  doInt.AccessToken,
			RefreshToken: doInt.RefreshToken,
		}
	} else if ceToken.GCPCredentialID != 0 {
		gcpInt, err := repo.GCPIntegration().ReadGCPIntegration(ceToken.ProjectID, ceToken.GCPCredentialID)

		if err != nil {
			apierrors.HandleAPIError(c.config.Logger, c.config.Alerter, w, r, apierrors.NewErrForbidden(err), true)
			return
		}

		resp.GCP = &credentials.GCPCredential{
			GCPKeyData: gcpInt.GCPKeyData,
		}
	} else if ceToken.AWSCredentialID != 0 {
		awsInt, err := repo.AWSIntegration().ReadAWSIntegration(ceToken.ProjectID, ceToken.AWSCredentialID)

		if err != nil {
			apierrors.HandleAPIError(c.config.Logger, c.config.Alerter, w, r, apierrors.NewErrForbidden(err), true)
			return
		}

		resp.AWS = &credentials.AWSCredential{
			AWSAccessKeyID:     awsInt.AWSAccessKeyID,
			AWSClusterID:       awsInt.AWSClusterID,
			AWSSecretAccessKey: awsInt.AWSSecretAccessKey,
			AWSSessionToken:    awsInt.AWSSessionToken,
			AWSRegion:          []byte(awsInt.AWSRegion),
		}
	}

	// return the decrypted credentials
	c.resultWriter.WriteResult(w, r, resp)
}

func verifyToken(reqToken string, ceToken *models.CredentialsExchangeToken) (bool, error) {
	// make sure the token is still valid and has not expired
	if ceToken.IsExpired() {
		return false, fmt.Errorf("token is expired")
	}

	// make sure the token is correct
	if err := bcrypt.CompareHashAndPassword([]byte(ceToken.Token), []byte(reqToken)); err != nil {
		return false, fmt.Errorf("verify token failed: %s", err)
	}

	return true, nil
}
