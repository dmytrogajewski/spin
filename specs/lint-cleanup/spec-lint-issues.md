# Lint Issues Specification

**Date:** 2026-03-12
**Source:** `make lint` (golangci-lint v2, `default: all`)
**Total issues:** 25,378

## Issue Distribution (sorted by count)

| Linter | Count | Category |
|--------|-------|----------|
| godot | 7,946 | Style: comment periods |
| wsl_v5 | 4,196 | Style: whitespace |
| paralleltest | 2,636 | Testing: parallel |
| nlreturn | 1,267 | Style: newlines before return |
| errcheck | 1,239 | Correctness: unchecked errors |
| revive | 772 | Style: mixed rules |
| modernize | 657 | Style: use modern Go idioms |
| noinlineerr | 588 | Style: no inline error |
| err113 | 505 | Correctness: error wrapping |
| mnd | 478 | Style: magic numbers |
| sloglint | 412 | Logging: slog usage |
| gocritic | 387 | Style: code suggestions |
| intrange | 348 | Style: integer ranges |
| varnamelen | 344 | Style: variable name length |
| govet | 336 | Correctness: vet checks |
| staticcheck | 306 | Correctness: static analysis |
| gosec | 256 | Security: potential vulnerabilities |
| perfsprint | 251 | Performance: string formatting |
| testpackage | 217 | Testing: separate test packages |
| testifylint | 216 | Testing: testify best practices |
| tagalign | 144 | Style: struct tag alignment |
| lll | 143 | Style: long lines |
| gocognit | 135 | Complexity: cognitive |
| goimports | 134 | Formatting: imports |
| funlen | 117 | Complexity: function length |
| misspell | 107 | Style: spelling |
| dupl | 107 | Style: code duplication |
| wrapcheck | 92 | Correctness: error wrapping |
| forbidigo | 86 | Style: forbidden calls (fmt.Print) |
| ireturn | 69 | Style: interface returns |
| errorlint | 69 | Correctness: error handling |
| nestif | 65 | Complexity: nested ifs |
| goconst | 57 | Style: constants |
| cyclop | 53 | Complexity: cyclomatic |
| gocyclo | 47 | Complexity: cyclomatic |
| godoclint | 45 | Docs: doc comments |
| gofmt | 49 | Formatting |
| unparam | 46 | Correctness: unused params |
| gochecknoglobals | 37 | Style: globals |
| exhaustive | 37 | Correctness: exhaustive switches |
| noctx | 32 | Correctness: HTTP without context |
| reassign | 28 | Style: reassignment |
| usetesting | 25 | Testing: use testing helpers |
| iface | 24 | Style: interface suggestions |
| thelper | 18 | Testing: test helpers |
| gosmopolitan | 17 | Style: i18n |
| usestdlibvars | 17 | Style: stdlib vars |
| tagliatelle | 15 | Style: tag naming |
| unused | 15 | Correctness: unused code |
| predeclared | 14 | Style: predeclared identifiers |
| contextcheck | 14 | Correctness: context propagation |
| mirror | 11 | Style: mirror functions |
| wastedassign | 10 | Correctness: wasted assignments |
| inamedparam | 8 | Style: named params |
| godot | 7 | Style: comment periods |
| forcetypeassert | 7 | Correctness: force type assert |
| unconvert | 7 | Correctness: unnecessary convert |
| ineffassign | 7 | Correctness: ineffective assign |
| interfacebloat | 6 | Style: large interfaces |
| nilnil | 6 | Correctness: nil,nil returns |
| godox | 5 | Style: TODO/FIXME markers |
| maintidx | 5 | Complexity: maintainability |
| prealloc | 5 | Performance: slice prealloc |
| gochecknoinits | 4 | Style: init functions |
| dogsled | 4 | Style: blank assignments |
| nolintlint | 4 | Style: nolint directives |
| dupword | 3 | Style: duplicate words |
| containedctx | 3 | Correctness: context in struct |
| tparallel | 3 | Testing: parallel |
| fatcontext | 1 | Correctness: context in loop |
| funcorder | 3 | Style: function ordering |
| embeddedstructfieldcheck | 2 | Correctness: embedded struct |
| errchkjson | 2 | Correctness: JSON error checks |
| errname | 1 | Style: error naming |
| musttag | 2 | Correctness: missing tags |
| recvcheck | 2 | Style: receiver consistency |

## Config

- `default: all` with minimal disables
- Strict settings across all enabled linters
- No existing nolint exclusions or per-file ignores
