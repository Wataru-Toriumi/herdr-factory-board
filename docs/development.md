# 開発

開発用の Go は [mise.toml](../mise.toml) で指定しています。シェルで mise を有効化している場合は、初回にリポジトリのルートで次のコマンドを実行します。

```sh
mise trust
mise install
go build -o factory-board ./src
```

プラグインのメタデータとビルド・起動コマンドは [herdr-plugin.toml](../herdr-plugin.toml)、ボードの画面は [src/board_ui.go](../src/board_ui.go)、コマンド処理は [src/main.go](../src/main.go)、定期実行は [src/scheduler.go](../src/scheduler.go)、カードの実行と復旧は [src/worker.go](../src/worker.go)、Herdr CLI との連携は [src/herdr.go](../src/herdr.go) にあります。
コードを変更したら、`go build -o factory-board ./src` を再実行し、ペインを開き直してください。
