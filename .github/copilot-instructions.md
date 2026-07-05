# Copilot Instructions for Fileganizer

## Project Overview
Fileganizer is a Go CLI tool that processes documents through a pipeline: text extraction → grok pattern parsing → Go template rendering → optional shell command execution. Primary use case: renaming PDF invoices by extracting fields from their text content.

## Tech Stack
- **Language**: Go 1.26
- **CLI parsing**: `github.com/spf13/pflag`
- **Grok patterns**: `github.com/logrusorgru/grokky`
- **Logging**: `log/slog` (stdlib, text handler, defaults to stdout)
- **Templates**: `text/template` (stdlib)
- **Testing**: `testing` + `github.com/stretchr/testify/assert`
- **Build**: GoReleaser, CGO_ENABLED=0, Linux only (amd64/arm64)

## Code Conventions

### Style
- Use `gofmt` / `goimports` formatting. Max line length 140.
- Group imports: stdlib first, third-party second, internal (`fileganizer/...`) last.
- Copyright header on every `.go` file:
  ```go
  // Copyright 2023 The Fileganizer Authors. All rights reserved.
  // SPDX-License-Identifier: MIT
  ```
- Copyright year is always 2023-XXXX (project founding year to the present)
- All public functions should have documentation comments
- Keep functions focused and under 50 lines when possible
- Use meaningful variable names

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
- With Koand, prefer getting typed values than using the Get() method.

### Environment Variables
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

### Key Dependencies
- Avoid `github.com/sirupsen/logrus` (blocked by depguard linter).
- Use `interface{}` → rewritten to `any` by gofmt.

## Testing Conventions
- Write tests alongside features in `*_test.go` files
- Use testify assertions (`assert.Equal`, `assert.Nil`, `assert.FileExists`)
- Tests should be isolated and use temporary files/directories
- Always clean up test artifacts with defer

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
├── testdata/              # Test fixtures (PDF, text, config YAMLs)
├── config.yaml.sample     # Example configuration
└── version.txt            # Embedded at build time (//go:embed)
```

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

## Dependencies
- `github.com/knadh/koanf` - Configuration management
- `gopkg.in/natefinch/lumberjack.v2` - Log rotation
- `github.com/spf13/pflag` - CLI flag parsing
- `github.com/stretchr/testify` - Testing utilities

## Build & Run
- Build: `echo dev > version.txt && go build`
- Test: `echo dev > version.txt && go test ./...`
- Run: `./fileganizer -c <config.yaml> -f <file.pdf>` (dry-run) or `-r` (execute).
- CLI flags: `-f` (file), `-c` (config), `-t` (show text only), `-r` (run command), `-V` (version).
- Lint: `golangci-lint run ./...`. When it fails for versionning reasons, fallback to `docker run -t --rm -v $(pwd):/app:z -w /app golangci/golangci-lint:v2.12.2 golangci-lint run ./...`

## Important Notes
- Never work in the `main` branch
- Never commit to `main` branch
- Never commit. Let the user commit their own changes. But suggest a commit message by showing the full `git commit` command.
- Configuration errors cause immediate exit with os.Exit(1)
