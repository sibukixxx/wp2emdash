# ケーススタディ: 大規模ニュースメディア (13,000 記事) の WordPress → EmDash 実移行から得た教訓

実際に進行中の WordPress → EmDash (Cloudflare Workers + D1 + R2) 移行プロジェクト
（publish 記事 ~13,000 件、mysqldump ~2GB、カスタムテーブル 75 個のニュースメディア）の
移行パイプラインを調査し、`wp2emdash` が汎用ツールとして先回りすべき論点を整理したもの。

対象プロジェクトのパイプラインは 3 ステージ構成:

```
mysqldump → source JSON (extract) → seed.json (build) → D1 (apply / apply-remote)
```

に加えて、EmDash コアが管理しないデータ（SEO メタ・ランキング・著者リンク・メディア台帳）を
**out-of-band ローダー**で別適用する二層構造だった。

## 教訓カタログ

### 1. DB 抽出フェーズ

| 事象 | 内容 | wp2emdash への示唆 |
| --- | --- | --- |
| deferred index deadlock | mysqldump は「CREATE TABLE → bulk INSERT → 末尾で deferred `ALTER TABLE ADD KEY`」の順。最後のテーブルの存在確認だけで SELECT を始めると索引ビルドと metadata-lock デッドロックになる。`PROCESSLIST` のアクティブクエリ枯渇まで待つ 2 フェーズ待機が必要 | dump リストア待機を自動化するなら 2 フェーズ待機を標準に |
| JSON 集約の silent truncate | `JSON_ARRAYAGG` は `group_concat_max_len`（既定 1MB）で黙って切れ、下流が "Unterminated string in JSON" で死ぬ。`max_allowed_packet` も同様 | DB 直読みする機能では session 変数を明示的に引き上げる |
| TSV パースの脆さ | 本文・description 等の自由テキストは tab / 改行を含み得る。1 行 1 `JSON_OBJECT` で吐いてパースする方が堅牢 | `seo extract-meta` の indexable 対応で採用済み |
| PHP serialize | `a:1:{i:0;i:2;}`（ID 配列）、`a:1:{i:0;s:1:"1";}`（flag）、`wp_kses` の `\/` エスケープ等が postmeta に多数 | audit の serialized カウントは実装済み。既知パターンのデコーダ提供は将来課題 |

### 2. 移行先 (Cloudflare D1) の構造的制約

すべて実プロジェクトで「事故ってから」発覚した:

- **~100KB/statement 上限 (SQLITE_TOOBIG)**: 長文記事の INSERT が落ちる。先頭 chunk を
  `INSERT OR REPLACE`、以降を `UPDATE SET col = col || <chunk>` に分割して回避。
  エスケープ済み quote やマルチバイト文字を chunk 境界で割らない atom 単位分割が必要
- **FTS5 shadow table へ直接 INSERT 不可**: dump からの適用時は除外が必要
- **per-table dump は索引を含まない**: `CREATE INDEX` を明示的に emit し、`ANALYZE` を実行
  （sqlite_stat1 が無いと D1 は索引を使わないことがある）
- **sqlite クライアントのバージョン差**: 新しめの sqlite3 は Unicode を `unistr()` で emit するが
  D1 は未対応。dump を作る sqlite のバージョンを固定する必要があった

→ `wp2emdash audit` に `customization.oversized_content_count`（本文 90KB 超の publish 投稿数）を
追加し、`db plan` がこのリスクと対処（chunk 分割）を事前に提示するようにした。

### 3. 認証・環境依存データ

- **passkey 資格情報は origin バインド**。ローカルの認証テーブルをそのまま remote へコピーすると
  「誰もログインできず、`setup_complete=true` のため setup wizard も起動しない」環境が生まれる。
  auth/identity テーブルは**構造のみ**運び、環境ごとに初回管理者を作るのが正解
- byline 等の **ULID は環境ごとに変わる**ため、環境を跨ぐリンクは slug をキーにして SQL を可搬化する

→ `db plan` の `target_notes` に恒常的な警告として反映済み。

### 4. タイムスタンプ

EmDash 0.8.0 の `emdash seed` は created/updated/published を**適用時刻で上書き**する。
移行後に `wp_published_at` からのバックフィル UPDATE（値が違う場合のみ更新＝冪等）が必要だった。

→ `target_notes` に「seed importer がソース日時を保持するか検証し、しなければバックフィルを計画」を追加。

### 5. メディア移行

- **方針 A（旧 URL 維持）**: `wp-content/uploads/...` パスを Worker → R2 で配信し続ける。
  さらに seed 内 URL を host-less に relativize する方式へ発展
- **hosted 環境は旧 origin に fallback しない** → R2 バケットを満たし切ってから公開。
  全 manifest URL の 200 検証（非空 body）をカットオーバーのゲートにする
- チェックポイントは **target（dry/local/staging/prod）別**に分離（dry-run が本適用を no-op 化する事故防止）。
  チェックポイント喪失時は deployed Worker へ Range probe して再構築
- orphan URL の 404/401 は大規模コーパスでは想定内 → fail 条件は「閾値超過 or 全件失敗」のみ
- ローカル (miniflare) は SQLITE_BUSY のため並列度 1、リモートは並列 6 + 指数バックオフ

→ `media scan` / `media sync` / `media verify` は実装済み。`target_notes` に完全性ゲートを追加。
checkpoint/resume と失敗ポリシーの体系化は v0.5 以降の `media sync` 拡張候補。

### 6. SEO メタデータ

**最重要の発見**: Yoast 14+ (2020〜) は SEO メタを `{prefix}yoast_indexable` テーブルに正規化して
保持し、旧来の `_yoast_wpseo_*` postmeta は空・stale なことがある。postmeta だけ見る抽出は
エディタの上書きを黙って取りこぼす。紛らわしい `{prefix}yoast_seo_meta` は内部リンク数だけの
テーブルで、メタデータ源ではない。

- 「エディタが実際に上書きした行のみ」を対象にする（all-NULL 行は移行先のデフォルト計算を
  上書きするだけ無駄）
- 主カテゴリは `{prefix}yoast_primary_term` に別テーブルで持つ（未対応・将来課題）

→ `seo extract-meta` が indexable テーブルを自動検出して postmeta とマージするよう対応済み
（postmeta 明示値 > indexable > core の優先順位、`source: "yoast_indexable"` で由来を保持）。

### 7. 冪等性・再実行安全性

実プロジェクトで機能した安全網:

- **決定論 ID**（`post-{wp_id}` 等）+ id ベース upsert で再実行が安全
- **row-count regression ガード**: 新しい抽出結果が前回より行数が少なければ
  「transformer バグ / 途中で切れた dump」として abort（明示フラグでのみ override）。
  source が空なら前回値を preserve
- out-of-band SQL はすべて `ON CONFLICT ... DO UPDATE` + 事前集約で同一キー二重 UPSERT を排除
- 「値が違う場合のみ UPDATE」「EXISTS ガード」で no-op 再実行を保証

→ `target_notes` に決定論 ID と regression ガードを追加。

### 8. 取りこぼしの体系的検出

- ある postmeta 由来フィールド（1,679 記事で使用）が seed から drop していたことが後から発覚。
  対策として「dump の per-post 全 field/meta を棚卸し → ターゲット schema の未 populate
  フィールドを列挙」する監査を実施
- 旧テーブル列 ↔ 新 TS 型のドリフト検出器（mapping 駆動・CI 実行）も別途作られた。
  WordPress は `admin_init` の runtime migration で列が増えるため、SQL dump のテーブル定義だけを
  信じてはいけない

→ v0.5 の `theme analyze` / `plugins analyze` と並ぶ候補として「postmeta キー棚卸しレポート」
（使用回数付きで meta_key を列挙し、移行計画の keep/transform/omit 判断材料にする）が有望。

## 今回 wp2emdash に反映したもの

1. `seo extract-meta`: `{prefix}yoast_indexable` の自動検出・抽出・マージ（§6）
2. `audit`: `customization.oversized_content_count` の計測（§2）
3. `db plan`: `target_notes`（移行先側の落とし穴チェックリスト）と oversized リスクの追加（§2, §3, §4, §5, §7）

## 今後のロードマップへの示唆（未実装）

優先度順:

1. **postmeta キー棚卸し** (`db meta-inventory` 等): meta_key ごとの使用行数・serialized 率を列挙（§8）
2. **`seo extract-meta` の primary term 対応**: `{prefix}yoast_primary_term` の抽出（§6）
3. **`media sync` の checkpoint/resume + 失敗ポリシー**: target 別 checkpoint、orphan 許容 +
   閾値 fail、`media verify` のカットオーバーゲート化（§5）
4. **`db plan` の chunk 計画**: oversized 投稿の一覧出力（ID・バイト数）と分割方針の提示（§2）
5. **HTML → Portable Text 変換は引き続きスコープ外**とする（案件ごとの WP ショートコード・
   埋め込み依存が強く「薄いラッパ」原則に反する）。ただし変換前の**ノイズ実態レポート**
  （shortcode 種別・iframe/script 出現数の内訳）は audit の延長として価値がある

なお、対象プロジェクトの複雑度計測スクリプトは本リポジトリの `legacy-bash/` と同系で、
スコアリング設計（14 観点・加点式）が実案件のスコーピングにそのまま使われていたことを確認した。
