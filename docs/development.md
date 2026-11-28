# 開発

シェルで mise を有効にし、初回はリポジトリのルートで次のコマンドを実行します。開発用の Go のバージョンは [mise.toml](../mise.toml) で確認できます。

```sh
mise trust
mise install
mise run build
```

プラグインのビルドと各ペインの起動コマンドはマニフェストで定義しています。[herdr-plugin.toml](../herdr-plugin.toml)

## テストと lint

コードを変更したら、リポジトリのルートで順に実行し、ペインを開き直してください。

| コマンド | 用途 |
| --- | --- |
| `mise run test` | race 検出付きで全テストを実行する |
| `mise run lint` | golangci-lint で静的解析を実行する |
| `mise run build` | プラグインを再ビルドする |

CI は PR の作成・更新時と `main` への push 時に、Linux と macOS でテストを、Linux で lint を実行します。[CI の設定](../.github/workflows/ci.yml)

## レイヤーと依存方向

起動時には [main.go](../src/main.go) がワークスペースの保存先を開き、引数に応じて TUI、ワーカー、スケジューラーを起動します。モジュール内の直接 import は、次の表の同じ層または下位の層に限ります。パッケージごとの制約もあります。

| 層 | 担当する処理 | パッケージ |
| --- | --- | --- |
| `src/tui` | ボードの表示、タブ切り替え、キー入力、作業先の選択 | [tui](../src/tui) と各画面のパッケージ |
| `src/application` | コマンドの実行、予定の登録と起動、カードの実行・キャンセル・復旧 | [command](../src/application/command)、[scheduler](../src/application/scheduler)、[worker](../src/application/worker) |
| `src/infrastructure` | ワークスペースごとの `board.json` の読み書きとロック、Herdr CLI によるペイン操作、GitHub CLI による PR の取得・表示 | [github](../src/infrastructure/github)、 [store](../src/infrastructure/store)、[herdr](../src/infrastructure/herdr) |
| `src/domain` | カードの状態と設計・レビューの条件、定期実行の時刻判定と次回時刻の計算 | [board](../src/domain/board)、[schedule](../src/domain/schedule) |

たとえば `every 15m` の入力は、TUI から `command` を経て `scheduler` に届きます。`scheduler` は `schedule` で間隔を設定し、`store` で保存します。画面更新時は TUI が `store` からボードを、`herdr` からペイン一覧を直接読み取ります。

### TUI の構成

[run.go](../src/tui/run.go) が端末を開き、[app](../src/tui/app) が画面を組み立てます。タブ切り替え、画面間のフォーカス移動、保存、ペイン操作と、それらをまたぐ結合テストは `app` にあります。端末の貼り付け入力は [terminal](../src/tui/terminal)、共通の一覧部品と選択行の保持は `widgets` が担当します。

| 画面のパッケージ | 担当する表示と操作 |
| --- | --- |
| [taskboard](../src/tui/taskboard) | レーンとカードの描画、画面幅に応じた列数、選択の追従、カードのキー操作、新規タスクフォーム |
| [agentlist](../src/tui/agentlist) | エージェント一覧、入力待ちの並び順と件数、担当ペインを開く操作 |
| [reviewlist](../src/tui/reviewlist) | GitHub のレビュー依頼一覧、選んだ PR を開く操作 |
| [detail](../src/tui/detail) | カードと元の予定の詳細、更新時の再描画、状態に応じたショートカットと操作案内 |
| [automation](../src/tui/automation) | 予定一覧、プロジェクトの絞り込み、稼働状態の表示、登録・編集フォームと時刻入力 |
| [projectpicker](../src/tui/projectpicker) | 作業先一覧、フォルダ移動、新しい作業先の選択 |

### 依存ルールの確認

依存テストは、層を逆向きに参照する import と、層の外に追加した `src` 配下のパッケージを検出します。検査対象は現在のビルド環境で使われる本体コードです。[依存テスト](../src/dependencies_test.go)

| 対象 | 許可する直接 import |
| --- | --- |
| `board` | モジュール内では `schedule` のみ |
| `schedule`、`herdr` | 標準ライブラリのみ |
| `taskboard`、`automation` | モジュール内では `domain` 配下と `widgets` |
| `agentlist` | モジュール内では `domain` 配下、`widgets`、ペイン・ワークスペースの型を持つ `infrastructure/herdr` |
| `reviewlist` | モジュール内では `infrastructure/github` と `widgets` |
| `detail` | モジュール内では `domain` 配下のみ |
| `projectpicker` | モジュール内では `widgets` のみ |
| `widgets`、`terminal` | モジュール内の他パッケージへの依存なし |

画面ライブラリの `tview` と `tcell/v2` を直接 import できるのは、起動入口の `tui`、`app`、上の「TUI の構成」にある画面パッケージ、`widgets`、`terminal` です。全テストは `go test ./...`、依存関係だけなら `go test -count=1 -run '^TestPackageDependencies$' ./src` で確認できます。

## リリースノートを公開する

1. リリースするバージョンを `herdr-plugin.toml` に設定し、`main` に取り込みます。
2. 公開前に本文を確認したい場合は、GitHub CLI でプレビューします。初回の `v0.1.0` なら次のコマンドで生成できます。プレビューではタグや Release は作成されません。

   ```sh
   gh api --method POST 'repos/{owner}/{repo}/releases/generate-notes' \
     -f tag_name=v0.1.0 -f target_commitish=main --jq .body
   ```

3. ローカルの `main` を更新し、対象のコミットに `vX.Y.Z` 形式のタグを付けて push します。

   ```sh
   git switch main
   git pull --ff-only
   git tag -a v0.1.0 -m 'v0.1.0'
   git push origin refs/tags/v0.1.0
   ```

タグの push 後、CI の全検査に通ると GitHub Release が公開されます。リリースノートには前回のリリース以降にマージした PR が載ります。[リリースの設定](../.github/workflows/release.yml)
