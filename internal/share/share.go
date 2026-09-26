package share

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/skaji/llm-session-share/internal/session"
)

type Options struct {
	Path             string
	Server           string
	Name             string
	Source           string
	Interval         time.Duration
	Once             bool
	Delete           bool
	Headers          http.Header
	CloudflareAccess bool
}

func Run(ctx context.Context, opts Options, stdout, stderr io.Writer) error {
	base, err := url.Parse(strings.TrimRight(opts.Server, "/"))
	if err != nil || base == nil || base.Host == "" || (base.Scheme != "http" && base.Scheme != "https") || base.RawQuery != "" || base.Fragment != "" || base.User != nil {
		return fmt.Errorf("-server must be an http(s) URL without credentials, query, or fragment")
	}
	if !opts.Delete && strings.TrimSpace(opts.Name) == "" {
		return fmt.Errorf("set -user or LLM_SESSION_SHARE_USER to your display name")
	}
	if !opts.Delete && opts.Interval <= 0 {
		return fmt.Errorf("-interval must be positive")
	}
	if opts.Source != "auto" && opts.Source != "codex" && opts.Source != "claude" {
		return fmt.Errorf("-source must be auto, codex, or claude")
	}
	if strings.HasPrefix(opts.Path, "codex:") && opts.Source == "claude" {
		return fmt.Errorf("cannot use Codex thread links with -source claude")
	}
	if strings.HasPrefix(opts.Path, "claude:") {
		if opts.Source == "codex" {
			return fmt.Errorf("cannot use Claude session links with -source codex")
		}
		opts.Source = "claude"
	}
	opts.Path, err = session.ResolvePath(opts.Path)
	if err != nil {
		return err
	}
	if _, err := os.Stat(opts.Path); err != nil {
		return err
	}
	client := &http.Client{
		Timeout: 30 * time.Second,
		// An Access login redirect is not a successful upload. Also keep service
		// credentials from being forwarded to a different endpoint.
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse },
	}
	if opts.Delete {
		file, err := os.Open(opts.Path)
		if err != nil {
			return err
		}
		snapshot, err := session.Read(file, opts.Path, "", opts.Source)
		_ = file.Close()
		if err != nil {
			return err
		}
		headers := opts.Headers
		if opts.CloudflareAccess {
			headers, err = cloudflareHeaders(ctx, base.String(), headers)
			if err != nil {
				return err
			}
		}
		if err := send(ctx, client, http.MethodDelete, base.String()+"/api/sessions/"+snapshot.ID, nil, headers); err != nil {
			return err
		}
		_, _ = fmt.Fprintln(stdout, "Deleted: "+base.String()+"/session/"+snapshot.ID)
		return nil
	}
	var lastHash [32]byte
	var sent bool
	var previous os.FileInfo
	var snapshot session.Session
	var body []byte
	timer := time.NewTicker(opts.Interval)
	defer timer.Stop()
	for {
		info, readErr := os.Stat(opts.Path)
		if readErr == nil && (previous == nil || !os.SameFile(previous, info) || previous.Size() != info.Size() || !previous.ModTime().Equal(info.ModTime())) {
			var file *os.File
			file, readErr = os.Open(opts.Path)
			if readErr == nil {
				snapshot, readErr = session.Read(file, opts.Path, strings.TrimSpace(opts.Name), opts.Source)
				_ = file.Close()
			}
			if readErr == nil {
				previous = info
			}
		}
		if readErr == nil {
			if snapshot.Source == "codex" {
				snapshot.Title = session.CodexTitle(opts.Path, snapshot.ID)
			}
			body, readErr = json.Marshal(snapshot)
		}
		if readErr == nil {
			hash := sha256.Sum256(body)
			if !sent || hash != lastHash {
				headers := opts.Headers
				if opts.CloudflareAccess {
					headers, readErr = cloudflareHeaders(ctx, base.String(), headers)
				}
				if readErr == nil {
					readErr = send(ctx, client, http.MethodPut, base.String()+"/api/sessions/"+snapshot.ID, body, headers)
				}
				if readErr == nil {
					if !sent {
						_, _ = fmt.Fprintln(stdout, "Sharing: "+base.String()+"/session/"+snapshot.ID+"?markdown=1&live=1")
					}
					sent, lastHash = true, hash
					_, _ = fmt.Fprintf(stderr, "Shared %d messages (%s, %s)\n", len(snapshot.Messages), snapshot.Source, snapshot.Name)
				}
			}
		}
		if opts.Once {
			return readErr
		}
		if readErr != nil && ctx.Err() == nil {
			_, _ = fmt.Fprintf(stderr, "Retrying: %v\n", readErr)
		}
		select {
		case <-ctx.Done():
			return nil
		case <-timer.C:
		}
	}
}

func send(ctx context.Context, client *http.Client, method, endpoint string, body []byte, headers http.Header) error {
	request, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header = headers.Clone()
	if request.Header == nil {
		request.Header = make(http.Header)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusNoContent {
		return fmt.Errorf("%s returned HTTP %d; expected 204 (check the URL and any upstream access policy)", method, response.StatusCode)
	}
	return nil
}
