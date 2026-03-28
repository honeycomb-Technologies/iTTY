package tmux

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseSessionLine(t *testing.T) {
	session, err := parseSessionLine("itty-pts-1|2|1711644000|1|nvim|/tmp/project")
	if err != nil {
		t.Fatalf("parseSessionLine() error = %v", err)
	}

	if session.Name != "itty-pts-1" || session.Windows != 2 || !session.Attached {
		t.Fatalf("unexpected session: %#v", session)
	}
	if session.LastPaneCmd != "nvim" || session.LastPanePath != "/tmp/project" {
		t.Fatalf("unexpected session metadata: %#v", session)
	}
}

func TestParseSessionLineRejectsMalformedInput(t *testing.T) {
	_, err := parseSessionLine("broken|not-a-number|oops")
	if err == nil {
		t.Fatal("parseSessionLine() error = nil, want malformed input error")
	}
}

func TestParseWindowLineRejectsMalformedInput(t *testing.T) {
	_, err := parseWindowLine("editor|1")
	if err == nil {
		t.Fatal("parseWindowLine() error = nil, want malformed input error")
	}
}

func TestParsePaneLineRejectsMalformedInput(t *testing.T) {
	_, err := parsePaneLine("%1|0|1|bash|/tmp|wide")
	if err == nil {
		t.Fatal("parsePaneLine() error = nil, want malformed input error")
	}
}

func TestGetSessionReturnsNotFoundSentinel(t *testing.T) {
	client := NewClient()
	client.TmuxPath = writeFakeTmux(t, `#!/bin/sh
if [ "$1" = "display-message" ] && [ "$4" = "missing" ]; then
  echo "can't find session" >&2
  exit 1
fi
echo "unexpected args: $@" >&2
exit 1
`)

	_, err := client.GetSession(context.Background(), "missing")
	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("GetSession() error = %v, want ErrSessionNotFound", err)
	}
}

func TestGetSessionParsesSessionDetail(t *testing.T) {
	client := NewClient()
	client.TmuxPath = writeFakeTmux(t, `#!/bin/sh
case "$1" in
  display-message)
    echo "itty-pts-1|2|1711644000|1|nvim|/tmp/project"
    ;;
  list-windows)
    echo "0|editor|1"
    echo "1|shell|0"
    ;;
  list-panes)
    if [ "$3" = "itty-pts-1:0" ]; then
      echo "%0|0|1|nvim|/tmp/project|120|40"
    else
      echo "%1|0|1|bash|/tmp/project|120|40"
    fi
    ;;
  *)
    echo "unexpected args: $@" >&2
    exit 1
    ;;
esac
`)

	detail, err := client.GetSession(context.Background(), "itty-pts-1")
	if err != nil {
		t.Fatalf("GetSession() error = %v", err)
	}

	if detail.Name != "itty-pts-1" || detail.Windows != 2 {
		t.Fatalf("unexpected session detail: %#v", detail)
	}
	if len(detail.WindowList) != 2 {
		t.Fatalf("len(WindowList) = %d, want 2", len(detail.WindowList))
	}
	if len(detail.WindowList[0].Panes) != 1 {
		t.Fatalf("len(WindowList[0].Panes) = %d, want 1", len(detail.WindowList[0].Panes))
	}
	if detail.Created != time.Unix(1711644000, 0) {
		t.Fatalf("Created = %v, want %v", detail.Created, time.Unix(1711644000, 0))
	}
}

func writeFakeTmux(t *testing.T, script string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "tmux")
	if err := os.WriteFile(path, []byte(strings.TrimSpace(script)+"\n"), 0o755); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return path
}
