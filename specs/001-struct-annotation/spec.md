# Feature Specification: Struct Annotation for Field Expansion

**Feature Branch**: `001-struct-annotation`
**Created**: 2026-01-01
**Status**: Draft
**Input**: User description: "構造体を型引数で受け取り、その全フィールドを展開するkessoku.Structアノテーションを追加"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Expand Config Struct Fields (Priority: P1)

A developer has a configuration struct with multiple fields that need to be injected into various services. Instead of creating individual provider functions for each field, they want to declare a single `kessoku.Struct` annotation that automatically makes all fields available as dependencies.

**Why this priority**: This is the core functionality of the feature. Without field expansion, the annotation provides no value.

**Independent Test**: Can be fully tested by creating a struct with multiple fields, using `kessoku.Struct` to expand them, and verifying that each field becomes an injectable dependency.

**Acceptance Scenarios**:

1. **Given** a struct `Config` with fields `DBHost string` and `DBPort int`, **When** `kessoku.Struct[*Config]()` is declared with a Config provider, **Then** both `string` (DBHost) and `int` (DBPort) become available as individual dependencies for injection.
2. **Given** a struct with exported fields only, **When** `kessoku.Struct` is used, **Then** only exported fields are expanded as dependencies.
3. **Given** a struct with unexported fields, **When** `kessoku.Struct` is used, **Then** unexported fields are ignored (not expanded as dependencies).
4. **Given** a struct with fields `Zebra int`, `Apple string`, `Mango bool`, **When** code is generated, **Then** field providers are generated in alphabetical order: Apple, Mango, Zebra (verifiable by inspecting generated code).
5. **Given** an injector with `kessoku.Struct[*ConfigA]()` followed by `kessoku.Struct[*ConfigB]()`, **When** code is generated, **Then** ConfigA's fields (alphabetically ordered) appear before ConfigB's fields (alphabetically ordered) in the generated output.

---

### User Story 2 - Type-Safe Field Access (Priority: P1)

A developer wants to inject specific struct fields into their services while maintaining type safety. The generated code should properly extract each field from the struct instance.

**Why this priority**: Type safety is essential for the feature to be useful in production. Incorrect type handling would cause compilation or runtime errors.

**Independent Test**: Can be tested by verifying that generated code correctly extracts fields with proper types and that compilation succeeds.

**Acceptance Scenarios**:

1. **Given** a struct with fields of different types (string, int, custom types), **When** code is generated, **Then** each field accessor returns the correct type.
2. **Given** a struct with pointer and non-pointer fields, **When** `kessoku.Struct` is used, **Then** both pointer and non-pointer types are correctly handled.

---

### User Story 3 - Use with Existing Annotations (Priority: P2)

A developer wants to use `kessoku.Struct` alongside existing annotations like `Provide`, `Async`, `Bind`, and `Set` to create a complete dependency injection setup.

**Why this priority**: Integration with existing annotations is important for adoption, but the core expansion functionality must work first.

**Independent Test**: Can be tested by combining `kessoku.Struct` with `Provide` (for the struct itself) and other annotations in an `Inject` declaration.

**Acceptance Scenarios**:

1. **Given** `kessoku.Provide(NewConfig)` and `kessoku.Struct[*Config]()` in the same injector, **When** code is generated, **Then** the Config is created first, then its fields are extracted and made available.
2. **Given** a `Set` containing `kessoku.Struct`, **When** the Set is used in an injector, **Then** the struct fields are properly expanded.
3. **Given** `kessoku.Struct[*Config]()` without a corresponding `*Config` provider, **When** code generation is attempted, **Then** an error is reported indicating the missing provider.
4. **Given** `kessoku.Async(kessoku.Provide(NewConfig))` and `kessoku.Struct[*Config]()`, **When** code is generated, **Then** field extraction waits for the async Config provider to complete before extracting fields.
5. **Given** a struct field of type `DB` and `kessoku.Bind[Database](kessoku.Provide(NewDB))`, **When** `kessoku.Struct` expands the field, **Then** the `DB` field is available and can satisfy dependencies requiring `Database` interface via the Bind.
6. **Given** a `Set` containing `kessoku.Struct[*ConfigA]()` then `kessoku.Struct[*ConfigB]()`, and this Set is used between other providers in an Inject call, **When** code is generated, **Then** ConfigA's fields appear before ConfigB's fields, and the Set's fields appear at the Set's position in the Inject call (in-place expansion).

---

### User Story 4 - Handle Struct with Embedded Fields (Priority: P3)

A developer has a struct with embedded (anonymous) fields and wants the embedded type to be available as a dependency.

**Why this priority**: Embedded fields are a common Go pattern, but handling them is more complex and can be addressed after the basic functionality works.

**Independent Test**: Can be tested by creating a struct with embedded fields and verifying that the embedded type becomes a dependency.

**Acceptance Scenarios**:

1. **Given** a struct `App` with embedded `Config` (value type), **When** `kessoku.Struct[*App]()` is used, **Then** the `Config` value becomes a dependency (extracted as `app.Config`).
2. **Given** a struct `App` with embedded `*Config` (pointer type), **When** `kessoku.Struct[*App]()` is used, **Then** the `*Config` pointer becomes a dependency.
3. **Given** a struct with embedded fields, **When** `kessoku.Struct` is used, **Then** nested fields of the embedded struct are NOT recursively expanded (only direct fields of the target struct are provided).
4. **Given** a struct `App` with an unexported embedded type (e.g., `type App struct { config }`), **When** `kessoku.Struct[*App]()` is used, **Then** the unexported embedded `config` is ignored and no dependency is produced for it.

---

### Edge Cases

- What happens when the struct has no exported fields? → No dependencies are produced; silent success (no warning).
- What happens when multiple fields have the same type (within one struct or across structs)? → All fields are expanded; type conflicts are detected and reported as an error by existing kessoku mechanisms.
- What happens when `kessoku.Struct` is used with a non-struct type? → Error is reported with a clear message.
- What happens when `kessoku.Struct[*Config]` is used but only `Config` (non-pointer) provider exists? → Error is reported; exact type match is required.
- What happens when `kessoku.Struct[T]` is used but no provider for `T` exists? → Error is reported with a message indicating the missing provider.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST provide a `kessoku.Struct[T]()` annotation function that accepts a struct type as a type parameter.
- **FR-002**: System MUST expand all exported fields of the struct type as individual dependencies when code is generated.
- **FR-003**: System MUST ignore unexported fields (fields starting with lowercase) during expansion. This includes embedded fields with unexported type names (e.g., `type Foo struct { bar }` - the embedded `bar` is ignored).
- **FR-004**: System MUST generate field accessor code that extracts each field from the struct instance.
- **FR-005**: System MUST detect and report type conflicts when multiple fields provide the same type, whether within a single struct (e.g., two `string` fields) or across multiple structs. Type conflict detection is delegated to existing kessoku mechanisms.
- **FR-006**: System MUST require exact type match between `kessoku.Struct[T]` and the provider type (e.g., `Struct[*Config]` requires a `*Config` provider; `Struct[Config]` requires a `Config` provider). Mismatches MUST result in an error at code generation time. The error message MUST include "type mismatch" and both the expected and actual types.
- **FR-007**: System MUST integrate with existing annotations as follows:
  - `Provide`: The struct type MUST be provided via `Provide` before `Struct` can expand its fields.
  - `Async`: When a struct provider is wrapped in `Async`, field extraction occurs after the async provider completes.
  - `Bind`: Expanded field types can be bound to interfaces using `Bind` (e.g., bind a field's concrete type to an interface).
  - `Set`: `Struct` can be included in a `Set` and will expand fields when the Set is used in an injector.
  - `Inject`: `Struct` is used within `Inject` declarations to specify which struct's fields should be expanded.
- **FR-008**: System MUST report a clear error at code generation time when `kessoku.Struct` is used with a non-struct type. The error message MUST include "not a struct type" and the actual type name provided.
- **FR-009**: System MUST handle embedded (anonymous) struct fields by providing the embedded type's value as a dependency. The embedded type follows the same pointer/value semantics as the parent struct field declaration. Nested fields of embedded structs are NOT recursively expanded.
- **FR-010**: System MUST report a clear error at code generation time when `kessoku.Struct[T]` is used but no provider for type `T` exists in the injector. The error message MUST include "no provider" and the missing type name.
- **FR-011**: System MUST produce no dependencies when a struct has no exported fields. No warning is emitted (silent success).
- **FR-012**: System MUST generate field providers in a deterministic order: fields within each struct are ordered alphabetically by field name. When multiple `Struct` annotations exist in an injector, each struct's fields are grouped together, and structs are ordered by the position of their `kessoku.Struct` annotation within the `kessoku.Inject` call (first `Struct` annotation's fields come first, then second, etc.). When `Struct` is used inside a `Set`, the Set is expanded in-place at its position in the Inject call, and `Struct` annotations within the Set follow their declaration order within that Set. This ensures stable, reproducible output across runs.

### Key Entities

- **Struct Annotation**: A marker that indicates a struct type should have its fields expanded as dependencies. Contains the struct type as a type parameter.
- **Field Provider**: A generated dependency provider for each exported field of the struct. Depends on the struct instance and provides the field value.
- **Struct Instance**: The source struct from which fields are extracted. Must be provided by another provider (e.g., `kessoku.Provide`).

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: For a struct with N exported fields (where no type conflicts exist), `kessoku.Struct` generates exactly N field providers (one per exported field). When type conflicts exist, code generation fails before providers are usable (see SC-004). This is verified by test cases that assert the count and types of generated providers match the struct's exported fields.
- **SC-002**: All existing kessoku features (Async, Bind, Set) continue to work correctly when combined with Struct annotation (verified by integration tests).
- **SC-003**: Generated code compiles without errors when Struct annotation is used correctly (verified by compilation in CI).
- **SC-004**: Clear, actionable error messages are displayed when Struct annotation is misused. Each error is detected at code generation time (not compile time) and includes the following key phrases in the message:
  - Non-struct type: message MUST contain "not a struct type" and the actual type name
  - Type conflicts: message MUST contain "multiple providers provide" and the conflicting type name (consistent with existing kessoku error messages)
  - Missing struct provider: message MUST contain "no provider" and the missing type name
  - Pointer/value mismatch: message MUST contain "type mismatch" and both expected and actual types

## Clarifications

### Session 2026-01-01

- Q: 同一構造体内に同じ型のフィールドが複数ある場合の扱いは？　→ A: 全フィールドを展開し、既存の型衝突検出に任せる（Option C）
- Q: ポインター/値の型不一致時の動作は？　→ A: 厳密な型一致を要求しエラーを報告（FR-006）
- Q: 構造体プロバイダーが存在しない場合は？　→ A: エラーを報告（FR-010）
- Q: エクスポートフィールドがない場合は？　→ A: 依存を生成せず、警告なしで成功（FR-011）
- Q: フィールド展開の順序は？　→ A: 各構造体内のフィールドはアルファベット順、複数のStruct注釈がある場合はInject内の宣言順で決定論的に生成（FR-012）
- Q: 埋め込みフィールドの扱いは？　→ A: 埋め込み型の値を依存として提供、ネストしたフィールドは再帰展開しない（FR-009）
- Q: 非公開の埋め込み型はどうなる？　→ A: 通常の非公開フィールドと同様に無視される（FR-003）

## Assumptions

- The struct type must be provided by another provider in the same injector (e.g., via `kessoku.Provide`). The Struct annotation only expands fields; it does not create the struct instance.
- Field types are used directly for dependency matching. If named types are needed for disambiguation, users should create wrapper types or use `Bind`.
- When a struct has multiple fields of the same type, all fields are expanded and type conflict detection is delegated to existing kessoku mechanisms (same behavior as other type conflicts).
- Only direct fields of the specified struct are expanded. Nested struct fields require additional `kessoku.Struct` annotations.
- The feature follows existing kessoku patterns for error handling and code generation.
