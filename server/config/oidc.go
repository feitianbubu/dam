package config

type OIDC struct {
	Enabled          bool   `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
	Provider         string `mapstructure:"provider" json:"provider" yaml:"provider"`                 // oidc provider name
	ClientID         string `mapstructure:"client-id" json:"clientId" yaml:"client-id"`                 // client id
	ClientSecret     string `mapstructure:"client-secret" json:"clientSecret" yaml:"client-secret"`     // client secret
	RedirectURL      string `mapstructure:"redirect-url" json:"redirectUrl" yaml:"redirect-url"`       // redirect url
	Scopes           string `mapstructure:"scopes" json:"scopes" yaml:"scopes"`                         // scopes
	Issuer           string `mapstructure:"issuer" json:"issuer" yaml:"issuer"`                         // issuer url
	AuthURL          string `mapstructure:"auth-url" json:"authUrl" yaml:"auth-url"`                    // auth url
	TokenURL         string `mapstructure:"token-url" json:"tokenUrl" yaml:"token-url"`                 // token url
	UserInfoURL      string `mapstructure:"userinfo-url" json:"userInfoUrl" yaml:"userinfo-url"`       // user info url
	EndSessionURL    string `mapstructure:"end-session-url" json:"endSessionUrl" yaml:"end-session-url"` // end session url
	AutoCreateUser   bool   `mapstructure:"auto-create-user" json:"autoCreateUser" yaml:"auto-create-user"` // auto create user
	DefaultAuthority uint   `mapstructure:"default-authority" json:"defaultAuthority" yaml:"default-authority"` // default authority id for new user
	// Clinx-specific configuration
	DiscoveryURL     string `mapstructure:"discovery-url" json:"discoveryUrl" yaml:"discovery-url"`     // discovery endpoint for auto-configuration
}