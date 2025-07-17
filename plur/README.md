# Plur

A simple Go CLI application built with [Bubbletea](https://github.com/charmbracelet/bubbletea).

## Installation

```bash
go mod tidy
go build -o plur
```

## Usage

```bash
./plur
```

Use arrow keys or `j`/`k` to navigate, space to select/deselect options, and `q` to quit.

## Development

This project uses:
- [Bubbletea](https://github.com/charmbracelet/bubbletea) for the TUI framework
- [Lipgloss](https://github.com/charmbracelet/lipgloss) for styling

To run in development:

```bash
go run main.go
```