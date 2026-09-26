package share

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

var accessJWT = regexp.MustCompile(`^[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+$`)

func cloudflareHeaders(ctx context.Context, app string, original http.Header) (http.Header, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, "cloudflared", "access", "token", "--app", app)
	// Capture stdout and never expose command output or tokens in error logs.
	output, err := command.Output()
	if err != nil {
		return nil, fmt.Errorf("cannot read Cloudflare Access token; ensure cloudflared is installed and run: cloudflared access login %s", app)
	}
	token := strings.TrimSpace(string(output))
	if !accessJWT.MatchString(token) {
		return nil, fmt.Errorf("cloudflared did not return a valid JWT; run: cloudflared access login %s", app)
	}
	headers := original.Clone()
	if headers == nil {
		headers = make(http.Header)
	}
	// Explicit user authentication takes precedence over service-token variables.
	headers.Del("CF-Access-Client-Id")
	headers.Del("CF-Access-Client-Secret")
	headers.Set("Cf-Access-Token", token)
	return headers, nil
}
