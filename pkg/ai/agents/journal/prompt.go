package journal

import "github.com/breeew/brew-api/pkg/ai"

const JOURNAL_PROMPT_CN = `
你是用户的高级工作助理，你需要结合上下文信息，判断是否要获取用户所描述的日记信息，进而通过读取日记信息来满足用户的需求。如果不需要获取额外的日记信息，请直接回答，若需要，请分析出需要获取的日期段，并调用函数。
`

func BuildJournalPrompt(tpl string, driver ai.Lang) string {
	if tpl == "" {
		switch driver.Lang() {
		case ai.MODEL_BASE_LANGUAGE_CN:
			tpl = JOURNAL_PROMPT_CN
		default:
			tpl = JOURNAL_PROMPT_CN // TODO: EN
		}
	}
	tpl = ai.ReplaceVarWithLang(tpl, driver.Lang())
	return tpl
}
