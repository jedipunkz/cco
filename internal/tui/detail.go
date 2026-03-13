package tui

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ansiRe matches ANSI/VT escape sequences produced by PTY output.
var ansiRe = regexp.MustCompile(`\x1b(\[[0-9;?]*[a-zA-Z]|[)(][AB012]|[A-Z\\^_@]|\][^\x07\x1b]*(?:\x07|\x1b\\))`)

func stripANSI(s string) string {
	s = ansiRe.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "")
	return s
}

func detailView(m Model) string {
	visible := visibleAgents(m.agents)
	if len(visible) == 0 || m.cursor >= len(visible) {
		return "No agent selected."
	}

	agent := visible[m.cursor]
	width := clampWidth(m.width)
	innerWidth := width - 4

	statusStr := formatStatus(agent, m)
	elapsed := formatElapsed(agent)

	argsStr := strings.Join(agent.Args, " ")

	var lines []string
	lines = append(lines, fr("╭─ "+agent.ID+" "+strings.Repeat("─", max(0, innerWidth-lipgloss.Width(agent.ID)-2))+"╮"))
	lines = append(lines, fr("│ ")+padRight(fmt.Sprintf("Status : %s", statusStr), innerWidth)+fr(" │"))
	lines = append(lines, fr("│ ")+padRight(fmt.Sprintf("PID    : %d", agent.PID), innerWidth)+fr(" │"))
	lines = append(lines, fr("│ ")+padRight(fmt.Sprintf("Dir    : %s", agent.WorkDir), innerWidth)+fr(" │"))
	lines = append(lines, fr("│ ")+padRight(fmt.Sprintf("Args   : %s", truncate(argsStr, innerWidth-9)), innerWidth)+fr(" │"))
	lines = append(lines, fr("│ ")+padRight(fmt.Sprintf("Started: %s", agent.StartedAt.Format("2006-01-02 15:04:05")), innerWidth)+fr(" │"))
	lines = append(lines, fr("│ ")+padRight(fmt.Sprintf("Elapsed: %s", elapsed), innerWidth)+fr(" │"))
	lines = append(lines, fr("│ ")+padRight("── Recent Output ──", innerWidth)+fr(" │"))

	// Viewport content
	vpLines := strings.Split(m.viewport.View(), "\n")
	for _, l := range vpLines {
		lines = append(lines, fr("│ ")+padRight(l, innerWidth)+fr(" │"))
	}

	lines = append(lines, fr("╰"+strings.Repeat("─", innerWidth+2)+"╯"))

	help := NormalItemStyle.Render("[esc] back  [K] kill")
	lines = append(lines, help)

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// loadLog returns a tea.Cmd that reads the log file and sends a logLoadedMsg.
func loadLog(path string) tea.Cmd {
	return func() tea.Msg {
		data, err := os.ReadFile(path)
		if err != nil {
			return logLoadedMsg{content: fmt.Sprintf("(could not read log: %v)", err)}
		}
		return logLoadedMsg{content: stripANSI(string(data))}
	}
}
