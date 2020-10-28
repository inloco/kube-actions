/*
Copyright 2020 In Loco Tecnologia da Informação S.A.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package facades

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-github/v32/github"
	"github.com/square/go-jose/v3"
	"github.com/square/go-jose/v3/jwt"
	"golang.org/x/oauth2"
)

const (
	typJWT = "JWT"
	expJWT = 10 * time.Minute
)

var (
	pem2base64 = regexp.MustCompile(`(-----.+?-----)|[\n ]`)
)

type repositoryVisibility uint8

const (
	repoPublic repositoryVisibility = 1 << iota
	repoPrivate
)

var (
	githubPAT     = os.Getenv("KUBEACTIONS_GITHUB_PAT")
	githubAppId   = os.Getenv("KUBEACTIONS_GITHUB_APP_ID")
	githubAppPK   = os.Getenv("KUBEACTIONS_GITHUB_APP_PK")
	githubInstlId = os.Getenv("KUBEACTIONS_GITHUB_INSTL_ID")

	githubOwners = func() map[string]struct{} {
		allowed := make(map[string]struct{})

		owners := os.Getenv("KUBEACTIONS_GITHUB_OWNERS")
		for _, allowedOwner := range strings.Split(owners, ",") {
			allowed[allowedOwner] = struct{}{}
		}

		return allowed
	}()

	githubVisibilities = func() repositoryVisibility {
		allowed := repositoryVisibility(0)

		visibility := os.Getenv("KUBEACTIONS_GITHUB_VISIBILITIES")
		for _, allowedVisibility := range strings.Split(visibility, ",") {
			switch allowedVisibility {
			case "pub", "public":
				allowed |= repoPublic

			case "priv", "private":
				allowed |= repoPrivate

			default:
				log.Printf("unknown visility: %v", allowedVisibility)
			}
		}

		return allowed
	}()
)

type GitHub struct {
	AppClient *github.Client

	Client     *github.Client
	Repository *github.Repository

	BridgeClient *github.Client
}

func (gh *GitHub) Init(ctx context.Context, repoOwner string, repoName string) error {
	if err := gh.initGitHubAppClient(ctx); err != nil {
		log.Print(err)
	}

	if err := gh.initGitHubClient(ctx); err != nil {
		return err
	}

	if err := gh.initGitHubRepository(ctx, repoOwner, repoName); err != nil {
		return err
	}

	if err := gh.initGitHubBridgeClient(ctx); err != nil {
		return err
	}

	return nil
}

func (gh *GitHub) GetGitHubAppToken() (string, error) {
	if githubAppId == "" {
		return "", errors.New(`githubAppId == ""`)
	}

	if githubAppPK == "" {
		return "", errors.New(`githubAppPK == ""`)
	}

	der, err := base64.StdEncoding.DecodeString(string(pem2base64.ReplaceAll([]byte(githubAppPK), []byte{})))
	if err != nil {
		return "", err
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(der)
	if err != nil {
		return "", err
	}

	signer, err := jose.NewSigner(
		jose.SigningKey{
			Algorithm: jose.RS256,
			Key:       privateKey,
		},
		new(jose.SignerOptions).WithType(typJWT),
	)
	if err != nil {
		return "", err
	}

	now := time.Now()
	claims := jwt.Claims{
		Issuer:   githubAppId,
		Expiry:   jwt.NewNumericDate(now.Add(expJWT)),
		IssuedAt: jwt.NewNumericDate(now),
	}

	token, err := jwt.Signed(signer).Claims(claims).CompactSerialize()
	if err != nil {
		return "", err
	}

	return token, nil
}

func (gh *GitHub) initGitHubAppClient(ctx context.Context) error {
	token, err := gh.GetGitHubAppToken()
	if err != nil {
		return err
	}

	gh.AppClient = github.NewClient(
		oauth2.NewClient(
			ctx,
			oauth2.StaticTokenSource(
				&oauth2.Token{
					AccessToken: token,
				},
			),
		),
	)
	return nil
}

func (gh *GitHub) GetGitHubInstallationToken(ctx context.Context) (string, error) {
	if gh.AppClient == nil {
		return "", errors.New(".AppClient == nil")
	}

	var installationId int64
	if githubInstlId != "" {
		id, err := strconv.ParseInt(githubInstlId, 10, 0)
		if err != nil {
			return "", err
		}

		installationId = id
	} else {
		log.Print(`githubInstlId == ""`)

		installations, _, err := gh.AppClient.Apps.ListInstallations(ctx, nil)
		if err != nil {
			return "", err
		}

		if len(installations) != 1 {
			return "", errors.New("len(installations) != 1")
		}

		installationId = installations[0].GetID()
	}

	installationToken, _, err := gh.AppClient.Apps.CreateInstallationToken(ctx, installationId, nil)
	if err != nil {
		return "", err
	}

	return installationToken.GetToken(), nil
}

func (gh *GitHub) initGitHubClient(ctx context.Context) error {
	token := githubPAT
	if token == "" {
		log.Print(`githubPAT == ""`)

		githubIAT, err := gh.GetGitHubInstallationToken(ctx)
		if err != nil {
			return err
		}

		token = githubIAT
	}

	gh.Client = github.NewClient(
		oauth2.NewClient(
			ctx,
			oauth2.StaticTokenSource(
				&oauth2.Token{
					AccessToken: token,
				},
			),
		),
	)
	return nil
}

func (gh *GitHub) initGitHubRepository(ctx context.Context, owner string, name string) error {
	if gh.Client == nil {
		return errors.New(".Client == nil")
	}

	repository, githubResponse, err := gh.Client.Repositories.Get(ctx, owner, name)
	if err != nil {
		return err
	}
	if githubResponse.StatusCode < 200 || githubResponse.StatusCode >= 300 {
		return errors.New(githubResponse.Status)
	}

	if _, ok := githubOwners[repository.GetOwner().GetLogin()]; !ok {
		return errors.New("not in githubOwners")
	}

	if private := repository.GetPrivate(); (private && githubVisibilities&repoPrivate == 0) || (!private && githubVisibilities&repoPublic == 0) {
		return errors.New("not in githubVisibilities")
	}

	gh.Repository = repository
	return nil
}

func (gh *GitHub) GetGitHubRegistrationToken(ctx context.Context) (*github.RegistrationToken, error) {
	if gh.Client == nil {
		return nil, errors.New(".Client == nil")
	}

	token, githubResponse, err := gh.Client.Actions.CreateRegistrationToken(ctx, gh.Repository.GetOwner().GetLogin(), gh.Repository.GetName())
	if err != nil {
		return nil, err
	}
	if githubResponse.StatusCode < 200 || githubResponse.StatusCode >= 300 {
		return nil, errors.New(githubResponse.Status)
	}

	return token, nil
}

func (gh *GitHub) GetGitHubRemoveToken(ctx context.Context) (*github.RemoveToken, error) {
	if gh.Client == nil {
		return nil, errors.New(".Client == nil")
	}

	token, githubResponse, err := gh.Client.Actions.CreateRemoveToken(ctx, gh.Repository.GetOwner().GetLogin(), gh.Repository.GetName())
	if err != nil {
		return nil, err
	}
	if githubResponse.StatusCode < 200 || githubResponse.StatusCode >= 300 {
		return nil, errors.New(githubResponse.Status)
	}

	return token, nil
}

func (gh *GitHub) initGitHubBridgeClient(ctx context.Context) error {
	token, err := gh.GetGitHubRegistrationToken(ctx)
	if err != nil {
		return err
	}

	gh.BridgeClient = github.NewClient(
		oauth2.NewClient(
			ctx,
			oauth2.StaticTokenSource(
				&oauth2.Token{
					AccessToken: token.GetToken(),
					TokenType:   "RemoteAuth",
				},
			),
		),
	)
	return nil
}

type RunnerEvent string

const (
	RunnerEventRegister RunnerEvent = "register"
	RunnerEventRemove   RunnerEvent = "remove"
)

func (gh *GitHub) GetGitHubTenantCredential(ctx context.Context, runnerEvent RunnerEvent) (*github.TenantCredential, error) {
	if gh.BridgeClient == nil {
		return nil, errors.New(".BridgeClient == nil")
	}

	tenantCredential, githubResponse, err := gh.BridgeClient.Actions.CreateTenantCredential(ctx, string(runnerEvent), gh.Repository.GetHTMLURL())
	if err != nil {
		return nil, err
	}
	if githubResponse.StatusCode < 200 || githubResponse.StatusCode >= 300 {
		return nil, errors.New(githubResponse.Status)
	}

	return tenantCredential, nil
}
