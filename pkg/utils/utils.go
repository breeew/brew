package utils

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"math/rand"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/holdno/snowFlakeByGo"

	"github.com/breeew/brew-api/pkg/errors"
	"github.com/breeew/brew-api/pkg/i18n"
)

var (
	// IdWorker 全局唯一id生成器实例
	idWorker *snowFlakeByGo.Worker
)

func SetupIDWorker(clusterID int64) {
	idWorker, _ = snowFlakeByGo.NewWorker(clusterID)
}

func GenSpecID() int64 {
	return idWorker.GetId()
}

func GenSpecIDStr() string {
	return strconv.FormatInt(GenSpecID(), 10)
}

func GenRandomID() string {
	return RandomStr(32)
}

// RandomStr 随机字符串
func RandomStr(l int) string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	seed := "1234567890qwertyuiopasdfghjklzxcvbnmQWERTYUIOPASDFGHJKLZXCVBNM"
	str := ""
	length := len(seed)
	for i := 0; i < l; i++ {
		point := r.Intn(length)
		str = str + seed[point:point+1]
	}
	return str
}

// Random 生成随机数
func Random(min, max int) int {
	if min == max {
		return max
	}
	max = max + 1
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return min + r.Intn(max-min)
}

func MD5(s string) string {
	md5Ctx := md5.New()
	md5Ctx.Write([]byte(s))
	cipherStr := md5Ctx.Sum(nil)

	return hex.EncodeToString(cipherStr)
}

func BindArgsWithGin(c *gin.Context, req interface{}) error {
	err := c.ShouldBindWith(req, binding.Default(c.Request.Method, c.ContentType()))
	if err != nil {
		return errors.New(fmt.Sprintf("Gin.ShouldBindWith.%s.%s", c.Request.Method, c.Request.URL.Path), i18n.ERROR_INVALIDARGUMENT, err).Code(http.StatusBadRequest)
	}
	return nil
}

type Binding interface {
	Name() string
	Bind(*http.Request, any) error
}

func TextEnterToBr(s string) string {
	return strings.TrimSpace(strings.Replace(strings.Replace(s, "\r\n", "(br)", -1), "\n", "(br)", -1))
}

func IsAlphabetic(s string) bool {
	match, _ := regexp.MatchString(`^[a-zA-Z]+$`, s)
	return match
}

func GenUserPassword(salt string, pwd string) string {
	return MD5(MD5(salt) + salt + MD5(pwd))
}
