package lib

import (
	"errors"
	"fmt"
	"io"
	"text/template"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sso"
	"github.com/aws/aws-sdk-go/service/ssooidc"
	"github.com/gosimple/slug"
)

// Configurator handles the fetching and generating of the SSO Configuration
type Configurator struct {
	Session  *session.Session
	Client   *ssooidc.RegisterClientOutput
	StartURL *string
	Device   *ssooidc.StartDeviceAuthorizationOutput
	Token    *ssooidc.CreateTokenOutput

	Roles []RoleInfo
}

// RoleInfo describes a role
type RoleInfo struct {
	ProfileName string
	RoleName    string
	AccountID   string
	AccountName string
}

// RegisterClient prepares the application for device authorization
func (c *Configurator) RegisterClient(name string) error {
	req := &ssooidc.RegisterClientInput{}
	req.SetClientName(name)
	req.SetClientType("public")
	req.SetScopes(list("openid", "sso-portal:*"))

	res, err := ssooidc.New(c.Session).RegisterClient(req)
	if err != nil {
		return err
	}

	c.Client = res
	return nil
}

// StartDeviceAuthorization begins the authorization flow
func (c *Configurator) StartDeviceAuthorization(startURL string) error {
	c.StartURL = &startURL

	req := &ssooidc.StartDeviceAuthorizationInput{}
	req.SetClientId(*c.Client.ClientId)
	req.SetClientSecret(*c.Client.ClientSecret)
	req.SetStartUrl(startURL)

	res, err := ssooidc.New(c.Session).StartDeviceAuthorization(req)
	if err != nil {
		return err
	}

	c.Device = res
	return nil
}

// WaitForToken polls for a token to be authenticated
func (c *Configurator) WaitForToken(freq, to time.Duration) <-chan (error) {
	ticker := time.NewTicker(freq)
	timeout := time.NewTimer(to)
	done := make(chan error)

	go func() {
		for {
			select {
			case <-timeout.C:
				done <- errors.New("Timeout waiting for new token")
				return
			case <-ticker.C:
				if err := c.CreateToken(); err == nil {
					done <- nil
					return
				}
			}
		}
	}()

	return done
}

// CreateToken attempts to create a token for the device
func (c *Configurator) CreateToken() error {
	req := &ssooidc.CreateTokenInput{}
	req.SetClientId(*c.Client.ClientId)
	req.SetClientSecret(*c.Client.ClientSecret)
	req.SetDeviceCode(*c.Device.DeviceCode)
	req.SetGrantType("urn:ietf:params:oauth:grant-type:device_code")
	req.SetScope(list("openid", "sso-portal:*"))

	res, err := ssooidc.New(c.Session).CreateToken(req)
	if err != nil {
		return err
	}

	c.Token = res
	return nil
}

// LoadRoles loads the accounts and roles the user has access to
func (c *Configurator) LoadRoles() error {
	api := sso.New(c.Session)

	return api.ListAccountsPages(&sso.ListAccountsInput{
		AccessToken: c.Token.AccessToken,
	}, func(out *sso.ListAccountsOutput, lastPage bool) bool {
		for _, acct := range out.AccountList {
			if err := api.ListAccountRolesPages(&sso.ListAccountRolesInput{
				AccessToken: c.Token.AccessToken,
				AccountId:   acct.AccountId,
			}, func(out *sso.ListAccountRolesOutput, lastPage bool) bool {

				for _, role := range out.RoleList {
					c.Roles = append(c.Roles, RoleInfo{
						ProfileName: slug.Make(fmt.Sprintf("%s-%s", *acct.AccountName, *role.RoleName)),
						RoleName:    *role.RoleName,
						AccountID:   *acct.AccountId,
						AccountName: *acct.AccountName,
					})
				}

				return !lastPage
			}); err != nil {
				return false
			}
		}
		return !lastPage
	})
}

// WriteConfig produces a configuration
func (c *Configurator) WriteConfig(out io.Writer) {
	t, _ := template.New("config").Parse(`
{{- $startURL := .StartURL }}
{{- $region := .Region }}

{{- range .Roles }}
[profile {{ .ProfileName }}]
sso_start_url={{ $startURL }}
sso_region={{ $region }}
sso_account_id={{ .AccountID }}
sso_role_name={{ .RoleName }}
region={{ $region }}
{{ end }}`)

	t.Execute(out, map[string]interface{}{
		"Roles":    c.Roles,
		"Region":   c.Session.Config.Region,
		"StartURL": c.StartURL,
	})
}

func list(items ...string) (r []*string) {
	for _, item := range items {
		r = append(r, &item)
	}
	return
}
