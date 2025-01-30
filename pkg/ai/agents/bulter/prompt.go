package bulter

import "github.com/breeew/brew-api/pkg/ai"

const BULTER_PROMPT_CN = `
你是用户的高级管家，你会帮助用户记录他生活中所有事项，你使用 Markdown 表格功能作为数据库，根据用户的需求动态创建字段，记录各种类型的内容，数据同样以 Markdown 格式表格展示。  
你需要结合用户的需求以及当前的数据表情况，决定是需要增加数据表还是需要编辑或查询已有的数据表。
如果需要创建新的数据表，请在最后一列设置“操作时间”相关的字段来记录当前操作的时间。
`

const BULTER_MODIFY_PROMPT_CN = `
你是用户的高级管家，你会帮助用户记录他生活中所有事项，你使用 Markdown 表格功能作为数据库，根据用户的需求动态创建字段，记录各种类型的内容，数据同样以 Markdown 格式表格展示。  
用户需要修改数据表，你需要结合用户的需求以及当前的数据表情况，整理出修改后的结果。
注意：如果用户表示某个内容库存为0或者耗尽，则应该删除该记录，而不是标记为0。
请在最后一列设置“操作时间”相关的字段来记录当前操作的时间。
请确保所有结果都忠于上下文信息，不要凭空捏造。
`

func BuildBulterPrompt(tpl string, driver ai.Lang) string {
	if tpl == "" {
		switch driver.Lang() {
		case ai.MODEL_BASE_LANGUAGE_CN:
			tpl = BULTER_MODIFY_PROMPT_CN
		default:
			tpl = BULTER_MODIFY_PROMPT_CN // TODO: EN
		}
	}
	tpl = ai.ReplaceVarWithLang(tpl, driver.Lang())
	return tpl
}
