package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/skaji/llm-session-share/internal/share"
)

func main() {
	var opts share.Options
	flag.StringVar(&opts.Server, "server", os.Getenv("LLM_SESSION_SHARE_URL"), "server URL (or LLM_SESSION_SHARE_URL)")
	flag.StringVar(&opts.Name, "user", os.Getenv("LLM_SESSION_SHARE_USER"), "display name (or LLM_SESSION_SHARE_USER)")
	flag.StringVar(&opts.Source, "source", "auto", "session format: auto, codex, claude")
	flag.DurationVar(&opts.Interval, "interval", time.Second, "file check and retry interval")
	flag.BoolVar(&opts.Once, "once", false, "upload one snapshot and exit")
	flag.BoolVar(&opts.Delete, "delete", false, "delete the shared session and exit; keep the local JSONL")
	flag.BoolVar(&opts.CloudflareAccess, "cloudflare-access", false, "read a user token with cloudflared access token --app <server-url> before uploads")
	flag.Usage = func() {
		_, _ = fmt.Fprintln(flag.CommandLine.Output(), "Usage: llm-session-share [options] <session.jsonl | codex://threads/UUID | claude://resume?session=UUID>\n\nShare user and assistant messages; watch for updates until interrupted.\nCodex links search $CODEX_HOME (default ~/.codex) for the local JSONL.\nClaude links search $CLAUDE_CONFIG_DIR (default ~/.claude) under projects.\nFor Codex paths under <home>/sessions or <home>/archived_sessions,\nalso read <home>/session_index.jsonl for the title.")
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(2)
	}
	opts.Path = flag.Arg(0)
	opts.Headers = make(http.Header)
	// These credentials belong to the upstream proxy, not the application.
	for _, pair := range [][2]string{
		{"CF_ACCESS_CLIENT_ID", "CF-Access-Client-Id"},
		{"CF_ACCESS_CLIENT_SECRET", "CF-Access-Client-Secret"},
	} {
		if value := os.Getenv(pair[0]); value != "" {
			opts.Headers.Set(pair[1], value)
		}
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := share.Run(ctx, opts, os.Stdout, os.Stderr); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
