package utils

import (
	"encoding/json"

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
		&goeditorjs.CodeBoxHandler{},
		&goeditorjs.ImageHandler{},
		&goeditorjs.TableHandler{},
	)
}

func ConvertEditorJSBlocksToMarkdown(blockString json.RawMessage) (string, error) {
	return editorJSMarkdownEngine.GenerateMarkdown(string(blockString))
}
