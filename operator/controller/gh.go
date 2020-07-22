package controller

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

type facadeGitHub struct {
	context context.Context

	ghClient     *github.Client
	ghRepository *github.Repository

	ghBridgeClient *github.Client
}

func (gh *facadeGitHub) Init(token string, owner string, name string) error {
	if err := gh.initGitHubClient(token); err != nil {
		return err
	}

	if err := gh.initGitHubRepository(owner, name); err != nil {
		return err
	}

	if err := gh.initGitHubBridgeClient(); err != nil {
		return err
	}

	return nil
}

func (gh *facadeGitHub) initGitHubClient(token string) error {
	if token == "" {
		return errors.New(`token == ""`)
	}

	gh.ghClient = github.NewClient(
		oauth2.NewClient(
			gh.context,
			oauth2.StaticTokenSource(
				&oauth2.Token{
					AccessToken: token,
				},
			),
		),
	)
	return nil
}

func (gh *facadeGitHub) initGitHubRepository(owner string, name string) error {
	if gh.context == nil {
		return errors.New(".context == nil")
	}

	if gh.ghClient == nil {
		return errors.New(".ghClient == nil")
	}

	repository, githubResponse, err := gh.ghClient.Repositories.Get(gh.context, owner, name)
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

	gh.ghRepository = repository
	return nil
}

func (gh *facadeGitHub) GetGitHubRegistrationToken() (*github.RegistrationToken, error) {
	if gh.context == nil {
		return nil, errors.New(".context == nil")
	}

	if gh.ghClient == nil {
		return nil, errors.New(".ghClient == nil")
	}

	token, githubResponse, err := gh.ghClient.Actions.CreateRegistrationToken(gh.context, gh.ghRepository.GetOwner().GetLogin(), gh.ghRepository.GetName())
	if err != nil {
		return nil, err
	}
	if githubResponse.StatusCode < 200 || githubResponse.StatusCode >= 300 {
		return nil, errors.New(githubResponse.Status)
	}

	return token, nil
}

func (gh *facadeGitHub) GetGitHubRemoveToken() (*github.RemoveToken, error) {
	if gh.context == nil {
		return nil, errors.New(".context == nil")
	}

	if gh.ghClient == nil {
		return nil, errors.New(".ghClient == nil")
	}

	token, githubResponse, err := gh.ghClient.Actions.CreateRemoveToken(gh.context, gh.ghRepository.GetOwner().GetLogin(), gh.ghRepository.GetName())
	if err != nil {
		return nil, err
	}
	if githubResponse.StatusCode < 200 || githubResponse.StatusCode >= 300 {
		return nil, errors.New(githubResponse.Status)
	}

	return token, nil
}

func (gh *facadeGitHub) initGitHubBridgeClient() error {
	token, err := gh.GetGitHubRegistrationToken()
	if err != nil {
		return err
	}

	gh.ghBridgeClient = github.NewClient(
		oauth2.NewClient(
			gh.context,
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

func (gh *facadeGitHub) GetGitHubTenantCredential(runnerEvent RunnerEvent) (*github.TenantCredential, error) {
	if gh.context == nil {
		return nil, errors.New(".context == nil")
	}

	if gh.ghBridgeClient == nil {
		return nil, errors.New(".ghBridgeClient == nil")
	}

	tenantCredential, githubResponse, err := gh.ghBridgeClient.Actions.CreateTenantCredential(gh.context, string(runnerEvent), gh.ghRepository.GetHTMLURL())
	if err != nil {
		return nil, err
	}
	if githubResponse.StatusCode < 200 || githubResponse.StatusCode >= 300 {
		return nil, errors.New(githubResponse.Status)
	}

	return tenantCredential, nil
}
