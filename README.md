# Factory Board

Herdr TUI で作業をカードにして Codex に渡すプラグインです。手動で始めた作業と毎日の定期実行を同じボードで追跡し、終了後に Codex の最終メッセージを読めます。

## ローカルで起動する

macOS または Linux で、Herdr 0.7.0 以降と Go 1.21 以降を使用します。Go が必要なのはビルド時だけです。
作業を実行するには、Herdr から起動したペインで `codex exec` を使えるようにし、対象の Git リポジトリで Codex CLI の認証と設定を済ませてください。ファイル編集を伴う作業には、Codex 側でその作業先への書き込みを許可します。
Herdr のセッション内で、このプラグインのリポジトリのルートから実行します。

```sh
go build -o factory-board ./src
herdr plugin link "$PWD"
herdr plugin pane open --plugin wataru-toriumi.factory-board --entrypoint board
```

## カードから作業を始める

ボード内で次のように入力します。各行を Enter で確定してください。

```text
dir /absolute/path/to/project
add ログイン画面のテストを追加する
start 1
```

`dir` は対象リポジトリの絶対パスです。カードと定期実行はそれぞれ登録時の作業先を記憶します。`start` にはボードに表示されたカード番号を指定します。専用タブでカード本文をプロンプトとして Codex が起動し、カードが `In Progress` に移ります。

- `open 1` で実行中の担当ペインを開きます。
- 空の入力で Enter を押すと、カードとワークスペースの状態を更新します。
- 終了コードが 0 なら `Done`、実行に失敗したら `Failed` になります。`result 1` で最終メッセージや起動エラーを表示します。
- 失敗したカードは `start 1` で再実行できます。強制終了で状態が残ったときは `recover 1` を実行してから再開します。
- `q` でボードを閉じます。カード、結果、定期実行の設定は Herdr のプラグイン状態ディレクトリにセッション・ワークスペース単位で保存します。

## 毎日決まった時刻に実行する

たとえば、毎朝 09:00 に前日の変更を要約するには、作業先を設定したボードで次のように入力します。

```text
daily 09:00 昨日のコミットを確認し、変更点を要約する
```

登録すると同じワークスペースに **Factory Board Scheduler** タブが開きます。ボードを閉じても、Herdr サーバーとこのタブが動いている間は定期実行が続きます。毎回新しいカードが作成され、手動カードと同じように結果を確認できます。

時刻は実行マシンのローカル時刻です。登録時にその日の指定時刻を過ぎていれば翌日から開始します。スリープや停止から復帰したときは、当日の時刻を過ぎていればその日の分を一度実行します。過去の日付の分は繰り越しません。

スケジューラーのタブを閉じると停止し、ボードで `scheduler` を入力すると再開します。個別の定期実行を解除するには `remove 定期番号` を使います。実行済みのカードは履歴として残ります。

## 開発

開発用の Go は [mise.toml](mise.toml) で指定しています。シェルで mise を有効化している場合は、初回にリポジトリのルートで次のコマンドを実行します。

```sh
mise trust
mise install
go build -o factory-board ./src
```

プラグインのメタデータとビルド・起動コマンドは [herdr-plugin.toml](herdr-plugin.toml)、ボードの操作は [src/main.go](src/main.go)、定期実行は [src/scheduler.go](src/scheduler.go)、カードの実行と復旧は [src/worker.go](src/worker.go)、Herdr CLI との連携は [src/herdr.go](src/herdr.go) にあります。
コードを変更したら、`go build -o factory-board ./src` を再実行し、ペインを開き直してください。

登録を解除するには `herdr plugin unlink wataru-toriumi.factory-board` を実行してください。[Herdr 公式プラグインガイド](https://herdr.dev/tui/docs/plugins/)
