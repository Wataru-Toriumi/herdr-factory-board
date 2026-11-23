# 開発

開発用の Go は [mise.toml](../mise.toml) で指定しています。シェルで mise を有効化している場合は、初回にリポジトリのルートで次のコマンドを実行します。

```sh
mise trust
mise install
go build -o factory-board ./src
```

プラグインのビルドと各ペインの起動コマンドは [herdr-plugin.toml](../herdr-plugin.toml) で定義しています。コードを変更したら、`go test ./...` で全パッケージをテストし、再ビルドしてペインを開き直してください。

## レイヤーと依存方向

[main.go](../src/main.go) はワークスペースの保存先を開き、起動引数に応じて TUI、ワーカー、スケジューラーを起動します。その下のコードは4層に分けています。モジュール内で直接 import できるのは、表の同じ行か、それより下の行のパッケージです。

| ディレクトリ | 担当する処理 | パッケージ |
| --- | --- | --- |
| `src/tui` | ボードの表示、タブ切り替え、キー入力、作業先の選択 | [tui](../src/tui) と下表の表示用パッケージ |
| `src/application` | コマンドの実行、予定の登録と起動、カードの実行・キャンセル・復旧 | [command](../src/application/command)、[scheduler](../src/application/scheduler)、[worker](../src/application/worker) |
| `src/infrastructure` | ワークスペースごとの `board.json` の読み書きとロック、Herdr CLI によるペイン操作 | [store](../src/infrastructure/store)、[herdr](../src/infrastructure/herdr) |
| `src/domain` | カードの状態と設計・レビューの条件、定期実行の時刻判定と次回時刻の計算 | [board](../src/domain/board)、[schedule](../src/domain/schedule) |

たとえば、`every 15m` の入力は TUI から `command` を経て `scheduler` に届きます。`scheduler` は `schedule` で実行間隔を設定し、`store` で保存します。画面更新の際は、TUI が `store` からボードを、`herdr` からペイン一覧を直接読み取ります。

画面に表示する内容は次のパッケージで組み立てます。ウィジェットへの反映、キー入力、フォーカス、操作の実行は `src/tui` が担当します。

| パッケージ | 担当する表示 |
| --- | --- |
| [taskboard](../src/tui/taskboard) | レーンごとの表示行、プロジェクト絞り込み、更新後の選択位置 |
| [agentlist](../src/tui/agentlist) | ペインと担当カードの対応、入力待ちの並び順と件数 |
| [detail](../src/tui/detail) | カードと元の予定の詳細本文、状態ごとのショートカットと操作案内 |
| [automation](../src/tui/automation) | 定期実行の一覧、プロジェクト絞り込み、スケジューラーの稼働表示 |

### 依存ルールの確認

[依存テスト](../src/dependencies_test.go) は層を逆向きに参照する import と、層の外に追加した `src` 配下のパッケージを検出します。個別の制約も確認します。

- `board` が直接 import できるモジュール内のパッケージは `schedule` だけです。
- `schedule` と `herdr` は標準ライブラリだけを直接 import します。
- 画面ライブラリの `tview` と `tcell/v2` を直接 import するのは `tui` だけです。
- 表示用パッケージはモジュール内の `domain` 配下を参照できます。`agentlist` はペイン・ワークスペースの型を使うため、`infrastructure/herdr` も参照できます。

検査対象は現在のビルド環境で使われる本体コードです。全体は `go test ./...`、依存関係だけなら `go test -count=1 -run '^TestPackageDependencies$' ./src` で確認できます。
