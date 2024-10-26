package ai

import "testing"

func Test_TimeTpl(t *testing.T) {
	tpl := GenerateTimeListAtNow()

	t.Log(tpl)
}
