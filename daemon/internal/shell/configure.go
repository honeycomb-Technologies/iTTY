package shell

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	// markerStart and markerEnd delimit the iTTY auto-wrap block in rc files.
	markerStart = "# >>> iTTY auto-session >>>"
	markerEnd   = "# <<< iTTY auto-session <<<"

	// bashZshSnippet is injected into .bashrc/.zshrc to auto-wrap terminals in tmux.
	// Guards:
	//   - $TMUX: prevents nesting (already inside tmux)
	//   - $ITTY_NOAUTO: user escape hatch to disable
	//   - $TERM_PROGRAM=vscode: skip VS Code integrated terminal
	//   - $INSIDE_EMACS: skip Emacs shell
	bashZshSnippet = `if [ -z "$TMUX" ] && [ -z "$ITTY_NOAUTO" ] && [ "$TERM_PROGRAM" != "vscode" ] && [ -z "$INSIDE_EMACS" ]; then
  exec tmux new-session -A -s "itty-$(basename "$(tty)" | tr / -)"
fi`

	// fishSnippet is the equivalent for fish shell.
	fishSnippet = `if not set -q TMUX; and not set -q ITTY_NOAUTO; and test "$TERM_PROGRAM" != "vscode"; and not set -q INSIDE_EMACS
  exec tmux new-session -A -s "itty-"(basename (tty) | tr / -)
end`
)

// Configure adds the iTTY auto-wrap snippet to the user's shell rc file.
// It is idempotent — calling it multiple times does not duplicate the snippet.
func Configure(info *ShellInfo) error {
	snippet := snippetFor(info.Type)
	if snippet == "" {
		return fmt.Errorf("no auto-wrap snippet for shell type %q", info.Type)
	}

	block := fmt.Sprintf("%s\n%s\n%s\n", markerStart, snippet, markerEnd)

	// Read existing rc file
	content, err := os.ReadFile(info.RCFile)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("reading %s: %w", info.RCFile, err)
	}

	existing := string(content)

	// Already configured? Remove old block first (idempotent update)
	if strings.Contains(existing, markerStart) {
		existing = removeBlock(existing)
	}

	// Append the block
	if !strings.HasSuffix(existing, "\n") && len(existing) > 0 {
		existing += "\n"
	}
	existing += "\n" + block

	if err := os.MkdirAll(filepath.Dir(info.RCFile), 0o755); err != nil {
		return fmt.Errorf("creating parent dir for %s: %w", info.RCFile, err)
	}

	if err := os.WriteFile(info.RCFile, []byte(existing), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", info.RCFile, err)
	}

	return nil
}

// Unconfigure removes the iTTY auto-wrap snippet from the user's shell rc file.
func Unconfigure(info *ShellInfo) error {
	content, err := os.ReadFile(info.RCFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // nothing to remove
		}
		return fmt.Errorf("reading %s: %w", info.RCFile, err)
	}

	existing := string(content)
	if !strings.Contains(existing, markerStart) {
		return nil // not configured
	}

	cleaned := removeBlock(existing)
	if err := os.WriteFile(info.RCFile, []byte(cleaned), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", info.RCFile, err)
	}

	return nil
}

// IsConfigured checks whether the iTTY auto-wrap is present in the rc file.
func IsConfigured(info *ShellInfo) bool {
	content, err := os.ReadFile(info.RCFile)
	if err != nil {
		return false
	}
	return strings.Contains(string(content), markerStart)
}

// snippetFor returns the appropriate auto-wrap snippet for the given shell type.
func snippetFor(st ShellType) string {
	switch st {
	case Bash, Zsh:
		return bashZshSnippet
	case Fish:
		return fishSnippet
	default:
		return ""
	}
}

// removeBlock removes the iTTY marker block from content.
func removeBlock(content string) string {
	startIdx := strings.Index(content, markerStart)
	endIdx := strings.Index(content, markerEnd)
	if startIdx == -1 || endIdx == -1 {
		return content
	}

	endIdx += len(markerEnd)
	// Also consume trailing newline
	if endIdx < len(content) && content[endIdx] == '\n' {
		endIdx++
	}
	// Also consume leading newline before marker
	if startIdx > 0 && content[startIdx-1] == '\n' {
		startIdx--
	}

	return content[:startIdx] + content[endIdx:]
}
