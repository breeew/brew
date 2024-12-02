package utils

import (
	"testing"
)

func TestGenRandomID(t *testing.T) {
	SetupIDWorker(1)

	t.Log(GenSpecIDStr(), len(GenSpecIDStr()))
}

func Test_ParseAcceptLanguage(t *testing.T) {
	res := ParseAcceptLanguage("zh-CN,zh;q=0.9,en-US;q=0.8,en;q=0.7")
	t.Log(res)
}
