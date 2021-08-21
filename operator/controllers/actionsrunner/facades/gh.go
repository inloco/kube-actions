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
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
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

	githubClient       *github.Client
	githubClientExpiry time.Time
	githubClientMutext sync.Mutex

	githubRepositories       sync.Map
	githubRepositoriesMutext sync.Mutex

	githubRegistrationTokens       sync.Map
	githubRegistrationTokensMutext sync.Mutex

	githubRemoveTokens       sync.Map
	githubRemoveTokensMutext sync.Mutex
)

func getGitHubAppToken() (string, error) {
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

func newGitHubAppClient(ctx context.Context) (*github.Client, error) {
	token, err := getGitHubAppToken()
	if err != nil {
		return nil, err
	}

	appClient := github.NewClient(
		oauth2.NewClient(
			ctx,
			oauth2.StaticTokenSource(
				&oauth2.Token{
					AccessToken: token,
				},
			),
		),
	)
	return appClient, nil
}

func getGitHubInstallationToken(ctx context.Context, appClient *github.Client) (*github.InstallationToken, error) {
	if appClient == nil {
		return nil, errors.New("appClient == nil")
	}

	var installationId int64
	if githubInstlId != "" {
		id, err := strconv.ParseInt(githubInstlId, 10, 0)
		if err != nil {
			return nil, err
		}

		installationId = id
	} else {
		log.Print(`githubInstlId == ""`)

		installations, _, err := appClient.Apps.ListInstallations(ctx, nil)
		if err != nil {
			return nil, err
		}

		if len(installations) != 1 {
			return nil, errors.New("len(installations) != 1")
		}

		installationId = installations[0].GetID()
	}

	installationToken, _, err := appClient.Apps.CreateInstallationToken(ctx, installationId, nil)
	if err != nil {
		return nil, err
	}

	return installationToken, nil
}

func initGitHubClient(ctx context.Context) error {
	githubClientMutext.Lock()
	defer githubClientMutext.Unlock()

	if githubClient != nil && (githubClientExpiry.IsZero() || githubClientExpiry.After(time.Now().Add(time.Minute))) {
		return nil
	}

	token := oauth2.Token{
		AccessToken: githubPAT,
	}
	if token.AccessToken == "" {
		appClient, err := newGitHubAppClient(ctx)
		if err != nil {
			log.Print(err)
		}

		githubIAT, err := getGitHubInstallationToken(ctx, appClient)
		if err != nil {
			return err
		}

		token.AccessToken = githubIAT.GetToken()
		token.Expiry = githubIAT.GetExpiresAt()
	}

	githubClient = github.NewClient(
		oauth2.NewClient(
			ctx,
			oauth2.StaticTokenSource(&token),
		),
	)
	githubClientExpiry = token.Expiry

	return nil
}

func getGitHubRepository(ctx context.Context, owner string, name string) (*github.Repository, error) {
	key := fmt.Sprintf("%s/%s", owner, name)

	if repository, ok := githubRepositories.Load(key); ok {
		return repository.(*github.Repository), nil
	}

	githubRepositoriesMutext.Lock()
	defer githubRepositoriesMutext.Unlock()

	if repository, ok := githubRepositories.Load(key); ok {
		return repository.(*github.Repository), nil
	}

	if githubClient == nil {
		return nil, errors.New("githubClient == nil")
	}

	repository, githubResponse, err := githubClient.Repositories.Get(ctx, owner, name)
	if err != nil {
		return nil, err
	}
	if githubResponse.StatusCode < 200 || githubResponse.StatusCode >= 300 {
		return nil, errors.New(githubResponse.Status)
	}

	if _, ok := githubOwners[repository.GetOwner().GetLogin()]; !ok {
		return nil, errors.New("not in githubOwners")
	}

	if private := repository.GetPrivate(); (private && githubVisibilities&repoPrivate == 0) || (!private && githubVisibilities&repoPublic == 0) {
		return nil, errors.New("not in githubVisibilities")
	}

	githubRepositories.Store(key, repository)

	return repository, nil
}

func getGitHubRegistrationToken(ctx context.Context, repository *github.Repository) (*github.RegistrationToken, error) {
	if repository == nil {
		return nil, errors.New("repository == nil")
	}

	key := fmt.Sprintf("%s/%s", repository.GetOwner(), repository.GetName())

	if token, ok := githubRegistrationTokens.Load(key); ok && token.(*github.RegistrationToken).GetExpiresAt().After(time.Now().Add(time.Minute)) {
		return token.(*github.RegistrationToken), nil
	}

	githubRegistrationTokensMutext.Lock()
	defer githubRegistrationTokensMutext.Unlock()

	if token, ok := githubRegistrationTokens.Load(key); ok && token.(*github.RegistrationToken).GetExpiresAt().After(time.Now().Add(time.Minute)) {
		return token.(*github.RegistrationToken), nil
	}

	if githubClient == nil {
		return nil, errors.New("githubClient == nil")
	}

	registrationToken, githubResponse, err := githubClient.Actions.CreateRegistrationToken(ctx, repository.GetOwner().GetLogin(), repository.GetName())
	if err != nil {
		return nil, err
	}
	if githubResponse.StatusCode < 200 || githubResponse.StatusCode >= 300 {
		return nil, errors.New(githubResponse.Status)
	}

	githubRegistrationTokens.Store(key, registrationToken)

	return registrationToken, nil
}

func newGitHubBridgeClientWithRegistrationToken(ctx context.Context, repository *github.Repository) (*github.Client, error) {
	if repository == nil {
		return nil, errors.New("repository == nil")
	}

	registrationToken, err := getGitHubRegistrationToken(ctx, repository)
	if err != nil {
		return nil, err
	}

	bridgeClient := github.NewClient(
		oauth2.NewClient(
			ctx,
			oauth2.StaticTokenSource(
				&oauth2.Token{
					AccessToken: registrationToken.GetToken(),
					TokenType:   "RemoteAuth",
				},
			),
		),
	)
	return bridgeClient, nil
}

func getGitHubRemoveToken(ctx context.Context, repository *github.Repository) (*github.RemoveToken, error) {
	if repository == nil {
		return nil, errors.New("repository == nil")
	}

	key := fmt.Sprintf("%s/%s", repository.GetOwner(), repository.GetName())

	if token, ok := githubRemoveTokens.Load(key); ok && token.(*github.RemoveToken).GetExpiresAt().After(time.Now().Add(time.Minute)) {
		return token.(*github.RemoveToken), nil
	}

	githubRemoveTokensMutext.Lock()
	defer githubRemoveTokensMutext.Unlock()

	if token, ok := githubRemoveTokens.Load(key); ok && token.(*github.RemoveToken).GetExpiresAt().After(time.Now().Add(time.Minute)) {
		return token.(*github.RemoveToken), nil
	}

	if githubClient == nil {
		return nil, errors.New("githubClient == nil")
	}

	removeToken, githubResponse, err := githubClient.Actions.CreateRemoveToken(ctx, repository.GetOwner().GetLogin(), repository.GetName())
	if err != nil {
		return nil, err
	}
	if githubResponse.StatusCode < 200 || githubResponse.StatusCode >= 300 {
		return nil, errors.New(githubResponse.Status)
	}

	githubRemoveTokens.Store(key, removeToken)

	return removeToken, nil
}

func newGitHubBridgeClientWithRemoveToken(ctx context.Context, repository *github.Repository) (*github.Client, error) {
	if repository == nil {
		return nil, errors.New("repository == nil")
	}

	removeToken, err := getGitHubRemoveToken(ctx, repository)
	if err != nil {
		return nil, err
	}

	bridgeClient := github.NewClient(
		oauth2.NewClient(
			ctx,
			oauth2.StaticTokenSource(
				&oauth2.Token{
					AccessToken: removeToken.GetToken(),
					TokenType:   "RemoteAuth",
				},
			),
		),
	)
	return bridgeClient, nil
}

type GitHub struct {
	Repository   *github.Repository
	BridgeClient *github.Client
}

func (gh *GitHub) InitWithRegistrationToken(ctx context.Context, repoOwner string, repoName string) error {
	if err := initGitHubClient(ctx); err != nil {
		return err
	}

	repository, err := getGitHubRepository(ctx, repoOwner, repoName)
	if err != nil {
		return err
	}
	gh.Repository = repository

	bridgeClient, err := newGitHubBridgeClientWithRegistrationToken(ctx, repository)
	if err != nil {
		return err
	}
	gh.BridgeClient = bridgeClient

	return nil
}

func (gh *GitHub) InitWithRemoveToken(ctx context.Context, repoOwner string, repoName string) error {
	if err := initGitHubClient(ctx); err != nil {
		return err
	}

	repository, err := getGitHubRepository(ctx, repoOwner, repoName)
	if err != nil {
		return err
	}
	gh.Repository = repository

	bridgeClient, err := newGitHubBridgeClientWithRemoveToken(ctx, repository)
	if err != nil {
		return err
	}
	gh.BridgeClient = bridgeClient

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
