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
- **Testing**: `testing` + `github.com/stretchr/testify/assert`
- **Build**: GoReleaser, CGO_ENABLED=0, Linux only (amd64/arm64)

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
├── textextract/
│   ├── textextract.go     # External command text extraction
│   └── textextract_test.go
├── testutil/
│   ├── testutil.go        # Test helpers (temp dir, etc.)
│   └── testutil_test.go
├── testdata/              # Test fixtures (PDF, text, config YAMLs)
├── config.yaml.sample     # Example configuration
└── version.txt            # Embedded at build time (//go:embed)
```

### Logging (logger/logger.go)
- Structured logging using log/slog
- File rotation via lumberjack.v2
- Configurable levels: INFO, DEBUG, ERROR, WARN
- JSON and text output formats
- Configuration through config.yaml

## Development Guidelines

### Code Style
- Use structured logging (slog) instead of fmt.Printf for application output
- All public functions should have documentation comments
- Keep functions focused and under 50 lines when possible
- Use meaningful variable names
- Use `gofmt` / `goimports` formatting. Max line length 140.
- Group imports: stdlib first, third-party second, internal (`fileganizer/...`) last.
- Copyright header on every `.go` file:
  ```go
  // Copyright 2023-2026 The Fileganizer Authors. All rights reserved.
  // SPDX-License-Identifier: MIT
  ```
- Copyright year is always 20XX-20YY or just 20XX if 20XX and 20YY are the same. 20XX is the file's creation year (from `git log --diff-filter=A --follow` for that file). 20YY is the current year.
- No copyright on `version.txt`
- Put copyright on `README.md` at the end of the file.
- All source file should have a copyright header (the syntax depends on the file type). For non-go files, use the appropriate comment syntax (e.g., `//` for `.txt`, `/*` for `.md`) and set a header similar to the `.go` files.

### Linting
- Flags (like `-c` or `-f`) are never constants. When the linter complain, add a `//nolint` directive to the line.

### Naming
- Package names: single word, lowercase, matching directory name.
- Files: `package.go` and `package_test.go` (same package, not `_test` external).
- Constants: PascalCase or ALL_CAPS for string constants. Exported types: PascalCase. Unexported: camelCase.

### Patterns
- Constructors: `New()` returns a value (not pointer) for small structs.
- Logger: singleton via `logger.Get()`.
- Context: pass `context.Context` to operations that may need cancellation.

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
- The environment variables defined in `config.yaml.sample` with key `env` are not related to the configuration management. They are used to set environment variables for the application. They are not supposed to be prefixed with `FILEGANIZER_`.

### Error Handling
- Use slog for error logging with context
- Return errors explicitly, don't panic
- Log errors with relevant context (IDs, filenames, URLs)
- Gracefully handle missing or corrupted configuration
- Configuration errors cause immediate exit with os.Exit(1)

### Key Dependencies
- Avoid `github.com/sirupsen/logrus` (blocked by depguard linter).
- Use `interface{}` → rewritten to `any` by gofmt.

## Testing Conventions
- Write tests alongside features in `*_test.go` files
- Use testify assertions (`assert.Equal`, `assert.Nil`, `assert.FileExists`)
- Tests should be isolated and use temporary files/directories
- Always clean up test artifacts with defer
- Test data files must be placed in the `testdata/` directory
- Unused testdata files must be removed

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

### Commits
- Never commit, never stage (`git add`), never run any `git commit` command — even if the user explicitly asks you to commit. If you ask and the user says yes, you can commit.
- Instead, always suggest a full `git commit` command for the user to run themselves.
- Never work in the `main` branch.
- Never commit to `main` branch.
- Commit message should be clear and descriptive.
- Commit message should follow the [Conventional Commits](https://www.conventionalcommits.org/) specification.
- Commit message should be in the format: `<type>: <description>`.
- Commit message should be lowercase and should not start with a capital letter.
- Commit message should be descriptive and should not be too short.

### Build & Run
- Build: `echo dev > version.txt && go build`
- Test: `echo dev > version.txt && go test ./...`
- Run: `./fileganizer -c <config.yaml> -f <file.pdf>` (dry-run) or `-r` (execute).
- CLI flags: `-f` (file), `-c` (config), `-t` (show text only), `-r` (run command), `-V` (version).

### Linting
- Lint: `golangci-lint run ./...`. When it fails for versioning reasons, fallback to `docker run -t --rm -v $(pwd):/app:z -w /app golangci/golangci-lint:v2.12.2 golangci-lint run ./...`

### Version Management
- Keep Go version consistent across `.github/workflows/*.yml`. Use `"1.26"` (resolves to latest patch) or `stable` for `actions/setup-go`.
- `go.mod` is the exception: its `go` directive sets the minimum Go version. Only bump when the code requires a newer toolchain feature.
- Keep all tooling in `.github/workflows/` (goreleaser, golangci-lint, actions/\*) at their latest stable versions.
- When updating a version, check all references across the project (go.mod, workflows, AGENTS.md).

