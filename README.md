<p align="center">
<img src="./docs/image/logo.svg" alt="Kessoku Logo" width="200">
</p>

# Kessoku

[![Go Reference](https://pkg.go.dev/badge/github.com/mazrean/kessoku.svg)](https://pkg.go.dev/github.com/mazrean/kessoku)
[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/mazrean/kessoku)
[![codecov](https://codecov.io/gh/mazrean/kessoku/branch/main/graph/badge.svg)](https://codecov.io/gh/mazrean/kessoku)

**Kessoku is a compile-time dependency injection library for Go that speeds up application startup through parallel dependency injection.** Unlike traditional DI frameworks that initialize services sequentially, Kessoku automatically executes independent providers in parallel, dramatically reducing startup time for applications with multiple slow services. Built as a powerful alternative to google/wire, it generates optimized code at compile time with zero runtime overhead.

**Sequential:** DB → Cache → Auth = Total waiting time  
**Parallel:** DB + Cache + Auth = Fastest service wins

```go
// Before: Sequential (google/wire)
wire.Build(NewDB, NewCache, NewAuth, NewApp)     // Each waits for previous

// After: Parallel (Kessoku)  
kessoku.Inject[*App]("InitApp",
    kessoku.Async(kessoku.Provide(NewDB)),      // }
    kessoku.Async(kessoku.Provide(NewCache)),   // } All run together
    kessoku.Async(kessoku.Provide(NewAuth)),    // }
    kessoku.Provide(NewApp),                    // waits for all
)                                               // Fastest possible startup
```

**Result:** Every restart gets faster. Multiple slow services? Maximum impact.

## Why This Matters

**Your typical day:** Restart your app 10 times during development. Each restart wastes time waiting for services to start one by one.

```mermaid
gantt
    title Sequential vs Parallel Startup
    dateFormat X
    axisFormat %L
    
    section Sequential (slow)
    DB Service    :0, 3
    Cache Service :3, 5  
    Auth Service  :5, 6
    
    section Parallel (fast)
    DB Service    :0, 3
    Cache Service :0, 2
    Auth Service  :0, 1
```

**Perfect for:**
- **Cold start nightmares:** Your Lambda/serverless function times out during initialization
- **Dev restart hell:** You restart your app 10+ times daily, losing 3+ seconds each time  
- **Multi-DB apps:** PostgreSQL + Redis + S3 + Auth0 = 800ms+ sequential startup pain
- **google/wire refugees:** You love compile-time DI but hate slow startup times

## Quick Start

**Install kessoku:**

```bash
go get -tool github.com/mazrean/kessoku/cmd/kessoku
```

**Create `main.go`:**
```go
package main

import (
    "fmt"
    "time"
    "github.com/mazrean/kessoku"
)

func SlowDB() string {
    time.Sleep(200 * time.Millisecond)
    return "DB-connected"
}

func SlowCache() string {
    time.Sleep(150 * time.Millisecond)
    return "Cache-ready"
}

//go:generate go tool kessoku $GOFILE

var _ = kessoku.Inject[string]("InitApp",
    kessoku.Async(kessoku.Provide(SlowDB)),
    kessoku.Async(kessoku.Provide(SlowCache)),
    kessoku.Provide(func(db, cache string) string {
        return fmt.Sprintf("App running with %s and %s", db, cache)
    }),
)

func main() {
    start := time.Now()
    result, _ := InitApp()
    fmt.Printf("%s in %v\n", result, time.Since(start))
}
```

**Run:**
```bash
go generate && go run main.go
# Shows: App running with DB-connected and Cache-ready (parallel startup)
```

## Installation

**Recommended:**
```bash
go get -tool github.com/mazrean/kessoku/cmd/kessoku
```

<details>
<summary>Download binary</summary>

Download the latest binary for your platform from the [releases page](https://github.com/mazrean/kessoku/releases).

**Linux/macOS:**
```bash
# Download and install (replace with your platform)
curl -L -o kessoku.tar.gz https://github.com/mazrean/kessoku/releases/latest/download/kessoku_Linux_x86_64.tar.gz
tar -xzf kessoku.tar.gz
sudo mv kessoku /usr/local/bin/
```

**Windows:**
```powershell
# Download and install
Invoke-WebRequest -Uri "https://github.com/mazrean/kessoku/releases/latest/download/kessoku_Windows_x86_64.zip" -OutFile "kessoku.zip"
Expand-Archive -Path "kessoku.zip" -DestinationPath "."
Move-Item "kessoku.exe" "$env:USERPROFILE\bin\" -Force
# Add $env:USERPROFILE\bin to your PATH if not already added
```

**Verify:**
```bash
kessoku --version
```

</details>

<details>
<summary>Homebrew (macOS/Linux)</summary>

```bash
brew install mazrean/tap/kessoku
```

</details>

<details>
<summary>Other Package Managers</summary>

**Debian/Ubuntu:**
```bash
wget https://github.com/mazrean/kessoku/releases/latest/download/kessoku_amd64.deb
sudo apt install ./kessoku_amd64.deb
```

**Red Hat/CentOS/Fedora:**
```bash
wget https://github.com/mazrean/kessoku/releases/latest/download/kessoku_amd64.rpm
# For CentOS/RHEL 7 and older
sudo yum install ./kessoku_amd64.rpm
# For CentOS/RHEL 8+ and Fedora
sudo dnf install ./kessoku_amd64.rpm
```

**Alpine Linux:**
```bash
wget https://github.com/mazrean/kessoku/releases/latest/download/kessoku_amd64.apk
sudo apk add --allow-untrusted kessoku_amd64.apk
```

</details>

## API Reference

**Full docs:** [pkg.go.dev/github.com/mazrean/kessoku](https://pkg.go.dev/github.com/mazrean/kessoku)

**Examples:** [examples/](./examples/) - basic, async_parallel, sets 

- **`kessoku.Async(provider)`** - Make this provider run in parallel
- **`kessoku.Provide(fn)`** - Regular provider (sequential)
- **`kessoku.Inject[T](name, ...)`** - Generate the injector function
- **`kessoku.Set(...)`** - Group providers for reuse
- **`kessoku.Value(val)`** - Inject constants
- **`kessoku.Bind[Interface](impl)`** - Interface → implementation

**Rule:** Independent async providers run in parallel, dependent ones wait automatically.

---

## Migrating from google/wire

Already using google/wire? Kessoku provides a migration tool to convert your wire configuration files automatically.

### Quick Migration

```bash
# on wire config directory
go tool kessoku migrate

# or specify wire config directory with patterns
go tool kessoku migrate ./pkg/wire -o kessoku.go
```

<details>
<summary>Advanced Migration Options</summary>

```text
Usage: kessoku migrate [<patterns> ...] [flags]

Migrate wire config to kessoku

Arguments:
  [<patterns> ...]    Go package patterns to migrate

Flags:
  -h, --help                   Show context-sensitive help.
  -l, --log-level="info"       Log level
  -v, --version                Show version and exit.

  -o, --output="kessoku.go"    Output file path
```
</details>

### Example

**Before (wire.go):**
```go
//go:build wireinject

package main

import "github.com/google/wire"

func InitializeApp() (*App, error) {
    wire.Build(
        NewApp,
        NewPostgresRepo,
        wire.Bind(new(Repository), new(*PostgresRepo)),
    )
    return nil, nil
}
```

**After (kessoku.go):**
```go
//go:generate go tool kessoku $GOFILE

package main

import "github.com/mazrean/kessoku"

var _ = kessoku.Inject[*App](
    "InitializeApp",
    kessoku.Provide(NewApp),
    kessoku.Bind[Repository](kessoku.Provide(NewPostgresRepo)),
)
```

---

## vs Alternatives

| | Kessoku | google/wire | uber-go/dig |
|---|---------|-------------|---------|
| **Startup Speed** | Parallel | Sequential | Sequential + runtime |
| **Learning Curve** | Minimal | Minimal | Steep |
| **Production Ready** | Yes | Yes | Yes |

**Choose Kessoku if:** You have multiple slow services (DB, cache, APIs) and startup time matters  
**Choose google/wire if:** You want maximum simplicity and startup speed isn't critical  
**Choose uber/fx if:** You need complex lifecycle management and don't mind runtime overhead

## Supports

This project receives support from GMO FlattSecurity's “GMO Open Source Developer Support Program” and regularly conducts security assessments using “Takumi byGMO.”

<a href="https://flatt.tech/oss/gmo/trampoline" target="_blank"><img src="https://flatt.tech/assets/images/badges/gmo-oss.svg" height="24px"/></a>

## License
[MIT License](./LICENSE)
