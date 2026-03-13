# cco — Claude Code Orchestrator

> Launch and monitor multiple Claude Code agents from a single terminal.

---

## Overview

`cco` is a CLI tool for running multiple Claude Code agents in parallel and visualising their state in a real-time TUI. It consists of three subcommands:

```
cco agent [-- <claude-args>...]   # Start a Claude Code agent and report its state
cco status                        # Open the TUI dashboard for all agents
cco daemon                        # (hidden) Run the state manager daemon
```

---

## Goals

- Simple commands to launch and track multiple Claude Code processes in parallel
- TUI list + detail view so the operator can see what each agent is doing at a glance
- Know at a glance whether an agent is actively processing or waiting for user input
- Local-first — no external services required
- Distributed as a single Go binary

## Non-Goals

- Controlling Claude Code's input programmatically
- Remote / distributed execution
- Web UI (TUI only)

---

## Architecture

```
┌─────────────────────────────────────────────────────┐
│                    User Terminal                     │
│                                                      │
│   cco agent [args]          cco status              │
│        │                        │                   │
└────────┼────────────────────────┼───────────────────┘
         │                        │
         ▼                        ▼
┌─────────────────┐      ┌────────────────────┐
│  Agent Process  │      │   Status TUI       │
│                 │      │   (bubbletea)      │
│  - fork claude  │      │                   │
│  - PTY monitor  │      │  - list view      │
│  - write state  │      │  - detail view    │
└────────┬────────┘      └────────┬───────────┘
         │                        │
         ▼                        ▼
┌─────────────────────────────────────────────┐
│              State Store (IPC)               │
│                                             │
│  Unix Domain Socket  +  ~/.cco/state.json   │
│                                             │
│  agents:                                    │
│    {id, pid, args, work_dir, status,        │
│     started_at, finished_at, exit_code,     │
│     last_output, log_file, waiting_user}    │
└─────────────────────────────────────────────┘
```

---

## Components

### 1. `cco agent`

**Role**: Start Claude Code as a child process, monitor its output via a PTY, and report lifecycle state to the daemon.

**Flow**:
1. Generate a unique agent ID (`cco-<timestamp>-<rand>`)
2. Create `~/.cco/agents/<id>/` and `output.log`
3. Connect to the state manager daemon (auto-starting it if needed)
4. Start `claude --dangerously-skip-permissions <user-args>` inside a **PTY** so Claude sees a real terminal
5. Put the calling process's stdin in raw mode and forward:
   - `os.Stdin` → PTY master (user keystrokes → Claude)
   - PTY master → `os.Stdout` (Claude output → user's screen)
6. Propagate `SIGWINCH` (terminal resize) to the PTY
7. Monitor byte activity on the PTY master:
   - If no output for **≥ 2 seconds** → `WaitingUser = true` (Claude is at the prompt)
   - When output resumes → `WaitingUser = false` (Claude is processing)
   - State changes are sent to the daemon immediately
8. On process exit, record `exit_code`, `finished_at`, and final status

**Agent State**:
```go
type AgentState struct {
    ID          string     `json:"id"`
    PID         int        `json:"pid"`
    Args        []string   `json:"args"`           // args passed to claude
    WorkDir     string     `json:"work_dir"`
    Status      Status     `json:"status"`          // running | success | failed | killed
    StartedAt   time.Time  `json:"started_at"`
    FinishedAt  *time.Time `json:"finished_at,omitempty"`
    ExitCode    *int       `json:"exit_code,omitempty"`
    LastOutput  string     `json:"last_output"`
    LogFile     string     `json:"log_file"`
    WaitingUser bool       `json:"waiting_user,omitempty"` // true = at prompt, false = processing
}

type Status string
const (
    StatusRunning Status = "running"
    StatusSuccess Status = "success"
    StatusFailed  Status = "failed"
    StatusKilled  Status = "killed"
)
```

**WaitingUser detection**:

Claude Code (a bubbletea TUI) only writes to stdout when it has something to render. While it is actively processing (thinking, running tools, streaming output), it produces a steady stream of bytes. When it finishes and shows the input prompt, stdout goes idle. `cco agent` uses a 2-second idle threshold to distinguish these two states.

---

### 2. State Store

**Role**: Persist agent state and act as an IPC hub between agents and the status TUI.

**Implementation**:
- `~/.cco/state.json` — snapshot of all agent states (survives restarts)
- `~/.cco/cco.sock` — Unix Domain Socket (JSON-lines protocol)
- Agents write updates; the TUI subscribes and receives broadcasts

**Socket protocol** (JSON-lines):
```jsonc
// agent → daemon: state update
{"type": "update", "agent": {...AgentState}}

// daemon → TUI: initial snapshot on subscribe
{"type": "snapshot", "agents": [...AgentState]}

// daemon → TUI: incremental update
{"type": "update", "agent": {...AgentState}}
```

**Daemon auto-start**:
Both `cco agent` and `cco status` call `ensureDaemon()` on startup. If the socket is not reachable, they fork `cco daemon` as a detached background process (new session via `Setsid`) and wait up to 3 seconds for the socket to become available.

---

### 3. `cco status` (TUI)

**Role**: Real-time dashboard for all agents.

**Libraries**: [bubbletea](https://github.com/charmbracelet/bubbletea) + [lipgloss](https://github.com/charmbracelet/lipgloss) + [bubbles](https://github.com/charmbracelet/bubbles)

**List view**:

```
╭─ cco status ──────────────────────────── 2 running ─╮
│                                                      │
│  ┌─ RUNNING ─────────────────────────────────────┐  │
│  │ ▶ cco-1748-ab12  ⏳ waiting you  0:02:31      │  │
│  │   cco-1748-cd34  ⠋ running       0:01:05      │  │
│  └────────────────────────────────────────────────┘  │
│                                                      │
│  ┌─ SUCCESS (recent) ────────────────────────────┐  │
│  │   cco-1748-ef56  ✓ success       0:05:10      │  │
│  └────────────────────────────────────────────────┘  │
│                                                      │
│  ┌─ KILLED (recent) ─────────────────────────────┐  │
│  │   (none)                                       │  │
│  └────────────────────────────────────────────────┘  │
│                                                      │
│  [↑↓/jk] select  [space] detail  [K] kill  [q] quit │
╰──────────────────────────────────────────────────────╯
```

**Status indicators**:

| Symbol | Colour | Meaning |
|--------|--------|---------|
| `⠋ running` | amber | Claude is actively processing |
| `⏳ waiting you` | blue | Claude is at the prompt, waiting for input |
| `✓ success` | green | Process exited 0 |
| `✗ failed` | red | Process exited non-zero |
| `✕ killed` | grey | Process was terminated by signal |

**Visibility rules**:
- All `running` agents are always shown
- `success` and `killed` agents are shown for 5 minutes after finishing
- `failed` agents are not shown (they appear in the killed section if terminated by signal, otherwise hidden)

**Detail view** (press `space`):

```
╭─ cco-1748-ab12 ──────────────────────────────────────╮
│ Status : ⏳ waiting you                               │
│ PID    : 12345                                        │
│ Dir    : ~/projects/myapp                             │
│ Args   : --dangerously-skip-permissions -p "Fix..."   │
│ Started: 2025-03-13 10:22:31                          │
│ Elapsed: 0:02:31                                      │
│ ── Recent Output ──────────────────────────────────── │
│  (scrollable log viewport)                            │
╰───────────────────────────────────────────────────────╯
[esc] back  [K] kill
```

**Key bindings**:

| Key | Action |
|-----|--------|
| `j` / `↓` | Move cursor down |
| `k` / `↑` | Move cursor up |
| `space` | Open detail view |
| `esc` / `q` | Back to list (in detail) or quit |
| `K` | Send SIGTERM to selected running agent |

**TUI Model**:
```go
type Model struct {
    agents     []AgentState
    cursor     int
    view       ViewMode  // viewList | viewDetail
    client     *store.Client
    sub        chan store.Message
    spinner    spinner.Model
    viewport   viewport.Model   // log scroll in detail view
    width      int
    height     int
    logContent string
}
```

**Real-time updates**:
A background `tea.Cmd` blocks on the Unix socket channel and delivers `agentUpdateMsg` to the bubbletea loop whenever the daemon broadcasts a state change.

---

## Directory Structure

```
cco/
├── main.go
├── cmd/
│   ├── root.go          # cobra root command
│   ├── agent.go         # cco agent + daemon auto-start helpers
│   ├── status.go        # cco status (TUI entry point)
│   └── daemon.go        # cco daemon (hidden, run by ensureDaemon)
├── internal/
│   ├── agent/
│   │   ├── runner.go    # PTY launch, stdin/stdout forwarding, WaitingUser detection
│   │   └── parser.go    # JSON line parser (utility)
│   ├── store/
│   │   ├── manager.go   # state manager (socket server + broadcaster)
│   │   ├── state.go     # AgentState type definitions
│   │   └── client.go    # socket client (used by both agent and TUI)
│   └── tui/
│       ├── run.go       # bubbletea program entry point
│       ├── model.go     # bubbletea Model + Update logic
│       ├── list.go      # list view render
│       ├── detail.go    # detail view render + log loading
│       └── styles.go    # lipgloss style definitions
├── go.mod
└── go.sum
```

---

## Key Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/spf13/cobra` | CLI subcommands |
| `github.com/charmbracelet/bubbletea` | TUI framework |
| `github.com/charmbracelet/lipgloss` | TUI styling |
| `github.com/charmbracelet/bubbles` | spinner / viewport |
| `github.com/creack/pty` | PTY allocation for output monitoring |
| `golang.org/x/term` | Raw mode for stdin forwarding |

---

## Runtime File Layout

```
~/.cco/
├── cco.sock              # Unix Domain Socket
├── state.json            # Agent state snapshot (all agents)
└── agents/
    └── cco-1748-ab12/
        └── output.log    # Full output log
```

---

## CLI Reference

```bash
# Start an agent (all args after -- are forwarded to claude)
cco agent
cco agent -- -p "Fix the authentication bug"
cco agent -- --model claude-opus-4 -p "Refactor the database layer"

# Open the status TUI
cco status
```

`--dangerously-skip-permissions` is always prepended to the claude arguments so that agents run without interactive permission prompts.

---

## Error Handling

- **`claude` not found**: `exec.Command` returns an error at start time
- **Daemon not reachable**: `ensureDaemon` forks a new daemon process; retries for 3 seconds before giving up
- **Agent killed by signal**: detected via `syscall.WaitStatus.Signaled()` in `cmd.Wait()`, status set to `killed`
- **PTY errors**: PTY read loop exits on any read error (including `EIO` when the slave side closes), then falls through to `cmd.Wait()`

---

## Future Considerations

- `--worktree` flag: bind an agent to a git worktree and show it in the TUI
- `cco run <taskfile>`: launch multiple agents from a task file
- `cco logs <agent-id>`: stream full log to stdout
- `cco clean`: remove finished agent records older than N minutes
- Agent-to-agent dependency graph (DAG execution)
