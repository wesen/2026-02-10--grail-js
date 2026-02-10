---
Title: Embed a real PTY into a Bubble Tea TUI pane
Ticket: EXP-001-PTY-BUBBLETEA
Status: active
Topics:
    - pty
    - bubbletea
    - tui
    - go
    - experiment
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: sends
      Note: q
    - Path: ttmp/2026/02/10/EXP-001-PTY-BUBBLETEA--embed-a-real-pty-into-a-bubble-tea-tui-pane/design/01-pty-in-bubbletea-design.md
      Note: Architecture and design doc
    - Path: ttmp/2026/02/10/EXP-001-PTY-BUBBLETEA--embed-a-real-pty-into-a-bubble-tea-tui-pane/scripts/pty-bubbletea/go.mod
      Note: Go module definition
    - Path: ttmp/2026/02/10/EXP-001-PTY-BUBBLETEA--embed-a-real-pty-into-a-bubble-tea-tui-pane/scripts/pty-bubbletea/main.go
      Note: Main Go source - PTY+vt10x+BubbleTea integration
    - Path: ttmp/2026/02/10/EXP-001-PTY-BUBBLETEA--embed-a-real-pty-into-a-bubble-tea-tui-pane/scripts/test-in-tmux.sh
      Note: Tmux test - launches vi
    - Path: ttmp/2026/02/10/EXP-001-PTY-BUBBLETEA--embed-a-real-pty-into-a-bubble-tea-tui-pane/scripts/test-insert-mode.sh
      Note: Tmux test - insert mode
ExternalSources: []
Summary: ""
LastUpdated: 2026-02-10T10:49:39.997993025-05:00
WhatFor: ""
WhenToUse: ""
---



# Embed a real PTY into a Bubble Tea TUI pane

## Overview

<!-- Provide a brief overview of the ticket, its goals, and current status -->

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- pty
- bubbletea
- tui
- go
- experiment

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Structure

- design/ - Architecture and design documents
- reference/ - Prompt packs, API contracts, context summaries
- playbooks/ - Command sequences and test procedures
- scripts/ - Temporary code and tooling
- various/ - Working notes and research
- archive/ - Deprecated or reference-only artifacts
