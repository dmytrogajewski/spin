package executor

import "regexp"

// ServerPatternCount is the expected number of compiled server patterns.
const ServerPatternCount = 16

// ServerPatterns contains compiled regexps that match server/watcher commands.
// Each pattern matches the full command string (Program + " " + Args joined).
var ServerPatterns = compileServerPatterns()

func compileServerPatterns() []*regexp.Regexp {
	rawPatterns := []string{
		// Node.js package managers: dev/start/serve subcommands.
		`^(npm|yarn|pnpm)\s+(run\s+)?(dev|start|serve)`,
		// Next.js dev server.
		`^npx\s+next\s+dev`,
		// Vite dev server.
		`^(npx\s+)?vite`,
		// Webpack dev server.
		`^(npx\s+)?webpack\s+serve`,
		// Flask dev server.
		`^flask\s+run`,
		// Uvicorn ASGI server.
		`^uvicorn\s+`,
		// Gunicorn WSGI server.
		`^gunicorn\s+`,
		// Rails server.
		`^(rails|bundle\s+exec\s+rails)\s+server`,
		// PHP artisan serve.
		`^php\s+artisan\s+serve`,
		// Go run (likely a server).
		`^go\s+run\s+`,
		// Cargo run (likely a server).
		`^cargo\s+run`,
		// Air (Go live reload).
		`^air\b`,
		// Nodemon (Node.js watcher).
		`^nodemon\s+`,
		// Docker compose up.
		`^docker\s+compose\s+up`,
		// Hugo server.
		`^hugo\s+server`,
		// Jekyll serve.
		`^(bundle\s+exec\s+)?jekyll\s+serve`,
	}

	patterns := make([]*regexp.Regexp, 0, ServerPatternCount)

	for _, raw := range rawPatterns {
		patterns = append(patterns, regexp.MustCompile(raw))
	}

	return patterns
}
