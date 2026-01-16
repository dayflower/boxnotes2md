# Implementation Plan

## Goal
Convert Box Notes JSON files to Markdown using a custom ProseMirror JSON-to-Markdown transformer implemented in Go, excluding specific mark types.

## Inputs and Scope
- Input: Box Notes JSON file (see `examples/example.boxnote`).
- ProseMirror content is under `root.doc` in the Box Notes JSON.
- Exclude mark types from Markdown conversion: `author_id`, `font_size`, `font_color`.
- No tests are required.

## Observed ProseMirror Types in the Example
Source: `examples/example.boxnote`.

### Node Types
- `doc`
- `heading` (attrs: `level`, `guid`, `collapsedUserIds`)
- `paragraph`
- `text`
- `hard_break`
- `bullet_list`
- `ordered_list` (attrs: `order`)
- `list_item`
- `check_list`
- `check_list_item` (attrs: `checked`)
- `horizontal_rule`
- `call_out_box` (attrs: `backgroundColor`, `emoji`)
- `blockquote`
- `table`
- `table_row`
- `table_header` (attrs: `colspan`, `rowspan`, `colwidth`, `background`)
- `table_cell` (attrs: `colspan`, `rowspan`, `colwidth`, `background`)

### Mark Types
- `author_id` (ignored)
- `alignment` (attrs: `alignment`, applied at paragraph level)
- `link` (attrs: `href`)
- `strong`
- `em`
- `underline`
- `strikethrough`
- `code`
- `font_size` (ignored)
- `font_color` (ignored)
- `highlight` (attrs: `color`)

## Proposed GFM Conversion Rules (Draft)
Target format: GitHub Flavored Markdown (GFM).

### Blocks and Structure
- `doc`: render children in order, separated by blank lines where appropriate.
- `heading`: `#`..`######` based on `attrs.level` (1-6). If level out of range, clamp to 1-6.
- `paragraph`: render inline content. Ignore paragraph-level `alignment` marks and preserve text only.
- `hard_break`: render as a hard line break using trailing `\\` and a newline.
- `horizontal_rule`: render `---`.
- `blockquote`: render with `> ` prefix per line; render child paragraphs inside.
- `call_out_box`: render as a plain blockquote, ignoring `backgroundColor` and `emoji`.
- `bullet_list`: render with `- ` per item; nested lists increase indentation by two spaces.
- `ordered_list`: render with `1. ` per item (let Markdown auto-number). Nesting uses two-space indentation.
- `list_item`: render children; if multiple paragraphs inside, separate with blank lines and keep indentation.
- `check_list`: render as GFM task list with `- [ ]`/`- [x]` prefix from `check_list_item`.
- `check_list_item`: map `attrs.checked` to `- [x]` when true, otherwise `- [ ]`.
- `table`: render as GFM table. Use first row with `table_header` as header; if missing, treat first row as header and convert cells to header. Generate separator row with `---`. Ignore alignment in cells.
- `table_row`: render row cells separated by `|`.
- `table_header`: render cell content as header cell text.
- `table_cell`: render cell content as normal cell text.

### Inline and Marks
- `text`: emit `text` with applied marks (nest in a stable order).
- Mark precedence/order (outer → inner): `link` → `strong` → `em` → `underline` → `strikethrough` → `code`.
  - `link`: `[text](href)`.
  - `strong`: `**text**`.
  - `em`: `*text*` by default; when `strong` and `em` are combined, prefer `**_text_**` to avoid `***` sequences.
  - `underline`: use HTML `<u>text</u>` (allowed).
  - `strikethrough`: `~~text~~`.
  - `code`: `` `text` `` (escape backticks when needed).
- Ignore marks: `author_id`, `font_size`, `font_color`, `highlight`.
  - When these are the only marks, render plain text.
  - If mixed with other marks, ignore only these and apply the rest.
- When applying inline marks (except links), insert zero-width space (`U+200B`) at the start/end of the marked text only if the boundary character is a Japanese punctuation symbol. This reduces emphasis parsing issues around punctuation. Apply this to `strong`, `em`, `strikethrough`, and `code` marks; skip for links.

### Escaping Rules
- Escape `|` inside table cells.
- Escape `[` `]` `(` `)` in link text when needed.
- Escape `*`, `_`, and backticks inside inline code or when they would break formatting.
- For emphasized text, escape `*`, `_`, `~`, and `\` as needed before applying marks to avoid broken emphasis.

### Open Decisions
None.

## High-Level Approach
1. Parse Box Notes JSON and extract the ProseMirror `doc` object.
2. Define Go structures mirroring the subset of ProseMirror nodes/marks needed for the example(s).
3. Implement a Markdown renderer that walks the ProseMirror JSON tree and emits Markdown.
4. Provide a CLI entry point that reads a `.boxnotes` file and outputs Markdown to stdout or a file.

## Detailed Steps
1. **Explore the example input**
   - Inspect `examples/example.boxnotes` to identify node types, marks, and structure used.
   - List the ProseMirror node types to support first (e.g., `doc`, `paragraph`, `text`, `heading`, `bullet_list`, `ordered_list`, `list_item`, etc.).

2. **Define data structures**
   - Create Go structs for Box Notes root JSON and ProseMirror nodes/marks.
   - Use flexible fields for `type`, `attrs`, `content`, and `marks` to support unknown nodes gracefully.
   - Add a generic `Node` and `Mark` structure with optional fields.

3. **Parse and validate**
   - Load the JSON file and decode into the root structure.
   - Extract the `doc` node and ensure it is non-nil and well-formed.

4. **Implement Markdown rendering**
   - Build a recursive renderer: `renderNode(node) -> string`.
   - Implement handlers for each supported node type.
   - Implement mark handling for inline nodes (e.g., bold, italic, code, link).
   - Explicitly ignore marks `author_id`, `font_size`, `font_color` during rendering.
   - Add a fallback behavior for unsupported node types (e.g., render children only).

5. **CLI interface**
   - Keep stdin/stdout conversion when no positional arguments are provided.
   - When positional arguments are provided, treat each argument as an input file.
   - For each input file, read JSON, render Markdown, and write to a new file.
   - Output file naming: if input ends with `.boxnote`, remove it, then append `.md`; otherwise append `.md` directly.
   - When processing files via arguments, prepend an H1 header using the input filename (with `.boxnote` removed) before the rendered content.
   - Return non-zero exit code on parse/render errors with a clear message (include the failing filename).

6. **Usage documentation**
   - Update `README.md` with examples for stdin/stdout and file-argument usage.

## Milestones
- M1: JSON parsing and `doc` extraction works on the sample file.
- M2: Markdown renderer produces readable output for sample content.
- M3: CLI outputs Markdown and handles errors gracefully.

## Open Questions
- Confirm which ProseMirror node/mark types must be supported beyond the example file.
- Decide if output should preserve line wrapping or emit a single-paragraph format.
