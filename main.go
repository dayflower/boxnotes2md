package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

type BoxNote struct {
	Doc Node `json:"doc"`
}

type Node struct {
	Type    string                 `json:"type"`
	Attrs   map[string]interface{} `json:"attrs"`
	Content []Node                 `json:"content"`
	Text    string                 `json:"text"`
	Marks   []Mark                 `json:"marks"`
}

type Mark struct {
	Type  string                 `json:"type"`
	Attrs map[string]interface{} `json:"attrs"`
}

type RenderContext struct {
	Indent int
}

func main() {
	forceOverwrite := flag.Bool("f", false, "overwrite output files without prompting")
	flag.Parse()
	args := flag.Args()

	if len(args) == 0 {
		input, err := io.ReadAll(os.Stdin)
		if err != nil {
			fatal("failed to read stdin", err)
		}
		if len(strings.TrimSpace(string(input))) == 0 {
			return
		}
		output, err := renderBoxNote(input)
		if err != nil {
			fatal(err.Error(), nil)
		}
		fmt.Fprint(os.Stdout, output)
		return
	}

	hadError := false
	for _, inputPath := range args {
		if err := processFile(inputPath, *forceOverwrite); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: %s: %v\n", inputPath, err)
			hadError = true
			continue
		}
		fmt.Fprintf(os.Stderr, "OK: %s\n", inputPath)
	}
	if hadError {
		os.Exit(1)
	}
}

func fatal(message string, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", message, err)
	} else {
		fmt.Fprintf(os.Stderr, "%s\n", message)
	}
	os.Exit(1)
}

func renderBoxNote(input []byte) (string, error) {
	var note BoxNote
	if err := json.Unmarshal(input, &note); err != nil {
		return "", fmt.Errorf("failed to parse JSON")
	}
	if note.Doc.Type == "" {
		return "", fmt.Errorf("missing doc node")
	}
	return renderNode(note.Doc, RenderContext{}), nil
}

func processFile(inputPath string, forceOverwrite bool) error {
	input, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("failed to read: %w", err)
	}

	outputPath := outputPathFor(inputPath)
	if exists(outputPath) && !forceOverwrite {
		confirmed, err := confirmOverwrite(outputPath)
		if err != nil {
			return err
		}
		if !confirmed {
			return fmt.Errorf("overwrite declined")
		}
	}

	if len(strings.TrimSpace(string(input))) == 0 {
		return os.WriteFile(outputPath, []byte(""), 0644)
	}

	output, err := renderBoxNote(input)
	if err != nil {
		return err
	}

	title := titleFromPath(inputPath)
	if title != "" {
		output = "# " + title + "\n\n" + output
	}

	if err := os.WriteFile(outputPath, []byte(output), 0644); err != nil {
		return fmt.Errorf("failed to write: %w", err)
	}
	return nil
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func confirmOverwrite(path string) (bool, error) {
	fmt.Fprintf(os.Stderr, "overwrite %s? [y/N]: ", path)
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return false, fmt.Errorf("failed to read overwrite confirmation: %w", err)
	}
	answer := strings.TrimSpace(strings.ToLower(line))
	return answer == "y" || answer == "yes", nil
}

func outputPathFor(inputPath string) string {
	return strings.TrimSuffix(inputPath, ".boxnote") + ".md"
}

func titleFromPath(inputPath string) string {
	base := filepath.Base(inputPath)
	return strings.TrimSuffix(base, ".boxnote")
}

func renderNode(node Node, ctx RenderContext) string {
	switch node.Type {
	case "doc":
		return renderBlocks(node.Content, ctx)
	default:
		return renderBlocks(node.Content, ctx)
	}
}

func renderBlocks(nodes []Node, ctx RenderContext) string {
	var blocks []string
	for _, node := range nodes {
		block, keep := renderBlock(node, ctx)
		if !keep {
			continue
		}
		blocks = append(blocks, block)
	}
	return strings.Join(blocks, "\n\n")
}

func renderBlock(node Node, ctx RenderContext) (string, bool) {
	switch node.Type {
	case "heading":
		level := clampInt(getIntAttr(node.Attrs, "level"), 1, 6)
		text := renderInline(node.Content)
		return fmt.Sprintf("%s %s", strings.Repeat("#", level), text), true
	case "paragraph":
		if len(node.Content) == 0 {
			return "", true
		}
		return renderInline(node.Content), true
	case "hard_break":
		return "\\\n", true
	case "bullet_list":
		return renderList(node, ctx, "- "), true
	case "ordered_list":
		return renderList(node, ctx, "1. "), true
	case "list_item":
		lines := renderListItem(node, ctx, "- ")
		return strings.Join(lines, "\n"), true
	case "check_list":
		return renderCheckList(node, ctx), true
	case "check_list_item":
		prefix := "- [ ] "
		if getBoolAttr(node.Attrs, "checked") {
			prefix = "- [x] "
		}
		lines := renderListItem(node, ctx, prefix)
		return strings.Join(lines, "\n"), true
	case "horizontal_rule":
		return "---", true
	case "blockquote":
		return renderBlockquote(node.Content, ctx), true
	case "call_out_box":
		return renderBlockquote(node.Content, ctx), true
	case "table":
		return renderTable(node, ctx), true
	default:
		if len(node.Content) == 0 {
			return "", false
		}
		return renderBlocks(node.Content, ctx), true
	}
}

func renderInline(nodes []Node) string {
	var b strings.Builder
	for _, node := range nodes {
		switch node.Type {
		case "text":
			b.WriteString(applyMarks(node.Text, node.Marks))
		case "hard_break":
			b.WriteString("\\\n")
		default:
			if len(node.Content) > 0 {
				b.WriteString(renderInline(node.Content))
			}
		}
	}
	return b.String()
}

func renderList(node Node, ctx RenderContext, prefix string) string {
	var lines []string
	hasItem := false
	for _, item := range node.Content {
		switch item.Type {
		case "list_item":
			lines = append(lines, renderListItem(item, ctx, prefix)...)
			hasItem = true
		case "bullet_list":
			if hasItem {
				nested := renderList(item, RenderContext{Indent: ctx.Indent + 2}, "- ")
				if nested != "" {
					lines = append(lines, strings.Split(nested, "\n")...)
				}
			}
		case "ordered_list":
			if hasItem {
				nested := renderList(item, RenderContext{Indent: ctx.Indent + 2}, "1. ")
				if nested != "" {
					lines = append(lines, strings.Split(nested, "\n")...)
				}
			}
		case "check_list":
			if hasItem {
				nested := renderCheckList(item, RenderContext{Indent: ctx.Indent + 2})
				if nested != "" {
					lines = append(lines, strings.Split(nested, "\n")...)
				}
			}
		}
	}
	return strings.Join(lines, "\n")
}

func renderCheckList(node Node, ctx RenderContext) string {
	var lines []string
	hasItem := false
	for _, item := range node.Content {
		switch item.Type {
		case "check_list_item":
			prefix := "- [ ] "
			if getBoolAttr(item.Attrs, "checked") {
				prefix = "- [x] "
			}
			lines = append(lines, renderListItem(item, ctx, prefix)...)
			hasItem = true
		case "bullet_list":
			if hasItem {
				nested := renderList(item, RenderContext{Indent: ctx.Indent + 2}, "- ")
				if nested != "" {
					lines = append(lines, strings.Split(nested, "\n")...)
				}
			}
		case "ordered_list":
			if hasItem {
				nested := renderList(item, RenderContext{Indent: ctx.Indent + 2}, "1. ")
				if nested != "" {
					lines = append(lines, strings.Split(nested, "\n")...)
				}
			}
		case "check_list":
			if hasItem {
				nested := renderCheckList(item, RenderContext{Indent: ctx.Indent + 2})
				if nested != "" {
					lines = append(lines, strings.Split(nested, "\n")...)
				}
			}
		}
	}
	return strings.Join(lines, "\n")
}

func renderListItem(node Node, ctx RenderContext, prefix string) []string {
	indent := ctx.Indent
	prefixLine := strings.Repeat(" ", indent) + prefix
	children := node.Content
	if len(children) == 0 {
		return []string{prefixLine}
	}

	var lines []string
	first := children[0]
	if first.Type == "paragraph" {
		text := renderInline(first.Content)
		text = indentMultiline(text, len(prefixLine))
		lines = append(lines, prefixLine+text)
		children = children[1:]
	} else {
		lines = append(lines, prefixLine)
	}

	for _, child := range children {
		block, keep := renderBlock(child, RenderContext{Indent: indent + 2})
		if !keep {
			continue
		}
		if block == "" {
			lines = append(lines, strings.Repeat(" ", indent+2))
			continue
		}
		lines = append(lines, indentAllLines(block, indent+2))
	}

	return lines
}

func renderBlockquote(nodes []Node, ctx RenderContext) string {
	content := renderBlocks(nodes, ctx)
	if content == "" {
		return ">"
	}
	return prefixLines(content, "> ")
}

func renderTable(node Node, ctx RenderContext) string {
	var rows [][]string
	for _, row := range node.Content {
		if row.Type != "table_row" {
			continue
		}
		rows = append(rows, renderTableRow(row, ctx))
	}
	if len(rows) == 0 {
		return ""
	}

	colCount := 0
	for _, row := range rows {
		if len(row) > colCount {
			colCount = len(row)
		}
	}
	if colCount == 0 {
		return ""
	}

	header := normalizeRow(rows[0], colCount)
	lines := []string{formatTableRow(header), formatTableSeparator(colCount)}
	for _, row := range rows[1:] {
		lines = append(lines, formatTableRow(normalizeRow(row, colCount)))
	}

	return strings.Join(lines, "\n")
}

func renderTableRow(row Node, ctx RenderContext) []string {
	var cells []string
	for _, cell := range row.Content {
		switch cell.Type {
		case "table_header", "table_cell":
			cells = append(cells, renderTableCell(cell, ctx))
		}
	}
	return cells
}

func renderTableCell(cell Node, ctx RenderContext) string {
	text := renderCellContent(cell.Content, ctx)
	text = strings.ReplaceAll(text, "\n", "<br>")
	text = escapeTableCell(text)
	return text
}

func renderCellContent(nodes []Node, ctx RenderContext) string {
	var parts []string
	for _, node := range nodes {
		switch node.Type {
		case "paragraph":
			if len(node.Content) > 0 {
				parts = append(parts, renderInline(node.Content))
			}
		case "text":
			parts = append(parts, applyMarks(node.Text, node.Marks))
		default:
			if len(node.Content) > 0 {
				parts = append(parts, renderCellContent(node.Content, ctx))
			}
		}
	}
	return strings.Join(parts, "<br>")
}

func applyMarks(text string, marks []Mark) string {
	filtered := filterMarks(marks)
	if len(filtered) == 0 {
		return text
	}

	hasStrong := hasMarkType(filtered, "strong")
	hasEm := hasMarkType(filtered, "em")
	hasStrike := hasMarkType(filtered, "strikethrough")
	hasCode := hasMarkType(filtered, "code")
	hasLink := hasMarkType(filtered, "link")
	emDelimiter := "*"
	if hasStrong && hasEm {
		emDelimiter = "_"
	}
	if !hasCode {
		text = escapeForMarkdown(text, emDelimiter, hasStrong, hasStrike)
	}
	if (hasStrong || hasEm || hasStrike || hasCode) && !hasLink {
		text = padWithZeroWidthSpace(text)
	}

	sort.SliceStable(filtered, func(i, j int) bool {
		return markOrder(filtered[i].Type) < markOrder(filtered[j].Type)
	})

	for i := len(filtered) - 1; i >= 0; i-- {
		mark := filtered[i]
		switch mark.Type {
		case "link":
			href, ok := getStringAttr(mark.Attrs, "href")
			if !ok || href == "" {
				continue
			}
			text = fmt.Sprintf("[%s](%s)", escapeLinkText(text), href)
		case "strong":
			text = "**" + text + "**"
		case "em":
			text = emDelimiter + text + emDelimiter
		case "underline":
			text = "<u>" + text + "</u>"
		case "strikethrough":
			text = "~~" + text + "~~"
		case "code":
			text = wrapInlineCode(text)
		}
	}
	return text
}

func filterMarks(marks []Mark) []Mark {
	var filtered []Mark
	for _, mark := range marks {
		switch mark.Type {
		case "author_id", "font_size", "font_color", "highlight":
			continue
		default:
			filtered = append(filtered, mark)
		}
	}
	return filtered
}

func markOrder(markType string) int {
	switch markType {
	case "link":
		return 0
	case "strong":
		return 1
	case "em":
		return 2
	case "underline":
		return 3
	case "strikethrough":
		return 4
	case "code":
		return 5
	default:
		return 100
	}
}

func wrapInlineCode(text string) string {
	if !strings.Contains(text, "`") {
		return "`" + text + "`"
	}
	max := maxConsecutiveBackticks(text)
	fence := strings.Repeat("`", max+1)
	return fence + text + fence
}

func hasMarkType(marks []Mark, markType string) bool {
	for _, mark := range marks {
		if mark.Type == markType {
			return true
		}
	}
	return false
}

func maxConsecutiveBackticks(text string) int {
	max := 0
	current := 0
	for _, r := range text {
		if r == '`' {
			current++
			if current > max {
				max = current
			}
		} else {
			current = 0
		}
	}
	return max
}

func getIntAttr(attrs map[string]interface{}, key string) int {
	if attrs == nil {
		return 0
	}
	value, ok := attrs[key]
	if !ok {
		return 0
	}
	switch v := value.(type) {
	case float64:
		return int(v)
	case int:
		return v
	case json.Number:
		intValue, err := v.Int64()
		if err == nil {
			return int(intValue)
		}
	}
	return 0
}

func getBoolAttr(attrs map[string]interface{}, key string) bool {
	if attrs == nil {
		return false
	}
	value, ok := attrs[key]
	if !ok {
		return false
	}
	boolValue, ok := value.(bool)
	return ok && boolValue
}

func getStringAttr(attrs map[string]interface{}, key string) (string, bool) {
	if attrs == nil {
		return "", false
	}
	value, ok := attrs[key]
	if !ok {
		return "", false
	}
	stringValue, ok := value.(string)
	return stringValue, ok
}

func clampInt(value, minValue, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func indentMultiline(text string, indent int) string {
	lines := strings.Split(text, "\n")
	if len(lines) == 0 {
		return text
	}
	for i := 1; i < len(lines); i++ {
		lines[i] = strings.Repeat(" ", indent) + lines[i]
	}
	return strings.Join(lines, "\n")
}

func indentAllLines(text string, indent int) string {
	if text == "" {
		return ""
	}
	lines := strings.Split(text, "\n")
	prefix := strings.Repeat(" ", indent)
	for i, line := range lines {
		if line == "" {
			lines[i] = prefix
			continue
		}
		lines[i] = prefix + line
	}
	return strings.Join(lines, "\n")
}

func prefixLines(text, prefix string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if line == "" {
			lines[i] = strings.TrimRight(prefix, " ")
			continue
		}
		lines[i] = prefix + line
	}
	return strings.Join(lines, "\n")
}

func escapeTableCell(text string) string {
	return strings.ReplaceAll(text, "|", "\\|")
}

func escapeLinkText(text string) string {
	replacer := strings.NewReplacer(
		"\\", "\\\\",
		"[", "\\[",
		"]", "\\]",
		"(", "\\(",
		")", "\\)",
	)
	return replacer.Replace(text)
}

func escapeForMarkdown(text, emDelimiter string, hasStrong, hasStrike bool) string {
	text = strings.ReplaceAll(text, "\\", "\\\\")
	if emDelimiter == "*" || hasStrong {
		text = strings.ReplaceAll(text, "*", "\\*")
	}
	if emDelimiter == "_" {
		text = strings.ReplaceAll(text, "_", "\\_")
	}
	if hasStrike {
		text = strings.ReplaceAll(text, "~", "\\~")
	}
	return text
}

func padWithZeroWidthSpace(text string) string {
	if text == "" {
		return text
	}
	zwsp := "\u200B"
	if !strings.HasPrefix(text, zwsp) {
		if r, ok := firstRune(text); ok && !unicode.IsSpace(r) && isYakumono(r) {
			text = zwsp + text
		}
	}
	if !strings.HasSuffix(text, zwsp) {
		if r, ok := lastRune(text); ok && !unicode.IsSpace(r) && isYakumono(r) {
			text = text + zwsp
		}
	}
	return text
}

func isYakumono(r rune) bool {
	switch r {
	case '、', '。', '，', '．', '｡', '､', '･', '・',
		'：', '；', '！', '？', '!', '?',
		'「', '」', '『', '』', '（', '）', '［', '］', '【', '】',
		'〈', '〉', '《', '》', '“', '”', '‘', '’',
		'…', '‥', '〜', '～', 'ー', '—', '―', '‐', '‑', 'ｰ':
		return true
	default:
		return false
	}
}

func firstRune(text string) (rune, bool) {
	for _, r := range text {
		return r, true
	}
	return 0, false
}

func lastRune(text string) (rune, bool) {
	var last rune
	found := false
	for _, r := range text {
		last = r
		found = true
	}
	return last, found
}

func normalizeRow(row []string, colCount int) []string {
	if len(row) == colCount {
		return row
	}
	if len(row) > colCount {
		return row[:colCount]
	}
	normalized := make([]string, colCount)
	copy(normalized, row)
	return normalized
}

func formatTableRow(row []string) string {
	for i, cell := range row {
		row[i] = strings.TrimSpace(cell)
	}
	return "| " + strings.Join(row, " | ") + " |"
}

func formatTableSeparator(colCount int) string {
	if colCount <= 0 {
		return ""
	}
	parts := make([]string, colCount)
	for i := range parts {
		parts[i] = "---"
	}
	return "| " + strings.Join(parts, " | ") + " |"
}
