package utils

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/davidscottmills/goeditorjs"
)

var editorJSMarkdownEngine *goeditorjs.MarkdownEngine

func init() {
	editorJSMarkdownEngine = goeditorjs.NewMarkdownEngine()
	// Register the handlers you wish to use
	editorJSMarkdownEngine.RegisterBlockHandlers(
		&goeditorjs.HeaderHandler{},
		&goeditorjs.ParagraphHandler{},
		&goeditorjs.ListHandler{},
		&ListV2Handler{},
		&goeditorjs.CodeBoxHandler{},
		&goeditorjs.CodeHandler{},
		&goeditorjs.ImageHandler{},
		&goeditorjs.TableHandler{},
	)
}

func ConvertEditorJSBlocksToMarkdown(blockString json.RawMessage) (string, error) {
	return editorJSMarkdownEngine.GenerateMarkdownWithUnknownBlock(string(blockString))
}

// list represents list data from EditorJS
type listv2 struct {
	Style string       `json:"style"`
	Items []listv2Item `json:"items"`
}

type listv2Item struct {
	Content string          `json:"content"`
	Items   json.RawMessage `json:"items"`
	Meta    any             `json:"meta"`
}

// ListV2Handler is the default ListV2Handler for EditorJS HTML generation
type ListV2Handler struct{}

func (*ListV2Handler) parse(editorJSBlock goeditorjs.EditorJSBlock) (*listv2, error) {
	list := &listv2{}
	return list, json.Unmarshal(editorJSBlock.Data, list)
}

// Type "listv2"
func (*ListV2Handler) Type() string {
	return "listv2"
}

func renderListv2Html(style string, list []listv2Item) (string, error) {
	result := ""
	if style == "ordered" {
		result = "<ol>%s</ol>"
	} else {
		result = "<ul>%s</ul>"
	}
	innerData := strings.Builder{}
	for _, s := range list {
		if len(s.Items) > 0 {

			var inner []listv2Item
			if err := json.Unmarshal(s.Items, &inner); err != nil {
				return "", err
			}
			innerHtml, err := renderListv2Html(style, inner)
			if err != nil {
				return "", err
			}

			s.Content = fmt.Sprintf("<span>%s</span>%s", s.Content, innerHtml)
		}
		innerData.WriteString("<li>")
		innerData.WriteString(s.Content)
		innerData.WriteString("</li>")
	}

	if innerData.Len() > 0 {
		return fmt.Sprintf(result, innerData.String()), nil
	}
	return "", nil
}

// GenerateHTML generates html for ListBlocks
func (h *ListV2Handler) GenerateHTML(editorJSBlock goeditorjs.EditorJSBlock) (string, error) {
	list, err := h.parse(editorJSBlock)
	if err != nil {
		return "", err
	}

	return renderListv2Html(list.Style, list.Items)
}

func renderListv2Markdown(style string, index int, list []listv2Item) (string, error) {
	positionPrefix := strings.Repeat("  ", index)
	listItemPrefix := positionPrefix + "- "
	results := []string{}
	for i, s := range list {
		if style == "ordered" {
			listItemPrefix = fmt.Sprintf("%d.", i+1)
		}

		results = append(results, fmt.Sprintf("%s%s  ", listItemPrefix, s.Content))
		if len(s.Items) > 0 {
			var inner []listv2Item
			if err := json.Unmarshal(s.Items, &inner); err != nil {
				return "", err
			}
			innerMarkdown, err := renderListv2Markdown(style, index+1, inner)
			if err != nil {
				return "", err
			}
			if innerMarkdown != "" {
				results = append(results, innerMarkdown)
			}
		}
	}

	if len(results) > 0 {
		return strings.Join(results, "\n"), nil
	}
	return "", nil
}

// GenerateMarkdown generates markdown for ListBlocks
func (h *ListV2Handler) GenerateMarkdown(editorJSBlock goeditorjs.EditorJSBlock) (string, error) {
	list, err := h.parse(editorJSBlock)
	if err != nil {
		return "", err
	}

	return renderListv2Markdown(list.Style, 0, list.Items)
}
