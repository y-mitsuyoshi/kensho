# Kensho Go ライブラリ

[![Go Reference](https://pkg.go.dev/badge/github.com/y-mitsuyoshi/kensho.svg)](https://pkg.go.dev/github.com/y-mitsuyoshi/kensho)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

`Kensho`は、Googleの**Gemini 2.5 Pro**モデルを使用して、運転免許証やマイナンバーカードなどの本人確認書類から情報を高精度に抽出し、JSONオブジェクトとして返すGoライブラリです。「見証」という言葉にインスパイアされています。

## ✨ 特徴

- **高精度な情報抽出**: Gemini 2.5 Proモデルを活用し、傾きや光の反射がある画像からでも正確に情報を抽出します。
- **構造化されたJSON出力**: 構造化されたJSONを返すため、他のシステムとの連携が容易です。
- **日本の本人確認書類に最適化**: 日本の主要な本人確認書類に特化してファインチューニングされています。
- **高度な画像前処理**: 傾き補正、コントラスト調整、ノイズ除去などの画像前処理機能を内蔵し、OCRの精度を向上させます。
- **シンプルなGo実装**: 標準ライブラリとGoogle AI Go SDKのみで構築されており、軽量かつ高速に動作します。

## 💻 技術スタック

- **言語**: Go
- **AIモデル**: Google Gemini 2.5 Pro
- **主要ライブラリ**: [Google AI Go SDK](https://github.com/google/generative-ai-go)

## 🚀 インストール

プロジェクトにKenshoを追加するには、`go get`を使用します。

```bash
go get -u github.com/y-mitsuyoshi/kensho
```

## 使い方

Kenshoクライアントの基本的な使い方です。

まず、Gemini APIキーを環境変数に設定してください。

```bash
export GEMINI_API_KEY="YOUR_API_KEY_HERE"
```

その後、Goアプリケーションでクライアントを使用します。

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/y-mitsuyoshi/kensho/kensho"
)

func main() {
	ctx := context.Background()
	apiKey := os.Getenv("GEMINI_API_KEY")

	// デフォルトの埋め込み設定で新しいクライアントを作成
	client, err := kensho.NewClient(ctx, apiKey)
	if err != nil {
		log.Fatalf("Failed to create kensho client: %v", err)
	}
	defer client.Close()

	// 画像ファイルを読み込む
	// 実際のアプリケーションでは、HTTPリクエストなどから取得することが想定されます。
	frontImage, err := os.ReadFile("/path/to/your/image.jpg")
	if err != nil {
		log.Fatalf("Failed to read image file: %v", err)
	}

	// API呼び出しのためにファイルパーツを準備
	fileParts := map[string]kensho.FilePart{
		"front": {
			Content:  frontImage,
			MimeType: "image/jpeg",
		},
	}

	// 抽出したい書類の種類を指定
	docType := "driver_license" // または "individual_number_card"

	// 抽出メソッドを呼び出す
	data, err := client.Extract(ctx, fileParts, docType)
	if err != nil {
		log.Fatalf("Failed to extract data: %v", err)
	}

	// 結果は map[string]interface{}
	// 表示用にJSON文字列にマーシャリング
	prettyJSON, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal JSON: %v", err)
	}

	fmt.Println(string(prettyJSON))
}
```

## 🌐 例: Webサービスとして実行する

このリポジトリには、KenshoライブラリをHTTP API経由で公開するサンプルWebサーバーも含まれています。

### 1. APIキーを設定する

まず、`.env.example`ファイルを`.env`にコピーします。

```bash
cp .env.example .env
```

次に、`.env`を開き、`GEMINI_API_KEY`を追加します。

```dotenv
# .env
PORT=8080
GEMINI_API_KEY="YOUR_API_KEY_HERE"
```

### 2. サービスを実行する

提供されている`Makefile`を使用してサービスを管理できます。

#### サーバーを起動する

このコマンドはDockerコンテナをビルドし、バックグラウンドで起動します。

```bash
make up
```

#### ログを確認する

```bash
make logs
```

`listening on :8080`と表示されれば、サーバーは準備完了です。

#### OCRリクエストを送信する

別のターミナルから`curl`を使用して本人確認書類の画像を送信します。

- `/path/to/your/image.png`を実際のファイルパスに置き換えてください。
- サーバーは`image/png`、`image/jpeg`、`image/webp`をサポートしています。
- 運転免許証（`driver_license`）の場合、`image_front`と`image_back`を送信できます。
- マイナンバーカード（`individual_number_card`）の場合、`image_front`を送信します。

```bash
curl -X POST http://localhost:8080/api/v1/extract \
  -F "document_type=driver_license" \
  -F "image_front=@/path/to/your/image.png"
```

リクエストが成功すると、次のようなJSONレスポンスが返されます。

```json
{
  "address": "東京都千代田区霞が関2-1-1",
  "birth_date": "昭和60年1月1日",
  "card_number": "第123456789012号",
  "expiry_date": "平成30年2月1日",
  "issue_date": "平成25年4月1日",
  "name": "見本太郎"
}
```

### 3. その他の `make` コマンド

| コマンド | 説明 |
|---|---|
| `make up` | コンテナをビルドしてバックグラウンドで起動します。 |
| `make down` | コンテナと関連ネットワークを停止・削除します。 |
| `make stop` | コンテナを停止します。 |
| `make logs` | 実行中のコンテナのログを表示します。 |
| `make shell` | 実行中の`api`サービスコンテナ内でシェルを起動します。 |
| `make build` | Dockerイメージをビルドします。 |

## 📜 ライセンス

このプロジェクトは**MITライセンス**のもとで公開されています。
