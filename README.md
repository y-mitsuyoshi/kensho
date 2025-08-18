# Kensho

[](https://www.google.com/search?q=https://goreportcard.com/report/github.com/your-username/Kensho)
[](https://www.google.com/search?q=https://pkg.go.dev/badge/license-MIT)

`Kensho` は、Googleの最新AIモデルである **Gemini 2.5 Pro** を利用して、運転免許証やマイナンバーカードなどの本人確認書類から情報を高精度で抽出し、JSON形式で返すGo言語製のOCRサービスです。「券面情報」の読み取りと「検証」をコンセプトにしています。

## 📜 概要

このプロジェクトは、画像ファイルとして受け取った本人確認書類の券面をGeminiの強力なマルチモーダル機能で解析し、氏名、住所、生年月日、各種番号などの情報を構造化されたデータとして提供することを目的としています。

単純な文字起こしに留まらず、Geminiの理解能力を活かして、それぞれの情報が「何を意味するのか」を判断し、キーとバリューが整ったJSONを生成します。

## ✨ 特徴

  * **高精度な情報抽出**: Gemini 2.5 Proモデルの活用により、傾きや光の反射がある画像からでも正確に情報を抽出します。
  * **JSON形式での出力**: 抽出した情報は、以下のように構造化されたJSON形式で返されるため、後続のシステムで容易に扱えます。
  * **主要な本人確認書類に対応**: 主要な本人確認書類情報抽出に最適化されています。
  * **Go言語によるシンプルな実装**: Go言語の標準ライブラリとGoogle AI Go SDKのみを使用しており、軽量かつ高速に動作します。

## 💻 技術スタック

  * **言語**: Go
  * **AIモデル**: Google Gemini 2.5 Pro
  * **主要ライブラリ**: [Google AI Go SDK](https://github.com/google/generative-ai-go)

## 🚀 使い方

### 1. APIキーの設定

プロジェクトのルートディレクトリに `.env.example` ファイルが用意されています。これをコピーして `.env` ファイルを作成してください。

```bash
cp .env.example .env
```

次に、作成した `.env` ファイルをテキストエディタで開き、`GEMINI_API_KEY` にご自身のGoogle AI StudioのAPIキーを設定してください。

```dotenv
# .env
PORT=8080
GEMINI_API_KEY="YOUR_API_KEY_HERE"
```

### 2. サービスの起動

Dockerがインストールされている環境で、以下のコマンドを実行します。

```bash
docker-compose up --build
```

初回はイメージのビルドが行われ、完了すると `listening on :8080` というログが表示され、APIサーバーが起動します。

### 3. OCRの実行

サーバーが起動したら、別のターミナルから以下の `curl` コマンドを実行して、本人確認書類の画像ファイルを送信します。

- `image.png` の部分は、実際の画像ファイルへのパスに置き換えてください。
- 対応している画像形式は `image/png`, `image/jpeg` などです。

```bash
curl -X POST http://localhost:8080/api/v1/extract \
  -F "image=@/path/to/your/image.png;type=image/png"
```

成功すると、以下のようなJSONレスポンスが返ってきます。

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

## 📜 ライセンス

このプロジェクトは **MIT License** のもとで公開されています。

-----