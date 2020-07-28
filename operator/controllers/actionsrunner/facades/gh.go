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
	"errors"
	"log"
	"os"
	"strings"

	"github.com/google/go-github/v32/github"
	"golang.org/x/oauth2"
)

type repositoryVisibility uint8

const (
	repoPublic repositoryVisibility = 1 << iota
	repoPrivate
)

var (
	githubPAT = os.Getenv("KUBEACTIONS_GITHUB_PAT")

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
	Client     *github.Client
	Repository *github.Repository

	BridgeClient *github.Client
}

func (gh *GitHub) Init(ctx context.Context, repoOwner string, repoName string) error {
	if err := gh.initGitHubClient(ctx, githubPAT); err != nil {
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

func (gh *GitHub) initGitHubClient(ctx context.Context, token string) error {
	if token == "" {
		return errors.New(`token == ""`)
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
