package extract

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"sync"
	"time"
)

const pdfToolName = "pdftotext"

var ErrPDFToolMissing = errors.New("extract: pdftotext not available")

var (
	pdfToolOnce sync.Once
	pdfToolPath string
)

func pdfTool() (string, error) {
	pdfToolOnce.Do(func() {
		p, err := exec.LookPath(pdfToolName)
		if err != nil {
			slog.Warn("extract: pdftotext not found; PDF URLs will keep returning unsupported_content_type",
				"tool", pdfToolName)
			return
		}
		pdfToolPath = p
	})
	if pdfToolPath == "" {
		return "", ErrPDFToolMissing
	}
	return pdfToolPath, nil
}

func isPDFContentType(ct string) bool {
	ct = strings.ToLower(strings.TrimSpace(ct))
	return strings.HasPrefix(ct, "application/pdf") || strings.HasPrefix(ct, "application/x-pdf")
}

func pdfToText(ctx context.Context, body []byte, timeout time.Duration) (string, error) {
	tool, err := pdfTool()
	if err != nil {
		return "", err
	}
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, tool, "-layout", "-nopgbrk", "-", "-")
	cmd.Stdin = bytes.NewReader(body)
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return "", fmt.Errorf("%w: pdftotext timed out", ErrExtractionFailed)
		}
		return "", fmt.Errorf("%w: pdftotext: %v: %s",
			ErrExtractionFailed, err, strings.TrimSpace(stderr.String()))
	}

	text := strings.TrimSpace(out.String())
	if text == "" {
		return "", fmt.Errorf("%w: pdf has no extractable text layer (scanned?)", ErrExtractionFailed)
	}
	return text, nil
}
