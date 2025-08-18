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





## 📜 ライセンス

このプロジェクトは **MIT License** のもとで公開されています。

-----