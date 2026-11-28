# ローカルで起動する

macOS または Linux で、Herdr 0.7.0 以降と Go 1.21 以降を使用します。macOS では音声入力ヘルパーのビルドに Xcode Command Line Tools（`xcode-select --install`）も使います。Go と Swift コンパイラが必要なのはビルド時だけです。
Herdr のペインで、使いたいエージェントの CLI を起動できるようにします。

- **Codex（初期設定）** は対象の Git リポジトリで `codex exec` を使えるよう認証し、作業に必要な書き込みを許可してください。
- **Claude Code** は同じ環境で `claude` の認証を済ませてください。ワーカーは非対話モードで起動します。設計には読み取り・検索ツールだけを渡し、直接実行と実装では Claude Code に設定した権限を使います。編集やテストに必要な操作を事前に許可してください。[Claude Code の非対話実行](https://code.claude.com/docs/en/headless)

Herdr のセッション内で、このプラグインのリポジトリのルートから実行します。

```sh
sh scripts/build.sh
herdr plugin link "$PWD"
herdr plugin pane open --plugin wataru-toriumi.factory-board --entrypoint board
```

起動したら、ボードで作業先を選んで最初のカードを作成します。[ボードの操作](board.md)

## キーバインドでボードを開く

Herdr の設定ファイル（通常は `~/.config/herdr/config.toml`）に次の割り当てを追加します。

```toml
[[keys.command]]
key = "prefix+shift+b"
type = "plugin_action"
command = "wataru-toriumi.factory-board.open-board"
description = "Factory Board を開く"
```

設定を保存し、Herdr のセッション内で `herdr server reload-config` を実行します。既定のプレフィックスなら、**Ctrl+B を押して離し、次に Shift+B** を押すと、現在のペインにボードが開きます。

## プラグインの登録を解除する

登録を解除するには `herdr plugin unlink wataru-toriumi.factory-board` を実行してください。[Herdr 公式プラグインガイド](https://herdr.dev/tui/docs/plugins/)
