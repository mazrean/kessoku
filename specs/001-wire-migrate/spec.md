# Feature Specification: Wire to Kessoku Migration Tool

**Feature Branch**: `001-wire-migrate`
**Created**: 2026-01-02
**Status**: Draft
**Input**: User description: "google/wireの設定からkessokuの設定へ変換するmigrateサブコマンドを作成して。この時、NewSet→Set、Bind→Bind、Value→Value、InterfaceValue→Bind+Value、Struct→Provide+無名関数、FieldsOf→Provide+無名関数、に変換するようにして"

## Clarifications

### Session 2026-01-02

- Q: 出力先のデフォルト動作は？ → A: デフォルトは`kessoku.go`ファイルに出力、`-o`フラグで出力先を変更可能
- Q: 複数ファイル変換時の出力動作は？ → A: すべての変換結果を1つの`kessoku.go`にマージして出力

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Basic Wire File Migration (Priority: P1)

開発者が既存の google/wire 設定ファイルを kessoku 形式に変換する。wire の基本的なプロバイダー定義（NewSet、Bind、Value）を含むファイルを入力として、対応する kessoku 設定を出力する。

**Why this priority**: 最も基本的で頻繁に使用される wire パターンの変換。これがなければ移行ツールとしての価値がない。

**Independent Test**: wire.NewSet、wire.Bind、wire.Value を含む単純なファイルを migrate コマンドで変換し、正しい kessoku.Set、kessoku.Bind、kessoku.Value が生成されることを確認できる。

**Acceptance Scenarios**:

1. **Given** kessokuツールがインストール済み **When** `kessoku migrate --help`を実行 **Then** migrateサブコマンドのヘルプが表示される
2. **Given** wire.NewSetを含むGoファイル **When** migrateコマンドを実行 **Then** kessoku.Setに変換された出力が生成される
3. **Given** wire.Bind[Interface, Impl]()を含むGoファイル **When** migrateコマンドを実行 **Then** `kessoku.Bind[Interface](kessoku.Provide(NewImpl))`形式に変換される
4. **Given** wire.Value(v)を含むGoファイル **When** migrateコマンドを実行 **Then** kessoku.Value(v)に変換された出力が生成される
5. **Given** 入力ファイルを指定しデフォルト出力 **When** migrateコマンドを実行 **Then** `kessoku.go`ファイルに出力される
6. **Given** `-o output.go`フラグを指定 **When** migrateコマンドを実行 **Then** `output.go`ファイルに出力される
7. **Given** wireパッケージをインポートしているファイル **When** migrateコマンドを実行 **Then** 出力ではwireインポートがkessokuインポートに置き換えられる

---

### User Story 2 - InterfaceValue Migration (Priority: P2)

開発者が wire.InterfaceValue を使用したファイルを kessoku 形式に変換する。wire.InterfaceValue は kessoku.Bind と kessoku.Value の組み合わせに展開される。

**Why this priority**: InterfaceValue は wire 特有のパターンであり、kessoku には直接対応する機能がないため、適切な展開が必要。

**Independent Test**: wire.InterfaceValue を含むファイルを変換し、kessoku.Bind と kessoku.Value のペアが正しく生成されることを確認できる。

**Acceptance Scenarios**:

1. **Given** wire.InterfaceValue(new(SomeInterface), someValue) を含むファイル, **When** migrate コマンドを実行, **Then** kessoku.Bind[SomeInterface](kessoku.Value(someValue)) が生成される

---

### User Story 3 - Struct Injection Migration (Priority: P2)

開発者が wire.Struct を使用したファイルを kessoku 形式に変換する。wire.Struct は構造体を構築するプロバイダーであるため、kessoku.Provide と無名関数に変換される。

**Why this priority**: Struct インジェクションは依存関係の注入で頻繁に使用されるパターン。

**Independent Test**: wire.Struct を含むファイルを変換し、フィールド指定に応じて適切な kessoku コードが生成されることを確認できる。

**Acceptance Scenarios**:

1. **Given** wire.Struct(new(Config), "*") を含むファイル（Config に Field1 と Field2 がある場合）, **When** migrate コマンドを実行, **Then** `kessoku.Provide(func(f1 Field1Type, f2 Field2Type) *Config { return &Config{Field1: f1, Field2: f2} })` が生成される
2. **Given** wire.Struct(new(Config), "Field1", "Field2") を含むファイル, **When** migrate コマンドを実行, **Then** `kessoku.Provide(func(f1 Field1Type, f2 Field2Type) *Config { return &Config{Field1: f1, Field2: f2} })` が生成される

---

### User Story 4 - FieldsOf Migration (Priority: P3)

開発者がwire.FieldsOfを使用したファイルをkessoku形式に変換する。wire.FieldsOfは構造体の特定フィールドをプロバイダーとして公開するため、各フィールドに対してkessoku.Provideと無名関数の組み合わせに変換される。

**Why this priority**: FieldsOfは比較的高度なwireパターンであり、使用頻度は低いが完全な移行のためには必要。

**Independent Test**: wire.FieldsOfを含むファイルを変換し、各フィールドに対応するkessoku.Provideが生成されることを確認できる。

**Acceptance Scenarios**:

1. **Given** wire.FieldsOf(new(Config), "DB", "Cache")を含むファイル, **When** migrateコマンドを実行, **Then** DBフィールド用の`kessoku.Provide(func(c *Config) DBType { return c.DB })`とCacheフィールド用の`kessoku.Provide(func(c *Config) CacheType { return c.Cache })`が生成される

---

### User Story 5 - Multiple File Migration (Priority: P3)

開発者がディレクトリ内の複数のwireファイルを一括でkessoku形式に変換する。すべての変換結果は1つの`kessoku.go`ファイルにマージされる。

**Why this priority**: 大規模プロジェクトでは複数ファイルの移行が必要だが、基本機能が動作した後の拡張として位置づけ。

**Independent Test**: 複数のwireファイルを含むディレクトリに対してmigrateコマンドを実行し、すべてのファイルが1つの`kessoku.go`に統合されることを確認できる。

**Acceptance Scenarios**:

1. **Given** 同一パッケージ内の複数のwire設定ファイル **When** migrateコマンドをディレクトリパスで実行 **Then** すべての変換結果が1つの`kessoku.go`にマージされる
2. **Given** 同名の識別子（変数/関数/型/定数）が複数ファイルに存在する場合 **When** migrateコマンドを実行 **Then** 名前衝突エラーを検出しエラーメッセージを表示する
3. **Given** 異なるパッケージのwireファイルを指定した場合 **When** migrateコマンドを実行 **Then** パッケージ境界エラーを表示して処理を中断する
4. **Given** 複数ファイルに同じインポートが存在する場合 **When** migrateコマンドを実行 **Then** インポートは重複排除され1回のみ出力される

---

### Edge Cases

**EC-001: wireパターンなしのファイル**
- **Given** wireパターンを含まないGoファイル, **When** migrateコマンドを実行, **Then** 警告メッセージ「No wire patterns found in [filename]」を表示し出力ファイルは生成されない

**EC-002: wireインポートなしのファイル**
- **Given** wireパッケージのインポートがないGoファイル, **When** migrateコマンドを実行, **Then** 警告メッセージ「No wire import found in [filename]」を表示しスキップする

**EC-003: 構文エラーのあるファイル**
- **Given** 構文エラーを含むGoファイル, **When** migrateコマンドを実行, **Then** エラーメッセージを表示し終了コード1で処理を中断する

**EC-004: 複数のNewSet**
- **Given** 複数のwire.NewSetが1ファイルに存在する場合, **When** migrateコマンドを実行, **Then** それぞれを個別のkessoku.Setに変換する

**EC-005: ネストされたNewSet**
- **Given** Set内にSetを含むネストされたwire.NewSet, **When** migrateコマンドを実行, **Then** ネスト構造を維持してkessoku.Setに変換する

**EC-006: 既存ファイルの上書き**
- **Given** 出力先ファイルが既に存在する場合, **When** migrateコマンドを実行, **Then** 既存ファイルを上書きする

**EC-007: インポートの重複排除**
- **Given** 複数ファイルをマージする際に同じインポートが複数回出現する場合, **When** migrateコマンドを実行, **Then** インポートを重複排除して1回のみ出力する

**EC-008: ビルドタグの保持**
- **Given** ビルドタグ（//go:buildまたは// +build）を含むファイル, **When** migrateコマンドを実行, **Then** ビルドタグは出力ファイルに保持されない（wireファイル固有のタグは移行不要）

**EC-009: コメントの扱い**
- **Given** wire関数呼び出しに付随するコメント **When** migrateコマンドを実行 **Then** コメントは保持せず変換後のコードのみを出力する

**EC-010: 変換不可パターンの検出**
- **Given** サポートされていないwireパターン（wire.Buildなど）を含むファイル **When** migrateコマンドを実行 **Then** 警告メッセージ「Unsupported pattern: [pattern] at [location]」を表示し処理を続行する

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: システムは`kessoku migrate`サブコマンドを提供しなければならない
- **FR-002**: システムはwire.NewSetをkessoku.Setに変換できなければならない
- **FR-003**: システムはwire.Bind[Interface, Impl]()を`kessoku.Bind[Interface](kessoku.Provide(NewImpl))`形式に変換できなければならない
- **FR-004**: システムはwire.Valueをkessoku.Valueに変換できなければならない
- **FR-005**: システムはwire.InterfaceValueをkessoku.Bind(kessoku.Value(...))の形式に変換できなければならない
- **FR-006**: システムはwire.Struct(型, "*")をkessoku.Provideと無名関数に変換できなければならない
- **FR-007**: システムはwire.Struct(型, フィールド指定)をkessoku.Provideと無名関数に変換できなければならない
- **FR-008**: システムはwire.FieldsOfをkessoku.Provideと無名関数の組み合わせに変換できなければならない
- **FR-009**: システムはデフォルトで`kessoku.go`ファイルに変換結果を出力し、`-o`フラグで出力先を変更できなければならない
- **FR-010**: システムは変換できないパターンを検出した場合、警告またはエラーを報告しなければならない
- **FR-011**: システムはwireパッケージのインポートをkessokuパッケージに置き換えなければならない
- **FR-012**: システムは複数ファイルをマージする際にインポートを重複排除しなければならない
- **FR-013**: システムは同一パッケージ内のファイルのみをマージ対象としなければならない

### Key Entities

- **Wire Configuration**: 変換元となる google/wire の設定コード。NewSet、Bind、Value、InterfaceValue、Struct、FieldsOf などのパターンを含む
- **Kessoku Configuration**: 変換先となる kessoku の設定コード。Set、Bind、Value、Struct、Provide などのパターンを使用
- **Conversion Rule**: wire パターンから kessoku パターンへの変換ルール。1対1変換と1対多変換がある

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 基本的な wire パターン（NewSet、Bind、Value）を含むファイルの変換成功率が 100% である
- **SC-002**: InterfaceValue を含むファイルの変換成功率が 100% である
- **SC-003**: Struct（全フィールド）を含むファイルの変換成功率が 100% である
- **SC-004**: Struct（フィールド指定）および FieldsOf を含むファイルの変換成功率が 100% である
- **SC-005**: 変換後のコードがコンパイルエラーなく通過する
- **SC-006**: 変換後のコードが go fmt 準拠のフォーマットである

## Assumptions

- 入力ファイルはGoソースコードとして提供される（構文エラーがある場合はEC-003に従いエラー処理される）
- wireパッケージは`github.com/google/wire`からインポートされていると仮定する
- 変換対象はprovider set定義ファイルのみであり、injectorファイル（wire_gen.go）は対象外とする
- 複雑なプロバイダー関数（カスタム関数）はそのまま保持され、kessoku.Provideでラップされると仮定する

## Scope

### In Scope

- wire.NewSetからkessoku.Setへの変換
- wire.Bindからkessoku.Bindへの変換
- wire.Valueからkessoku.Valueへの変換
- wire.InterfaceValueからkessoku.Bind(kessoku.Value(...))への展開
- wire.Struct(型, "*")からkessoku.Provide+無名関数への変換
- wire.Struct(型, フィールド指定)からkessoku.Provide+無名関数への変換
- wire.FieldsOfからkessoku.Provide+無名関数への変換
- wireインポートのkessokuインポートへの置き換え
- 複数ファイルマージ時のインポート重複排除
- 同一パッケージ制約の検証

### Out of Scope

- wire_gen.go（生成された injector コード）の変換
- wire.Build の変換（kessoku.Inject への変換は別機能として検討）
- 実行時の依存関係解決や検証
- 変換結果の自動テスト実行
