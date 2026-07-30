# AGENTS.md

## Project Overview
Fileganizer is a Go CLI tool that processes documents through a pipeline: text extraction → grok pattern parsing → Go template rendering → optional shell command execution. Primary use case: renaming PDF invoices by extracting fields from their text content.

## Technology Stack
- **Language**: Go 1.26+
- **Configuration**: `github.com/knadh/koanf` (YAML parsing)
- **CLI parsing**: `github.com/spf13/pflag`
- **Grok patterns**: `github.com/logrusorgru/grokky`
- **Logging**: `log/slog` (stdlib, text handler, defaults to stderr)
- **Log rotation**: `gopkg.in/natefinch/lumberjack.v2`
- **Templates**: `text/template` (stdlib)
- **PDF extraction**: `github.com/pdfcpu/pdfcpu`
- **Testing**: `testing` + `github.com/stretchr/testify/assert`
- **Build**: GoReleaser (`.goreleaser.yaml`), CGO_ENABLED=0, Linux only (amd64/arm64)
- Avoid `github.com/sirupsen/logrus` (blocked by depguard linter)

## Project Structure
```
.
├── main.go                # Entry point, pipeline orchestration
├── main_test.go           # End-to-end tests
├── config/
│   ├── config.go          # CLI parsing, YAML config loading
│   └── config_test.go
├── grok/
│   ├── grok.go            # Grok pattern matching wrapper
│   └── grok_test.go
├── output/
│   ├── output.go          # Go template rendering
│   └── output_test.go
├── logger/
│   ├── logger.go          # Structured logging (slog singleton)
│   └── logger_test.go
├── pdftotext/
│   ├── pdftotext.go       # Builtin PDF text extraction (pdfcpu)
│   ├── pdftotext_test.go
│   └── testdata/          # PDF test fixtures (forged + bank statements)
├── textextract/
│   ├── textextract.go     # External command text extraction
│   └── textextract_test.go
├── testutil/
│   ├── testutil.go        # Test helpers (temp dir, etc.)
│   └── testutil_test.go
├── testdata/              # Config YAMLs, text fixtures, minimal.wav
├── config.yaml.sample     # Example configuration
├── go.mod                 # Go module definition
├── go.sum                 # Go module checksums
├── LICENSE                # License (also mandatory in testdata subdirectories)
├── README.md
└── version.txt            # Embedded at build time (//go:embed)
```

## Development Guidelines

### Code Style
- Use structured logging (slog) instead of fmt.Printf for application output. The exception is `main()` in `main.go` which may use `fmt.Print`/`fmt.Printf`/`fmt.Fprintf` directly for CLI output (result to stdout, version to stdout, error message to stderr). Never use it in any other function.
- All public functions should have documentation comments
- Keep functions focused and under 50 lines when possible
- Use meaningful variable names
- Use `gofmt` / `goimports` formatting. Max line length 140.
- Group imports: stdlib first, third-party second, internal (`fileganizer/...`) last.
- Flags (like `-c` or `-f`) are never constants. When the linter complains, add `//nolint`.
- No global or function-scoped `//nolint`. Only line-scoped `//nolint` is allowed.
- Exception: `pdftotext/fontwidths_std.go` may use declaration-scoped `//nolint:gochecknoglobals,dupl` — the font metric data is inherently repetitive and the global map is intentional.
- Keep the whole code simple and stupid (KISS). No over-engineering, no unnecessary abstractions.
- Copyright header on every source file. For `.go` files:
  ```go
  // Copyright 2023-2026 The Fileganizer Authors. All rights reserved.
  // SPDX-License-Identifier: MIT
  ```
- Copyright year: `20XX-20YY` (creation year to current year), or `20XX` if same year. Derive `20XX` from `git log --diff-filter=A --follow <file>`.
- No copyright on `version.txt`. README.md copyright goes at end of file.
- Non-Go files use appropriate comment syntax: `//` for `.txt`, `#` for `.yaml`/`.yml`, `/*` for `.md`.
- Use `any` instead of `interface{}` (gofmt rewrites it)

### Naming
- Package names: single word, lowercase, matching directory name.
- Files: `package.go` and `package_test.go` (same package, not `_test` external).
- Constants: PascalCase or ALL_CAPS for string constants. Exported types: PascalCase. Unexported: camelCase.

### Patterns
- Constructors: `New()` returns a value (not pointer) for small structs.
- Logger: singleton via `logger.Get()`.
- Context: pass `context.Context` to operations that may need cancellation.
- Grok patterns are compiled fresh on every `Parse()` call. Each pattern is only used a few times (once per file per fileDescription), so caching adds complexity for no measurable gain.

### Configuration Management
- Use Koanf for all YAML parsing
- Use spf13/pflag for CLI flag parsing
- Provide sensible defaults for all config options
- Validate configuration values at startup
- Support both short and long CLI flags
- With Koanf, prefer getting typed values than using the Get() method.

### Environment Variables
- Use optional environment variables for configuration (e.g., `FILEGANIZER_LOGGING_LEVEL`...)
- Environment variables should override values from the config file
- Environment variable names should be in uppercase with underscores (e.g., `FILEGANIZER_LOGGING_LEVEL`)
- Environment variables should be documented in the `config.yaml.sample`
- Environment variables should be prefixed with `FILEGANIZER_` (e.g., `FILEGANIZER_DIR`, `FILEGANIZER_LOGGING_LEVEL`)
- The `env` key in `config.yaml.sample` is not related to configuration management. It sets environment variables for the application and is not prefixed with `FILEGANIZER_`.

### Error Handling
- Use slog for error logging with context
- Return errors explicitly, don't panic
- Log errors with relevant context (IDs, filenames, URLs)
- Gracefully handle missing or corrupted configuration
- Configuration errors cause immediate exit with os.Exit(1)

### Logging (logger/logger.go)
- Structured logging using log/slog
- File rotation via lumberjack.v2
- Configurable levels: INFO, DEBUG, ERROR, WARN
- JSON and text output formats
- Configuration through config.yaml

### Testing Conventions
- Write tests alongside features in `*_test.go` files
- Use testify assertions (`assert.Equal`, `assert.Nil`, `assert.FileExists`)
- Tests should be isolated and use temporary files/directories
- Always clean up test artifacts with defer
- Test data files go in the package-local `testdata/` directory (e.g., `pdftotext/testdata/` for PDFs, root `testdata/` for config YAMLs)
- A `LICENSE` file (MIT) is mandatory in `testdata/` directories containing files from external sources — it is not test data and is excluded from the unused file check
- Config files in `testdata/` testing invalid/broken patterns must be named `config.broken.<what>.yaml` (e.g., `config.broken.grok.yaml`, `config.broken.template.yaml`)
- `config.broken.mime.yaml` tests the "no extractor for MIME type" error path — `audio/wave` is deliberately unsupported and will never have a builtin extractor added
- Unused testdata files must be removed
- **Coverage requirements**:
  - All packages: 100% statement coverage
  - Exception: `testutil/` minimum 70% (error branches for unreachable system calls)
  - Exception: `main` (root package) minimum 90% (end-to-end tests)

## Common Tasks

### Adding a New Configuration Option
1. Add field to Config struct in config/config.go
2. Add parsing logic in readConfig() function
3. Add test case in config_test.go
4. Update config.yaml.sample with example value

### Modifying CLI Flags
1. Update parseFlags() in config/config.go
2. Add both short and long form support
3. Update help text
4. Test with -h flag

### Updating Go Version
1. Update `go 1.xx.x` in `go.mod`
2. Update Go version reference in AGENTS.md
3. Update `.golangci.yml` if it references a Go version
4. Run `go mod tidy` after updating

## Commits
- Never commit, never stage (`git add`), never run `git commit` — even if explicitly asked. Always suggest the command for the user to run.
- Never work in or commit to the `main` branch.
- Commit message: clear, descriptive, lowercase, no capital start.
- Follow [Conventional Commits](https://www.conventionalcommits.org/): `<type>: <description>`.

## Build & Run
- Build: `echo dev > version.txt && go build`
- Test: `echo dev > version.txt && go test ./...`
- Run: `./fileganizer -c <config.yaml> -f <file.pdf>` (dry-run) or `-r` (execute).
- CLI flags: `-f` (file), `-c` (config), `-t` (show text only), `-r` (run command), `-V` (version).

## Linting
- Run: `golangci-lint run ./...`
- Fallback (version mismatch): `docker run -t --rm -v $(pwd):/app:z -w /app golangci/golangci-lint:v2.12.2 golangci-lint run ./...`

## Version Management
- Keep Go version consistent across `.github/workflows/*.yml`. Use `"1.26"` (resolves to latest patch) or `stable` for `actions/setup-go`.
- `go.mod` is the exception: its `go` directive sets the minimum Go version. Only bump when the code requires a newer toolchain feature.
- Keep all tooling in `.github/workflows/` (goreleaser, golangci-lint, actions/\*) at their latest stable versions.
- When updating a version, check all references across the project (go.mod, workflows, AGENTS.md).
