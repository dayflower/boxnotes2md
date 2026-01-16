# Agent Instructions

## Project Overview
This repository contains a Go CLI that converts Box Notes JSON files into GitHub Flavored Markdown using a custom ProseMirror renderer.

## Key Commands
- Build: `go build ./...`
- Run (stdin): `cat examples/example.boxnote | go run .`
- Run (files): `go run . examples/example.boxnote`

## Behavior Notes
- The CLI writes output files next to inputs with a `.md` extension when file arguments are provided.
- When file arguments are used, the output is prefixed with an H1 title derived from the input filename.
- Unsupported ProseMirror nodes are rendered by recursively rendering their children.

## Testing
- No tests are currently defined.
