package gh

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"

	"golang.org/x/oauth2"

	"github.com/bradleyfalzon/ghinstallation"
	"github.com/google/go-github/v32/github"
	"github.com/whywaita/myshoes/internal/config"
)

// NewClient create a client of GitHub
func NewClient(ctx context.Context, personalToken, gheDomain string) (*github.Client, error) {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{
			AccessToken: personalToken,
		})
	tc := oauth2.NewClient(ctx, ts)

	if gheDomain == "" {
		return github.NewClient(tc), nil
	}

	return github.NewEnterpriseClient(gheDomain, gheDomain, tc)

}

// CheckSignature check trust installation id from event.
func CheckSignature(installationID int64) error {
	appID := config.Config.GitHub.AppID
	pem := config.Config.GitHub.PEM

	tr := http.DefaultTransport
	_, err := ghinstallation.New(tr, appID, installationID, pem)
	if err != nil {
		return fmt.Errorf("failed to create GitHub installation: %w", err)
	}

	return nil
}

// ExistGitHubRepository check exist of github repository
func ExistGitHubRepository(scope, gheDomain string, gheDomainValid bool, githubPersonalToken string) error {
	repoURL, err := getRepositoryURL(scope, gheDomain, gheDomainValid)
	if err != nil {
		return fmt.Errorf("failed to get repository url: %w", err)
	}

	client := http.DefaultClient
	req, err := http.NewRequest(http.MethodGet, repoURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Add("Authorization", fmt.Sprintf("token %s", githubPersonalToken))

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to do request: %w", err)
	}

	if resp.StatusCode == http.StatusOK {
		return nil
	} else if resp.StatusCode == http.StatusNotFound {
		return errors.New("not found")
	}

	return fmt.Errorf("invalid response code (%d)", resp.StatusCode)
}

func getRepositoryURL(scope, gheDomain string, gheDomainValid bool) (string, error) {
	// github.com
	//   => https://api.github.com/repos/:owner/:repo
	//   => https://api.github.com/orgs/:owner
	// GitHub Enterprise Server
	//   => https://{your_ghe_server_url}/api/repos/:owner/:repo
	//   => https://{your_ghe_server_url}/api/orgs/:owner

	s := DetectScope(scope)
	if s == Unknown {
		return "", fmt.Errorf("failed to detect valid scope")
	}

	var apiEndpoint *url.URL
	if gheDomainValid {
		u, err := url.Parse(gheDomain)
		if err != nil {
			return "", fmt.Errorf("failed to parse GHE url: %w", err)
		}

		u.Path = path.Join(u.Path, "api")
		apiEndpoint = u
	} else {
		u, err := url.Parse("https://api.github.com")
		if err != nil {
			return "", fmt.Errorf("failed to parse github.com: %w", err)
		}
		apiEndpoint = u
	}

	p := path.Join(apiEndpoint.Path, s.String(), scope)
	apiEndpoint.Path = p

	return apiEndpoint.String(), nil
}

// Scope is scope for auto-scaling target
type Scope int

// Scope values
const (
	Unknown Scope = iota
	Repository
	Organization
)

// String is fmt.Stringer interface
func (s Scope) String() string {
	switch s {
	case Repository:
		return "repos"
	case Organization:
		return "orgs"
	default:
		return "unknown"
	}
}

// DetectScope detect a scope (repo or org)
func DetectScope(scope string) Scope {
	sep := strings.Split(scope, "/")
	switch len(sep) {
	case 1:
		return Organization
	case 2:
		return Repository
	default:
		return Unknown
	}
}