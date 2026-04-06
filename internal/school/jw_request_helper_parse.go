package school

import (
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

var (
	courseNamePattern = regexp.MustCompile(`课程名称[:：]\s*([^\n]+)`)
	courseRoomPattern = regexp.MustCompile(`上课地点[:：]\s*([^\n]+)`)
	courseTeacherRe   = regexp.MustCompile(`(?:教师|授课教师)[:：]\s*([^\n]+)`)
	courseTimePattern = regexp.MustCompile(`星期([一二三四五六日天])\s*\[?(\d{2}-\d{2})\]?`)
	welcomeNameRe     = regexp.MustCompile(`欢迎您[:：]?\s*([^\s<]+)`)
)

func looksLikeJWLoginPage(raw string) bool {
	lower := strings.ToLower(raw)
	return strings.Contains(lower, `id="loginform"`) ||
		strings.Contains(lower, `name="useraccount"`) ||
		strings.Contains(lower, `name="userpassword"`) ||
		strings.Contains(lower, `id="j_username"`) ||
		strings.Contains(lower, `统一认证中心`) ||
		strings.Contains(lower, `请先登录系统`)
}

func hasUsefulHTML(raw string) bool {
	text := strings.TrimSpace(stripSpace(raw))
	return text != "" && !looksLikeJWLoginPage(raw)
}

func parseProfileMeta(raw string) (string, string, bool) {
	fields := extractLabeledFields(raw)
	name := firstNonEmpty(
		fields["姓名"],
		fields["学生姓名"],
		fields["名字"],
	)
	major := firstNonEmpty(
		fields["专业"],
		fields["专业名称"],
		fields["所在专业"],
		fields["班级"],
		fields["学院专业"],
	)
	if name == "" {
		if match := welcomeNameRe.FindStringSubmatch(raw); len(match) > 1 {
			name = cleanCellText(match[1])
		}
	}
	if name == "" && major == "" {
		return "", "", false
	}
	return name, major, true
}

func parseHTMLTableRows(raw string) []map[string]string {
	tables := extractHTMLTables(raw)
	best := tableData{}
	for _, table := range tables {
		if len(table.Rows) > len(best.Rows) {
			best = table
		}
	}
	if len(best.Rows) == 0 {
		return []map[string]string{}
	}

	headers := best.Headers
	if len(headers) == 0 {
		headers = make([]string, len(best.Rows[0]))
		for idx := range headers {
			headers[idx] = "col" + strconv.Itoa(idx+1)
		}
	}

	out := make([]map[string]string, 0, len(best.Rows))
	for _, row := range best.Rows {
		item := make(map[string]string, len(headers))
		for idx, header := range headers {
			if idx >= len(row) {
				item[header] = ""
				continue
			}
			item[header] = row[idx]
		}
		if len(item) > 0 {
			out = append(out, item)
		}
	}
	return out
}

func parseWeeklyCourseHTML(raw string, term string, targetDate string, week int) []map[string]any {
	doc, err := html.Parse(strings.NewReader(raw))
	if err != nil {
		return []map[string]any{}
	}

	var out []map[string]any
	for _, node := range findNodes(doc, "p") {
		text := cleanCellText(nodeText(node))
		if text == "" || !strings.Contains(text, "课程名称") {
			continue
		}

		item := map[string]any{
			"term":   term,
			"week":   week,
			"date":   targetDate,
			"text":   text,
			"course": firstCapture(courseNamePattern, text),
		}
		if weekday := firstCapture(courseTimePattern, text); weekday != "" {
			item["weekday"] = "星期" + weekday
		}
		if match := courseTimePattern.FindStringSubmatch(text); len(match) > 2 {
			item["section"] = match[2]
		}
		if room := firstCapture(courseRoomPattern, text); room != "" {
			item["location"] = room
		}
		if teacher := firstCapture(courseTeacherRe, text); teacher != "" {
			item["teacher"] = teacher
		}
		out = append(out, item)
	}
	return out
}

type tableData struct {
	Headers []string
	Rows    [][]string
}

func extractLabeledFields(raw string) map[string]string {
	tables := extractHTMLTables(raw)
	fields := make(map[string]string)
	for _, table := range tables {
		for _, row := range table.Rows {
			if len(row) < 2 {
				continue
			}
			for idx := 0; idx+1 < len(row); idx += 2 {
				label := normalizeLabel(row[idx])
				value := cleanCellText(row[idx+1])
				if label == "" || value == "" {
					continue
				}
				if _, exists := fields[label]; !exists {
					fields[label] = value
				}
			}
		}
	}
	return fields
}

func extractHTMLTables(raw string) []tableData {
	doc, err := html.Parse(strings.NewReader(raw))
	if err != nil {
		return nil
	}

	var tables []tableData
	for _, tableNode := range findNodes(doc, "table") {
		rows := extractRows(tableNode)
		if len(rows) == 0 {
			continue
		}

		table := tableData{}
		dataRows := rows
		if hasHeaderRow(tableNode, rows[0]) {
			table.Headers = rows[0]
			dataRows = rows[1:]
		}
		for _, row := range dataRows {
			if len(row) == 0 {
				continue
			}
			table.Rows = append(table.Rows, row)
		}
		if len(table.Headers) > 0 || len(table.Rows) > 0 {
			tables = append(tables, table)
		}
	}
	return tables
}

func extractRows(tableNode *html.Node) [][]string {
	var rows [][]string
	for _, rowNode := range findNodes(tableNode, "tr") {
		var row []string
		for _, cell := range rowCells(rowNode) {
			row = append(row, cleanCellText(nodeText(cell)))
		}
		if len(row) > 0 {
			rows = append(rows, row)
		}
	}
	return rows
}

func hasHeaderRow(tableNode *html.Node, firstRow []string) bool {
	for _, rowNode := range findNodes(tableNode, "tr") {
		for _, cell := range rowCells(rowNode) {
			if cell.Data == "th" {
				return true
			}
		}
		break
	}
	return false
}

func rowCells(rowNode *html.Node) []*html.Node {
	var cells []*html.Node
	for child := rowNode.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.ElementNode && (child.Data == "th" || child.Data == "td") {
			cells = append(cells, child)
		}
	}
	return cells
}

func findNodes(root *html.Node, tag string) []*html.Node {
	var out []*html.Node
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == tag {
			out = append(out, node)
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(root)
	return out
}

func nodeText(node *html.Node) string {
	if node == nil {
		return ""
	}
	if node.Type == html.TextNode {
		return node.Data
	}

	var parts []string
	if node.Type == html.ElementNode && (node.Data == "br" || node.Data == "p" || node.Data == "div" || node.Data == "li" || node.Data == "tr") {
		parts = append(parts, "\n")
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		parts = append(parts, nodeText(child))
	}
	return strings.Join(parts, "")
}

func cleanCellText(text string) string {
	text = strings.ReplaceAll(text, "\u00a0", " ")
	lines := strings.Split(text, "\n")
	clean := make([]string, 0, len(lines))
	for _, line := range lines {
		line = stripSpace(line)
		if line != "" {
			clean = append(clean, line)
		}
	}
	return strings.Join(clean, "\n")
}

func normalizeLabel(label string) string {
	label = cleanCellText(label)
	label = strings.TrimSuffix(label, "：")
	label = strings.TrimSuffix(label, ":")
	return label
}

func stripSpace(text string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(text)), " ")
}

func firstCapture(pattern *regexp.Regexp, text string) string {
	match := pattern.FindStringSubmatch(text)
	if len(match) > 1 {
		return cleanCellText(match[1])
	}
	return ""
}
