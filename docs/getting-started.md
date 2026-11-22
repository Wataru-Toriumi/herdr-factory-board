# ローカルで起動する

macOS または Linux で、Herdr 0.7.0 以降と Go 1.21 以降を使用します。Go が必要なのはビルド時だけです。
作業を実行するには、Herdr から起動したペインで `codex exec` を使えるようにし、対象の Git リポジトリで Codex CLI の認証と設定を済ませてください。ファイル編集を伴う作業には、Codex 側でその作業先への書き込みを許可します。
Herdr のセッション内で、このプラグインのリポジトリのルートから実行します。

```sh
go build -o factory-board ./src
herdr plugin link "$PWD"
herdr plugin pane open --plugin wataru-toriumi.factory-board --entrypoint board
```

起動したら、ボードで作業先を選んで最初のカードを作成します。[ボードの操作](board.md)

## プラグインの登録を解除する

登録を解除するには `herdr plugin unlink wataru-toriumi.factory-board` を実行してください。[Herdr 公式プラグインガイド](https://herdr.dev/tui/docs/plugins/)
