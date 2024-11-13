package utils

import (
	"testing"
)

func TestGenRandomID(t *testing.T) {
	SetupIDWorker(1)

	t.Log(GenSpecIDStr(), len(GenSpecIDStr()))
}
