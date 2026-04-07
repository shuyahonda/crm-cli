# crm-cli

Dynamics 365 CRM のマイルストーンを管理する CLI ツールです。  
GitHub Copilot から MCP (Model Context Protocol) 経由で呼び出すこともできます。

## 特徴

- **アプリ登録不要** — Microsoft の公開クライアントを使った Device Code / パスワード認証に対応
- **トークンキャッシュ** — MSAL によりトークンをファイルに保存。初回ログイン後は約 90 日間再認証不要
- **GitHub Copilot 連携** — MCP サーバーモードで Copilot からマイルストーンを自然言語操作
- **Windows ARM64 対応** — クロスコンパイル済みバイナリを提供

## インストール

### ビルド済みバイナリ（Windows ARM64）

```powershell
# Releases ページからダウンロードして PATH の通った場所に配置
# 例: C:\Users\<yourname>\bin\crm-cli.exe
```

### ソースからビルド（Go が必要）

PowerShell で直接ビルドできます（`make` は不要）。

```powershell
# Windows ARM64（メインターゲット）
$env:GOOS = "windows"; $env:GOARCH = "arm64"
go build -o crm-cli.exe .

# Windows AMD64
$env:GOOS = "windows"; $env:GOARCH = "amd64"
go build -o crm-cli.exe .
```

`make` が使える環境（WSL / Git Bash）では `Makefile` も利用できます。

```bash
make build-windows-arm64   # → crm-cli-windows-arm64.exe
```

Go 1.21 以上が必要です。[golang.org/dl](https://golang.org/dl/) からインストールしてください。

## 設定

### 方法 1: 設定ファイル（推奨・永続設定）

以下のコマンドで設定ファイルを作成します（毎回環境変数を設定する必要がなくなります）。

```powershell
mkdir "$env:USERPROFILE\.config\crm-cli"
@"
crm_base_url: "https://yourorg.crm.dynamics.com"
auth_flow: "device_code"
"@ | Out-File "$env:USERPROFILE\.config\crm-cli\crm-cli.yaml" -Encoding UTF8
```

詳細な設定例は `crm-cli.yaml.example` を参照してください。

配置場所の優先順位:
1. カレントディレクトリの `crm-cli.yaml`
2. `%USERPROFILE%\.config\crm-cli\crm-cli.yaml`

### 方法 2: 環境変数

PowerShell と cmd.exe では構文が異なります。

```powershell
# PowerShell
$env:CRM_BASE_URL  = "https://yourorg.crm.dynamics.com"
$env:CRM_AUTH_FLOW = "device_code"
```

```cmd
:: コマンドプロンプト (cmd.exe)
set CRM_BASE_URL=https://yourorg.crm.dynamics.com
set CRM_AUTH_FLOW=device_code
```

> 環境変数はターミナルを閉じるとリセットされます。永続化するには設定ファイルを使ってください。

## 認証方式

### Device Code（推奨・アプリ登録不要）

ブラウザで Microsoft アカウントにサインイン。初回のみ操作が必要で、以降はトークンキャッシュが使われます。

```yaml
crm_base_url: "https://yourorg.crm.dynamics.com"
auth_flow: "device_code"
# tenant_id は省略可能（省略時は "common" を使用）
```

トークンキャッシュの保存先: `%LOCALAPPDATA%\crm-cli\token_cache.json`

### Password / ROPC（アプリ登録不要・MFA なし環境向け）

MFA が有効なアカウントでは使用できません。

```yaml
crm_base_url: "https://yourorg.crm.dynamics.com"
auth_flow: "password"
username:  "yourname@yourorg.onmicrosoft.com"
password:  "your-password"
```

### Client Credentials（管理者によるアプリ登録が必要）

サービス間通信向け。Azure AD へのアプリ登録が必要です。

```yaml
crm_base_url:   "https://yourorg.crm.dynamics.com"
auth_flow:      "client_credentials"
tenant_id:      "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
client_id:      "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
client_secret:  "your-client-secret"
```

## CLI コマンド

```
crm-cli milestone <subcommand>
```

| サブコマンド | 説明 |
|---|---|
| `list` | マイルストーン一覧表示 |
| `show <id>` | 詳細表示 |
| `create` | 新規作成 |
| `update <id>` | 更新 |
| `close <id>` | クローズ（非アクティブ化） |
| `reopen <id>` | 再オープン |
| `delete <id>` | 削除 |

### 使用例

```powershell
# 一覧表示
crm-cli milestone list

# プロジェクトで絞り込み
crm-cli milestone list --project 00000000-0000-0000-0000-000000000000

# 作成
crm-cli milestone create --name "Phase 1 Complete" --date 2026-06-30

# 詳細確認
crm-cli milestone show 00000000-0000-0000-0000-000000000000

# 名前と日付を更新
crm-cli milestone update 00000000-0000-0000-0000-000000000000 --name "Go-Live" --date 2026-12-01

# クローズ
crm-cli milestone close 00000000-0000-0000-0000-000000000000

# 再オープン
crm-cli milestone reopen 00000000-0000-0000-0000-000000000000

# 削除（確認プロンプトあり）
crm-cli milestone delete 00000000-0000-0000-0000-000000000000
crm-cli milestone delete 00000000-0000-0000-0000-000000000000 --force
```

## GitHub Copilot 連携（MCP サーバー）

`crm-cli mcp` で MCP サーバーとして起動し、GitHub Copilot からマイルストーン操作を自然言語で行えます。

### VS Code への設定

`.github/copilot-mcp.json` の内容を VS Code の `settings.json` に追加します。

```json
{
  "github.copilot.chat.mcp.servers": {
    "crm-milestone": {
      "command": "crm-cli",
      "args": ["mcp"],
      "env": {
        "CRM_BASE_URL": "https://yourorg.crm.dynamics.com",
        "CRM_AUTH_FLOW": "device_code"
      }
    }
  }
}
```

### 利用できるツール

| ツール名 | 説明 |
|---|---|
| `milestone_list` | 一覧取得（プロジェクト絞り込み可） |
| `milestone_get` | ID 指定で詳細取得 |
| `milestone_create` | 新規作成 |
| `milestone_update` | 名前・説明・日付の更新 |
| `milestone_close` | クローズ |
| `milestone_reopen` | 再オープン |
| `milestone_delete` | 削除 |

### Copilot への指示例

```
@copilot CRM のマイルストーン一覧を見せて
@copilot "Phase 2 完了" というマイルストーンを 2026-09-30 で作成して
@copilot ID が xxx のマイルストーンをクローズして
```

## プロジェクト構成

```
crm-cli/
├── main.go
├── Makefile
├── cmd/
│   ├── root.go
│   ├── milestone/          # CLI コマンド群
│   └── mcp/serve.go        # MCP サーバー起動
└── internal/
    ├── auth/
    │   ├── auth.go         # MSAL 認証（device_code / password / client_credentials）
    │   └── cache.go        # ファイルベーストークンキャッシュ
    ├── config/config.go    # 設定管理
    ├── crm/
    │   ├── client.go       # Dynamics 365 Web API クライアント
    │   └── milestone.go    # msdyn_milestone エンティティ操作
    └── mcp/
        ├── protocol.go     # JSON-RPC 2.0 / MCP 型定義
        ├── tools.go        # ツール定義・ハンドラー
        └── server.go       # MCP サーバーループ
```

## 技術スタック

| 用途 | ライブラリ |
|---|---|
| CLI フレームワーク | [cobra](https://github.com/spf13/cobra) |
| 設定管理 | [viper](https://github.com/spf13/viper) |
| 認証 | [MSAL for Go](https://github.com/AzureAD/microsoft-authentication-library-for-go) |
| API | Dynamics 365 Web API (OData v4) |
| MCP | JSON-RPC 2.0 over stdin/stdout |
