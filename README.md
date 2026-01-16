# boxnotes2md

Convert Box Notes JSON files into GitHub Flavored Markdown (GFM) using a custom ProseMirror renderer written in Go.

## Features

- Reads Box Notes JSON (`.boxnote`) files and renders GFM.
- Ignores visual-only marks: `author_id`, `font_size`, `font_color`, `highlight`.
- Supports headings, lists, task lists, blockquotes, tables, and inline marks.
- CLI works with stdin/stdout or file arguments.

## Install

### Homebrew

```bash
brew install dayflower/tap/boxnotes2md
```

### From GitHub Releases

Download the appropriate archive for your OS/arch from GitHub Releases and place the
`boxnotes2md` binary in your `PATH`.

### From Source

Requirements: Go 1.20+ (module mode)

```bash
go build -o boxnotes2md .
```

## Usage

### Stdin to stdout

```bash
cat examples/example.boxnote | boxnotes2md
```

If stdin is empty (only whitespace), the command exits successfully without output.

### Files to Markdown outputs

```bash
boxnotes2md examples/example.boxnote
```

When file arguments are provided, an output file is written next to each input:

- `examples/example.boxnote` -> `examples/example.md`

The rendered Markdown is prefixed with an H1 title derived from the input filename (without `.boxnote`).

### Multiple files

```bash
boxnotes2md examples/example.boxnote examples/another.boxnote
```

Each file is processed independently. Progress and errors are written to stderr:

- `OK: <path>` on success.
- `ERROR: <path>: <message>` on failure.

The command exits with status 0 when all files succeed, or 1 if any file fails.

### Overwrite behavior

If the output file already exists, the CLI prompts before overwriting:

```
overwrite examples/example.md? [y/N]:
```

Use `-f` to force overwrite without prompting.

```bash
boxnotes2md -f examples/example.boxnote
```

## Input Format

Box Notes JSON files contain a ProseMirror document under `doc`. The renderer walks this tree and emits Markdown.

## Supported Nodes

- `doc`, `heading`, `paragraph`, `text`, `hard_break`
- `bullet_list`, `ordered_list`, `list_item`
- `check_list`, `check_list_item`
- `horizontal_rule`, `blockquote`, `call_out_box`
- `table`, `table_row`, `table_header`, `table_cell`

Unsupported nodes are rendered by recursively rendering their children.

## Supported Marks

- `link`, `strong`, `em`, `underline`, `strikethrough`, `code`

Ignored marks:

- `author_id`, `font_size`, `font_color`, `highlight`

## Notes

- Inline code fences expand as needed when backticks are present in text.

## License

See `LICENSE`.
