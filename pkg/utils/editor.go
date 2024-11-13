package utils

import (
	"github.com/davidscottmills/goeditorjs"
	"github.com/starbx/brew-api/pkg/types"
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
	)
}

func ConvertEditorJSBlocksToMarkdown(blockString types.KnowledgeContent) (string, error) {
	return editorJSMarkdownEngine.GenerateMarkdown(string(blockString))
}
