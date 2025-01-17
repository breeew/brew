package utils

import (
	"fmt"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenRandomID(t *testing.T) {
	SetupIDWorker(1)

	t.Log(GenSpecIDStr(), len(GenSpecIDStr()))
}

func Test_ParseAcceptLanguage(t *testing.T) {
	res := ParseAcceptLanguage("zh-CN,zh;q=0.9,en-US;q=0.8,en;q=0.7")
	t.Log(res)
}

func Test_Crypt(t *testing.T) {
	// 密钥和明文
	key := []byte("examplekey123456")
	plaintext := []byte("Sensitive Data to be encrypted")

	// 加密
	ciphertext, err := EncryptCFB(plaintext, key)
	if err != nil {
		log.Fatalf("加密失败: %v", err)
	}
	fmt.Printf("加密后的数据: %s\n", ciphertext)

	// 解密
	decrypted, err := DecryptCFB(ciphertext, key)
	if err != nil {
		log.Fatalf("解密失败: %v", err)
	}
	fmt.Printf("解密后的数据: %s\n", string(decrypted))

	assert.Equal(t, plaintext, decrypted)
}
