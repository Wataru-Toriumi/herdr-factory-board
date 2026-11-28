# 開発

開発用の Go は [mise.toml](../mise.toml) で指定しています。シェルで mise を有効化している場合は、初回にリポジトリのルートで次のコマンドを実行します。

```sh
mise trust
mise install
mise run build
```

プラグインのビルドと各ペインの起動コマンドは [herdr-plugin.toml](../herdr-plugin.toml) で定義しています。コードを変更したら、テストと lint を実行し、再ビルドしてペインを開き直してください。

## テストと lint

`mise run test` で race 検出付きの全テスト、`mise run lint` で golangci-lint による静的解析を実行できます。CI は PR の作成・更新と `main` への push を対象に、Linux と macOS でテスト、Linux で lint を実行します。[CI の設定](../.github/workflows/ci.yml)

## リリースノートを公開する

リリースするバージョンを `herdr-plugin.toml` に設定して `main` に取り込みます。そのコミットに付けた `vX.Y.Z` 形式のタグを push すると、CI の全検査に通った後に GitHub Release が公開されます。リリースノートには前回のリリース以降にマージした PR が載ります。[リリースの設定](../.github/workflows/release.yml)

たとえば初回の `v0.1.0` なら、先に GitHub CLI で本文をプレビューできます。このコマンドでタグや Release は作成されません。

```sh
gh api --method POST 'repos/{owner}/{repo}/releases/generate-notes' \
  -f tag_name=v0.1.0 -f target_commitish=main --jq .body
```

公開するときはローカルの `main` を更新し、対象のコミットにタグを付けて送ります。

```sh
git switch main
git pull --ff-only
git tag -a v0.1.0 -m 'v0.1.0'
git push origin refs/tags/v0.1.0
```

## レイヤーと依存方向

[main.go](../src/main.go) はワークスペースの保存先を開き、起動引数に応じて TUI、ワーカー、スケジューラーを起動します。その下のコードは4層に分けています。モジュール内で直接 import できるのは、表の同じ行か、それより下の行のパッケージです。

| ディレクトリ | 担当する処理 | パッケージ |
| --- | --- | --- |
| `src/tui` | ボードの表示、タブ切り替え、キー入力、作業先の選択 | [tui](../src/tui) と下表の表示用パッケージ |
| `src/application` | コマンドの実行、予定の登録と起動、カードの実行・キャンセル・復旧 | [command](../src/application/command)、[scheduler](../src/application/scheduler)、[worker](../src/application/worker) |
| `src/infrastructure` | ワークスペースごとの `board.json` の読み書きとロック、Herdr CLI によるペイン操作 | [store](../src/infrastructure/store)、[herdr](../src/infrastructure/herdr) |
| `src/domain` | カードの状態と設計・レビューの条件、定期実行の時刻判定と次回時刻の計算 | [board](../src/domain/board)、[schedule](../src/domain/schedule) |

たとえば、`every 15m` の入力は TUI から `command` を経て `scheduler` に届きます。`scheduler` は `schedule` で実行間隔を設定し、`store` で保存します。画面更新の際は、TUI が `store` からボードを、`herdr` からペイン一覧を直接読み取ります。

画面に表示する内容は次のパッケージで組み立てます。各機能が一覧・フォーム・詳細画面を描画し、専用のキー操作を受け取ります。`src/tui` 直下の [run.go](../src/tui/run.go) が端末を開き、[app](../src/tui/app) が画面を組み立てます。タブ切替、画面間のフォーカス移動、保存、ペイン操作と、これらをまたぐ結合テストは `app` にあります。端末の貼り付け入力は [terminal](../src/tui/terminal) が扱います。複数の画面で使う一覧部品と選択行の保持は `widgets` にまとめています。

| パッケージ | 担当する表示 |
| --- | --- |
| [taskboard](../src/tui/taskboard) | レーンの描画、画面幅に応じた列数、選択の追従、カードのキー操作、新規タスクの入力フォーム |
| [agentlist](../src/tui/agentlist) | エージェント一覧の描画、入力待ちの並び順と件数、選択したペインへの移動受付 |
| [detail](../src/tui/detail) | カードと元の予定の詳細画面、更新時の再描画、状態ごとのショートカットと操作案内 |
| [automation](../src/tui/automation) | 定期実行の一覧、プロジェクト絞り込み、稼働表示、登録・編集フォームと時刻入力 |
| [projectpicker](../src/tui/projectpicker) | 作業先一覧、フォルダ移動、新しい作業先の選択 |

### 依存ルールの確認

[依存テスト](../src/dependencies_test.go) は層を逆向きに参照する import と、層の外に追加した `src` 配下のパッケージを検出します。個別の制約も確認します。

- `board` が直接 import できるモジュール内のパッケージは `schedule` だけです。
- `schedule` と `herdr` は標準ライブラリだけを直接 import します。
- 画面ライブラリの `tview` と `tcell/v2` を直接 import できるのは起動入口の `tui`、`app`、上表の画面パッケージ、`widgets`、`terminal` です。
- 表示用パッケージはモジュール内の `domain` 配下を参照できます。一覧を持つ `taskboard`・`agentlist`・`automation` は `widgets` も参照できます。`projectpicker` のモジュール内依存は `widgets` のみです。`widgets` と `terminal` はモジュール内の他パッケージから独立しています。`agentlist` はペイン・ワークスペースの型を使うため、`infrastructure/herdr` も参照できます。

検査対象は現在のビルド環境で使われる本体コードです。全体は `go test ./...`、依存関係だけなら `go test -count=1 -run '^TestPackageDependencies$' ./src` で確認できます。
