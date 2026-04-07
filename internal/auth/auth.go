// Package auth handles Azure AD / Entra ID authentication for Dynamics 365.
// It uses the official Microsoft Authentication Library (MSAL) for Go:
// https://github.com/AzureAD/microsoft-authentication-library-for-go
//
// Supported flows:
//   - interactive         — ブラウザを直接開いてサインイン（Intune管理デバイス推奨）
//   - device_code         — コードをブラウザに入力してサインイン
//   - password (ROPC)     — ユーザー名/パスワード直接入力（MFAなし環境向け）
//   - client_credentials  — サービスアカウント（アプリ登録が必要）
package auth

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/confidential"
	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/public"

	"github.com/shuyahonda/crm-cli/internal/config"
)

// Provider acquires and caches OAuth2 tokens via MSAL.
type Provider struct {
	cfg   *config.Config
	cache *fileCache
}

// NewProvider creates a new MSAL-backed auth provider.
func NewProvider(cfg *config.Config) *Provider {
	return &Provider{
		cfg:   cfg,
		cache: newFileCache(),
	}
}

// GetToken returns a valid access token for Dynamics 365.
// Tokens are cached on disk; silent acquisition is attempted first.
func (p *Provider) GetToken(ctx context.Context) (string, error) {
	switch p.cfg.AuthFlow {
	case "interactive":
		return p.tokenViaInteractive(ctx)
	case "device_code":
		return p.tokenViaDeviceCode(ctx)
	case "password":
		return p.tokenViaPassword(ctx)
	case "client_credentials":
		return p.tokenViaClientCredentials(ctx)
	default:
		return "", fmt.Errorf("unsupported auth_flow: %q", p.cfg.AuthFlow)
	}
}

// scopes returns the Dynamics 365 delegated permission scope.
func (p *Provider) scopes() []string {
	base := strings.TrimRight(p.cfg.CRMBaseURL, "/")
	return []string{base + "/user_impersonation"}
}

// authority returns the Azure AD authority URL for the configured tenant.
func (p *Provider) authority() string {
	return "https://login.microsoftonline.com/" + p.cfg.TenantID
}

// newPublicClient creates a MSAL public client with file cache.
func (p *Provider) newPublicClient() (public.Client, error) {
	return public.New(p.cfg.ClientID,
		public.WithAuthority(p.authority()),
		public.WithCache(p.cache),
	)
}

// tokenViaInteractive acquires a token by opening the system browser directly.
//
// Intune管理デバイスでの推奨フロー。
// Edge などの管理対象ブラウザが Windows WAM (Web Account Manager) 経由で
// デバイスの PRT (Primary Refresh Token) を使用するため、
// デバイスコンプライアンスを要求する Conditional Access ポリシーを通過できます。
//
// ブラウザの起動は MSAL 内蔵の pkg/browser に委譲します。
// Windows では rundll32 url.dll,FileProtocolHandler を使用するため、
// OAuth2 URL に含まれる "&" が cmd によってコマンド区切りとして解釈される問題を回避します。
func (p *Provider) tokenViaInteractive(ctx context.Context) (string, error) {
	app, err := p.newPublicClient()
	if err != nil {
		return "", fmt.Errorf("MSAL: failed to create public client: %w", err)
	}

	scopes := p.scopes()

	// サイレント取得を先に試みる（キャッシュ済みトークンの再利用）
	accounts, err := app.Accounts(ctx)
	if err == nil && len(accounts) > 0 {
		result, err := app.AcquireTokenSilent(ctx, scopes,
			public.WithSilentAccount(accounts[0]))
		if err == nil {
			return result.AccessToken, nil
		}
	}

	// ブラウザを開いてインタラクティブ認証
	// WithOpenURL を使わず MSAL 内蔵のブラウザ起動ロジックに任せる
	result, err := app.AcquireTokenInteractive(ctx, scopes)
	if err != nil {
		return "", fmt.Errorf("MSAL: interactive authentication failed: %w", err)
	}

	return result.AccessToken, nil
}

// tokenViaDeviceCode acquires a token using the device code flow.
// Uses a file cache so the user only authenticates once (refresh token lasts ~90 days).
func (p *Provider) tokenViaDeviceCode(ctx context.Context) (string, error) {
	app, err := p.newPublicClient()
	if err != nil {
		return "", fmt.Errorf("MSAL: failed to create public client: %w", err)
	}

	scopes := p.scopes()

	// サイレント取得を先に試みる
	accounts, err := app.Accounts(ctx)
	if err == nil && len(accounts) > 0 {
		result, err := app.AcquireTokenSilent(ctx, scopes,
			public.WithSilentAccount(accounts[0]))
		if err == nil {
			return result.AccessToken, nil
		}
	}

	// デバイスコードフロー（コードを表示してブラウザで入力）
	dc, err := app.AcquireTokenByDeviceCode(ctx, scopes)
	if err != nil {
		return "", fmt.Errorf("MSAL: device code initiation failed: %w", err)
	}

	// ユーザーへの案内メッセージを表示
	fmt.Fprintf(os.Stderr, "\n%s\n\n", dc.Result.Message)

	result, err := dc.AuthenticationResult(ctx)
	if err != nil {
		return "", fmt.Errorf("MSAL: device code authentication failed: %w", err)
	}

	return result.AccessToken, nil
}

// tokenViaPassword acquires a token using username/password (ROPC).
// Does not work when MFA is enforced on the account.
func (p *Provider) tokenViaPassword(ctx context.Context) (string, error) {
	app, err := p.newPublicClient()
	if err != nil {
		return "", fmt.Errorf("MSAL: failed to create public client: %w", err)
	}

	scopes := p.scopes()

	// サイレント取得を先に試みる
	accounts, err := app.Accounts(ctx)
	if err == nil {
		for _, account := range accounts {
			if strings.EqualFold(account.PreferredUsername, p.cfg.Username) {
				result, err := app.AcquireTokenSilent(ctx, scopes,
					public.WithSilentAccount(account))
				if err == nil {
					return result.AccessToken, nil
				}
				break
			}
		}
	}

	result, err := app.AcquireTokenByUsernamePassword(ctx, scopes,
		p.cfg.Username, p.cfg.Password)
	if err != nil {
		return "", fmt.Errorf("MSAL: password authentication failed: %w\n"+
			"  Hint: if MFA is enforced on your account, use auth_flow: interactive", err)
	}

	return result.AccessToken, nil
}

// tokenViaClientCredentials acquires a token using the client credentials flow.
// Requires an Azure AD app registration with a client secret.
func (p *Provider) tokenViaClientCredentials(ctx context.Context) (string, error) {
	cred, err := confidential.NewCredFromSecret(p.cfg.ClientSecret)
	if err != nil {
		return "", fmt.Errorf("MSAL: invalid client secret: %w", err)
	}

	app, err := confidential.New(p.authority(), p.cfg.ClientID, cred,
		confidential.WithCache(p.cache),
	)
	if err != nil {
		return "", fmt.Errorf("MSAL: failed to create confidential client: %w", err)
	}

	// client_credentials スコープは /.default を使用
	base := strings.TrimRight(p.cfg.CRMBaseURL, "/")
	scopes := []string{base + "/.default"}

	result, err := app.AcquireTokenSilent(ctx, scopes)
	if err != nil {
		result, err = app.AcquireTokenByCredential(ctx, scopes)
		if err != nil {
			return "", fmt.Errorf("MSAL: client credentials token acquisition failed: %w", err)
		}
	}

	return result.AccessToken, nil
}

