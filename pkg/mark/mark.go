package mark

import (
	"regexp"
	"strings"

	"github.com/starbx/brew-api/pkg/utils"
)

type sensitiveWorker struct {
	contents []string
	index    map[int]sensitiveWords
}

var (
	HiddenRegexp = regexp.MustCompile(`\$hidden\[(.*?)\]`)
)

func (s *sensitiveWorker) Do(text string) string {
	matches := HiddenRegexp.FindAllStringSubmatch(text, -1)

	for i, match := range matches {
		s.contents = append(s.contents, match[0])
		s.index[i] = sensitiveWords{
			Old: match[0],
			New: strings.ReplaceAll(match[0], match[1], utils.RandomStr(10)),
		}

		text = strings.Replace(text, s.index[i].Old, s.index[i].New, 1)
	}
	return text
}

func (s *sensitiveWorker) Undo(text string) string {
	for _, v := range s.index {
		text = strings.ReplaceAll(text, v.New, v.Old)
	}
	return text
}

type sensitiveWords struct {
	Old string
	New string
}

func NewSensitiveWork() *sensitiveWorker {
	return &sensitiveWorker{
		index: make(map[int]sensitiveWords),
	}
}
