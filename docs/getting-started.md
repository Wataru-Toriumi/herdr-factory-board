# ローカルで起動する

## 必要な環境

macOS または Linux で Herdr 0.7.0 以降を使います。ビルドには Go 1.21 以降が必要です。macOS では音声入力ヘルパーを Swift でビルドするため、Xcode Command Line Tools（`xcode-select --install`）も用意してください。Go と Swift のコンパイラを使うのはビルド時だけです。

Herdr のペインで、使いたいエージェントの CLI を起動できるようにします。

- **Codex（初期設定）** を使う場合は、対象の Git リポジトリで `codex exec` を実行できるように認証し、作業に必要な書き込みを許可します。
- **Claude Code** を使う場合は、同じ環境で `claude` の認証を済ませ、編集やテストに必要な操作を許可します。ワーカーは非対話モードで起動します。設計時は読み取り・検索ツールだけを使い、直接実行時と実装時は Claude Code に設定した権限を使います。[Claude Code の非対話実行](https://code.claude.com/docs/en/headless)

## ボードを開く

Herdr のセッション内で、このリポジトリのルートから順に実行します。

```sh
sh scripts/build.sh
herdr plugin link "$PWD"
herdr plugin pane open --plugin wataru-toriumi.factory-board --entrypoint board
```

ボードが開いたら、タスクタブで `n` を押して最初のカードを作成します。[ボードの操作](board.md)

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

## 旧版から更新する

実行中の作業が終わったら、旧版のボードと稼働中の **Factory Board Scheduler** タブを閉じ、新版で開き直してください。旧版で保存すると会話履歴が失われ、新しく登録したフォルダの一覧も失われる場合があります。

毎日の時刻指定だけに対応した旧版から更新する場合は、新しいボードで間隔指定の予定を登録すると、新版のスケジューラーが起動します。[定期実行](automation.md)

## プラグインの登録を解除する

`herdr plugin unlink wataru-toriumi.factory-board` を実行します。[Herdr 公式プラグインガイド](https://herdr.dev/tui/docs/plugins/)
