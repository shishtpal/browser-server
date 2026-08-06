// Command bs-ai-chat asks the AI stack configured for browser-server a question
// from a terminal. It reuses bs-ai-config.json / bs-ai-models.json /
// bs-ai-mcp.json unchanged, drives the same chat.Service pipeline as the HTTP
// server, and persists every run as a normal conversation in .data/bs-ai.db.
//
// One-shot, non-interactive: tool calls run with --yolo (auto-approve) or are
// disabled with --no-tools. --file inlines file contents into the prompt,
// --image attaches a validated image, and --json emits machine-readable output.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

// options collects every flag. The plain string flags are resolved against the
// loaded config inside runCLI, so an empty value means "use the default".
type options struct {
	provider     string
	model        string
	prompt       string
	files        stringList
	images       stringList
	yolo         bool
	verbose      bool
	json         bool
	tools        string
	noTools      bool
	profile      string
	skills       string
	config       string
	conversation string
	listModels   bool
	listTools    bool
	timeout      time.Duration
}

// stringList implements flag.Value for repeatable string flags (--file, --image).
type stringList []string

func (s *stringList) String() string { return strings.Join(*s, ",") }
func (s *stringList) Set(v string) error {
	*s = append(*s, v)
	return nil
}

func main() {
	os.Exit(run())
}

func run() int {
	var opts options
	flag.StringVar(&opts.provider, "provider", "", "provider name (default: default_provider in bs-ai-config.json)")
	flag.StringVar(&opts.model, "model", "", "model ID (default: the provider's default model)")
	flag.StringVar(&opts.prompt, "prompt", "", "prompt text; also accepted as the positional argument")
	flag.Var(&opts.files, "file", "file to inline into the prompt (repeatable)")
	flag.Var(&opts.images, "image", "image to attach (requires a supports_vision model; repeatable)")
	flag.BoolVar(&opts.yolo, "yolo", false, "auto-approve all tool calls")
	flag.BoolVar(&opts.verbose, "verbose", false, "stream reasoning + tool-call trace to stderr")
	flag.BoolVar(&opts.json, "json", false, "emit one structured JSON object to stdout")
	flag.StringVar(&opts.tools, "tools", "", "comma-separated tool allowlist (default: all configured)")
	flag.BoolVar(&opts.noTools, "no-tools", false, "disable tools entirely")
	flag.StringVar(&opts.profile, "profile", "", "profile name for the system prompt")
	flag.StringVar(&opts.skills, "skills", "", "comma-separated skills to activate")
	flag.StringVar(&opts.config, "config", "", "path to bs-ai-config.json")
	flag.StringVar(&opts.conversation, "conversation", "", "continue an existing conversation ID")
	flag.BoolVar(&opts.listModels, "list-models", false, "print configured providers/models and exit")
	flag.BoolVar(&opts.listTools, "list-tools", false, "print allowed tools (incl. MCP) and exit")
	flag.DurationVar(&opts.timeout, "timeout", 5*time.Minute, "overall run deadline")
	flag.Usage = usage
	flag.Parse()

	opts.prompt = resolvePrompt(opts.prompt, flag.Args())
	return runCLI(opts)
}

// resolvePrompt merges --prompt with positional arguments. Positional args are
// joined with spaces and only used when --prompt is empty.
func resolvePrompt(flagPrompt string, args []string) string {
	if flagPrompt != "" {
		return flagPrompt
	}
	return strings.Join(args, " ")
}

func usage() {
	fmt.Fprint(os.Stderr, `Usage: bs-ai-chat [options] [prompt...]

Asks the configured AI models a question from the terminal. Runs are persisted
as normal conversations in .data/bs-ai.db and appear in the web UI. Reuses the
same bs-ai-config.json / bs-ai-models.json / bs-ai-mcp.json files as the server.

Prompt sources (first match wins):
  1. --prompt
  2. positional arguments
  3. stdin, when piped (bs-ai-chat < answer.md)

Options:
  --provider <name>     Provider name (default: default_provider)
  --model <id>          Model ID (default: the provider's default model)
  --prompt <text>       Prompt text; also accepted as the positional argument
  --file <path>         File to inline into the prompt (repeatable)
  --image <path>        Image to attach; requires a supports_vision model (repeatable)
  --yolo                Auto-approve all tool calls
  --verbose             Stream reasoning + tool-call trace to stderr
  --json                Emit one structured JSON object to stdout
  --tools <list>        Comma-separated tool allowlist (default: all configured)
  --no-tools            Disable tools entirely
  --profile <name>      Profile name for the system prompt
  --skills <list>       Comma-separated skills to activate
  --conversation <id>   Continue an existing conversation ID
  --config <path>       Path to bs-ai-config.json
  --list-models         Print configured providers/models and exit
  --list-tools          Print allowed tools (incl. MCP) and exit
  --timeout <duration>  Overall run deadline (default: 5m)
  -h, --help            Show this help

Config path resolution (first match wins):
  1. --config flag
  2. BS_AI_CONFIG_PATH environment variable
  3. bs-ai-config.json next to the compiled binary
  bs-ai-models.json and bs-ai-mcp.json resolve as siblings automatically.

Tools require --yolo: interactive approval is not yet supported in the CLI.
Use --no-tools to disable them. The assistant's answer goes to stdout; all
trace output goes to stderr, so `+"`bs-ai-chat \"...\" > answer.txt`"+` captures
exactly the answer.
`)
}
