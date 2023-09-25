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
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-jose/go-jose/v3"
	"github.com/go-jose/go-jose/v3/jwt"
	"github.com/google/go-github/v32/github"
	"github.com/inloco/kube-actions/operator/metrics"
	"github.com/patrickmn/go-cache"
	"golang.org/x/oauth2"
	"sigs.k8s.io/controller-runtime/pkg/log"
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
				logger := log.FromContext(context.TODO())
				logger.Info("unknown visility: " + allowedVisibility)
			}
		}

		return allowed
	}()

	githubClient       *github.Client
	githubClientExpiry time.Time
	githubClientMutext sync.Mutex

	githubRepositories       = cache.New(time.Hour, time.Hour)
	githubRepositoriesMutext sync.Mutex

	githubRegistrationTokens       = cache.New(time.Hour, time.Hour)
	githubRegistrationTokensMutext sync.Mutex

	githubRemoveTokens       = cache.New(time.Hour, time.Hour)
	githubRemoveTokensMutext sync.Mutex

	githubTenantCredentials       = cache.New(time.Hour, time.Hour)
	githubTenantCredentialsMutext sync.Mutex
)

func collectGitHubRateLimitMetrics(ctx context.Context, client *github.Client, clientName string) error {
	logger := log.FromContext(ctx)

	if client == nil {
		return errors.New("client == nil")
	}

	rateLimits, githubResponse, err := client.RateLimits(ctx)
	if err != nil {
		return err
	}
	if githubResponse.StatusCode < 200 || githubResponse.StatusCode >= 300 {
		return errors.New(githubResponse.Status)
	}
	logger.Info(rateLimits.String(), "clientName", clientName)

	core := rateLimits.GetCore()
	if core == nil {
		return errors.New("core == nil")
	}

	metrics.SetGitHubRateLimitCollector(clientName, core.Limit)
	metrics.SetGitHubRateRemainingCollector(clientName, core.Remaining)
	return nil
}

func tryCollectGitHubRateLimitMetrics(ctx context.Context, client *github.Client, clientName string) {
	logger := log.FromContext(ctx)

	if err := collectGitHubRateLimitMetrics(ctx, client, clientName); err != nil {
		logger.Error(err, err.Error(), "clientName", clientName)
	}
}

func collectGitHubAPICallMetrics(ctx context.Context, client *github.Client, clientName string, response *github.Response) error {
	tryCollectGitHubRateLimitMetrics(ctx, client, clientName)

	if response == nil {
		return errors.New("response == nil")
	}
	resLabelValue := fmt.Sprintf("%s", response.Status)

	request := response.Request
	if request == nil {
		return errors.New("request == nil")
	}
	reqLabelValue := fmt.Sprintf("%s %s", request.Method, request.URL.String())

	metrics.IncGitHubAPICallsCollector(clientName, reqLabelValue, resLabelValue)
	return nil
}

func tryCollectGitHubAPICallMetrics(ctx context.Context, client *github.Client, clientName string, response *github.Response) {
	logger := log.FromContext(ctx)

	if err := collectGitHubAPICallMetrics(ctx, client, clientName, response); err != nil {
		logger.Error(err, err.Error(), "clientName", clientName)
	}
}

func handleGitHubResponse(ctx context.Context, client *github.Client, clientName string, response *github.Response, err error) error {
	tryCollectGitHubAPICallMetrics(ctx, client, clientName, response)

	if err != nil {
		return err
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("Error response from GitHub: " + response.Status)
	}

	return nil
}

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
	logger := log.FromContext(ctx)

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
		logger.Info(`githubInstlId == ""`)

		installations, githubResponse, err := appClient.Apps.ListInstallations(ctx, nil)
		if err := handleGitHubResponse(ctx, appClient, "app", githubResponse, err); err != nil {
			return nil, err
		}

		if len(installations) != 1 {
			return nil, errors.New("len(installations) != 1")
		}

		installationId = installations[0].GetID()
	}

	installationToken, githubResponse, err := appClient.Apps.CreateInstallationToken(ctx, installationId, nil)
	if err := handleGitHubResponse(ctx, appClient, "app", githubResponse, err); err != nil {
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
			return err
		}

		githubIAT, err := getGitHubInstallationToken(ctx, appClient)
		if err != nil {
			return err
		}

		token.AccessToken = githubIAT.GetToken()
		token.Expiry = githubIAT.GetExpiresAt()
	}

	client := github.NewClient(
		oauth2.NewClient(
			ctx,
			oauth2.StaticTokenSource(&token),
		),
	)

	if err := collectGitHubRateLimitMetrics(ctx, client, ""); err != nil {
		return err
	}

	githubClient = client
	githubClientExpiry = token.Expiry
	return nil
}

func getGitHubRepository(ctx context.Context, owner string, name string) (*github.Repository, error) {
	key := fmt.Sprintf("%s/%s", owner, name)

	if repository, ok := githubRepositories.Get(key); ok {
		metrics.IncGitHubCacheHitCollector("repository", true)
		return repository.(*github.Repository), nil
	}

	githubRepositoriesMutext.Lock()
	defer githubRepositoriesMutext.Unlock()

	if repository, ok := githubRepositories.Get(key); ok {
		metrics.IncGitHubCacheHitCollector("repository", true)
		return repository.(*github.Repository), nil
	}

	metrics.IncGitHubCacheHitCollector("repository", false)

	if githubClient == nil {
		return nil, errors.New("githubClient == nil")
	}

	repository, githubResponse, err := githubClient.Repositories.Get(ctx, owner, name)
	if err := handleGitHubResponse(ctx, githubClient, "entry", githubResponse, err); err != nil {
		return nil, err
	}

	if _, ok := githubOwners[repository.GetOwner().GetLogin()]; !ok {
		return nil, errors.New("not in githubOwners")
	}

	if private := repository.GetPrivate(); (private && githubVisibilities&repoPrivate == 0) || (!private && githubVisibilities&repoPublic == 0) {
		return nil, errors.New("not in githubVisibilities")
	}

	githubRepositories.SetDefault(key, repository)

	return repository, nil
}

type registrationTokenContainer struct {
	Token *github.RegistrationToken
	Rate  uint64
}

func tryGetGitHubRegistrationToken(key string) (*github.RegistrationToken, error) {
	i, ok := githubRegistrationTokens.Get(key)
	if !ok {
		return nil, errors.New("githubRegistrationTokens.Load(key) !ok")
	}

	container, ok := i.(*registrationTokenContainer)
	if !ok {
		return nil, errors.New("i.(*registrationTokenContainer) !ok")
	}

	// TODO: make rate limit dynamic by getting the value from GET /rate_limits
	if rate := atomic.AddUint64(&container.Rate, 1); rate >= 60 || !container.Token.GetExpiresAt().After(time.Now().Add(time.Minute)) {
		return nil, errors.New("rate >= 60 || !container.Token.GetExpiresAt().After(time.Now().Add(time.Minute))")
	}

	return container.Token, nil
}

func getGitHubRegistrationToken(ctx context.Context, repository *github.Repository) (*github.RegistrationToken, error) {
	if repository == nil {
		return nil, errors.New("repository == nil")
	}

	key := fmt.Sprintf("%s/%s", repository.GetOwner().GetLogin(), repository.GetName())

	if registrationToken, err := tryGetGitHubRegistrationToken(key); err == nil {
		metrics.IncGitHubCacheHitCollector("registrationToken", true)
		return registrationToken, nil
	}

	githubRegistrationTokensMutext.Lock()
	defer githubRegistrationTokensMutext.Unlock()

	if registrationToken, err := tryGetGitHubRegistrationToken(key); err == nil {
		metrics.IncGitHubCacheHitCollector("registrationToken", true)
		return registrationToken, nil
	}

	metrics.IncGitHubCacheHitCollector("registrationToken", false)

	if githubClient == nil {
		return nil, errors.New("githubClient == nil")
	}

	registrationToken, githubResponse, err := githubClient.Actions.CreateRegistrationToken(ctx, repository.GetOwner().GetLogin(), repository.GetName())
	if err := handleGitHubResponse(ctx, githubClient, "entry", githubResponse, err); err != nil {
		return nil, err
	}

	githubRegistrationTokens.SetDefault(key, &registrationTokenContainer{
		Token: registrationToken,
	})

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

	if err := collectGitHubRateLimitMetrics(ctx, bridgeClient, "bridge"); err != nil {
		return nil, err
	}

	return bridgeClient, nil
}

type removeTokenContainer struct {
	Token *github.RemoveToken
	Rate  uint64
}

func tryGetGitHubRemoveToken(key string) (*github.RemoveToken, error) {
	i, ok := githubRemoveTokens.Get(key)
	if !ok {
		return nil, errors.New("githubRemoveTokens.Get(key) !ok")
	}

	container, ok := i.(*removeTokenContainer)
	if !ok {
		return nil, errors.New("i.(*removeTokenContainer) !ok")
	}

	// TODO: make rate limit dynamic by getting the value from GET /rate_limits
	if rate := atomic.AddUint64(&container.Rate, 1); rate >= 60 || !container.Token.GetExpiresAt().After(time.Now().Add(time.Minute)) {
		return nil, errors.New("rate >= 60 || !container.Token.GetExpiresAt().After(time.Now().Add(time.Minute))")
	}

	return container.Token, nil
}

func getGitHubRemoveToken(ctx context.Context, repository *github.Repository) (*github.RemoveToken, error) {
	if repository == nil {
		return nil, errors.New("repository == nil")
	}

	key := fmt.Sprintf("%s/%s", repository.GetOwner().GetLogin(), repository.GetName())

	if removeToken, err := tryGetGitHubRemoveToken(key); err == nil {
		metrics.IncGitHubCacheHitCollector("removeToken", true)
		return removeToken, nil
	}

	githubRemoveTokensMutext.Lock()
	defer githubRemoveTokensMutext.Unlock()

	if removeToken, err := tryGetGitHubRemoveToken(key); err == nil {
		metrics.IncGitHubCacheHitCollector("removeToken", true)
		return removeToken, nil
	}

	metrics.IncGitHubCacheHitCollector("removeToken", false)

	if githubClient == nil {
		return nil, errors.New("githubClient == nil")
	}

	removeToken, githubResponse, err := githubClient.Actions.CreateRemoveToken(ctx, repository.GetOwner().GetLogin(), repository.GetName())
	if err := handleGitHubResponse(ctx, githubClient, "entry", githubResponse, err); err != nil {
		return nil, err
	}

	githubRemoveTokens.SetDefault(key, &removeTokenContainer{
		Token: removeToken,
	})

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

	if err := collectGitHubRateLimitMetrics(ctx, bridgeClient, "bridge"); err != nil {
		return nil, err
	}

	return bridgeClient, nil
}

func tryGetGitHubTenantCredential(key string) (*github.TenantCredential, error) {
	i, ok := githubTenantCredentials.Get(key)
	if !ok {
		return nil, errors.New("githubTenantCredentials.Load(key) !ok")
	}

	tenantCredential, ok := i.(*github.TenantCredential)
	if !ok {
		return nil, errors.New("i.(*github.TenantCredential) !ok")
	}

	token, err := jwt.ParseSigned(tenantCredential.GetToken())
	if err != nil {
		return nil, err
	}

	var claims jwt.Claims
	if err := token.UnsafeClaimsWithoutVerification(&claims); err != nil {
		return nil, err
	}

	if !claims.Expiry.Time().After(time.Now().Add(time.Minute)) {
		return nil, errors.New("!claims.Expiry.Time().After(time.Now().Add(time.Minute))")
	}

	return tenantCredential, nil
}

type RunnerEvent string

const (
	RunnerEventRegister RunnerEvent = "register"
	RunnerEventRemove   RunnerEvent = "remove"
)

func GetGitHubTenantCredential(ctx context.Context, repository *github.Repository, runnerEvent RunnerEvent) (*github.TenantCredential, error) {
	if repository == nil {
		return nil, errors.New("repository == nil")
	}

	key := fmt.Sprintf("%s@%s/%s", string(runnerEvent), repository.GetOwner().GetLogin(), repository.GetName())

	if tenantCredential, err := tryGetGitHubTenantCredential(key); err == nil {
		metrics.IncGitHubCacheHitCollector("tenantCredential", true)
		return tenantCredential, nil
	}

	githubTenantCredentialsMutext.Lock()
	defer githubTenantCredentialsMutext.Unlock()

	if tenantCredential, err := tryGetGitHubTenantCredential(key); err == nil {
		metrics.IncGitHubCacheHitCollector("tenantCredential", true)
		return tenantCredential, nil
	}

	metrics.IncGitHubCacheHitCollector("tenantCredential", false)

	var bridgeClient *github.Client
	switch runnerEvent {
	case RunnerEventRegister:
		client, err := newGitHubBridgeClientWithRegistrationToken(ctx, repository)
		if err != nil {
			return nil, err
		}
		bridgeClient = client

	case RunnerEventRemove:
		client, err := newGitHubBridgeClientWithRemoveToken(ctx, repository)
		if err != nil {
			return nil, err
		}
		bridgeClient = client

	default:
		return nil, errors.New("unknown runnerEvent: " + string(runnerEvent))
	}

	tenantCredential, githubResponse, err := bridgeClient.Actions.CreateTenantCredential(ctx, string(runnerEvent), repository.GetHTMLURL())
	if err := handleGitHubResponse(ctx, bridgeClient, "bridge-"+key, githubResponse, err); err != nil {
		return nil, err
	}

	githubTenantCredentials.SetDefault(key, tenantCredential)

	return tenantCredential, nil
}

type GitHub struct {
	Repository *github.Repository
}

func (gh *GitHub) Init(ctx context.Context, repoOwner string, repoName string) error {
	if err := initGitHubClient(ctx); err != nil {
		return err
	}

	repository, err := getGitHubRepository(ctx, repoOwner, repoName)
	if err != nil {
		return err
	}
	gh.Repository = repository

	return nil
}
