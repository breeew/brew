package v1

import (
	"context"
	"net/http"

	"github.com/sashabaranov/go-openai"

	"github.com/breeew/brew-api/app/core"
	"github.com/breeew/brew-api/app/core/srv"
	"github.com/breeew/brew-api/app/logic/v1/process"
	"github.com/breeew/brew-api/pkg/ai"
	"github.com/breeew/brew-api/pkg/errors"
	"github.com/breeew/brew-api/pkg/i18n"
	"github.com/breeew/brew-api/pkg/types"
)

type ReaderLogic struct {
	ctx  context.Context
	core *core.Core
	UserInfo
}

func NewReaderLogic(ctx context.Context, core *core.Core) *ReaderLogic {
	l := &ReaderLogic{
		ctx:      ctx,
		core:     core,
		UserInfo: SetupUserInfo(ctx, core),
	}

	return l
}

func (l *ReaderLogic) Reader(endpoint string) (*ai.ReaderResult, error) {
	res, err := l.core.Srv().AI().Reader(l.ctx, endpoint)
	if err != nil {
		errMsg := i18n.ERROR_INTERNAL
		code := http.StatusInternalServerError

		if err == srv.ERROR_UNSUPPORTED_FEATURE {
			errMsg = i18n.ERROR_UNSUPPORTED_FEATURE
			code = http.StatusForbidden
		}
		return nil, errors.New("ReaderLogic.Reader.Srv.AI.Reader", errMsg, err).Code(code)
	}

	process.NewRecordUsageRequest("", types.USAGE_TYPE_USER, types.USAGE_SUB_TYPE_READ, "", l.GetUserInfo().User, &openai.Usage{
		CompletionTokens: res.Usage.Tokens,
	})

	return res, nil
}

func (l *ReaderLogic) DescribeImage(imageURL string) (string, error) {
	url, err := l.core.FileStorage().GenGetObjectPreSignURL(imageURL)
	if err != nil {
		return "", errors.New("KnowledgeLogic.DescribeImage.GenGetObjectPreSignURL", i18n.ERROR_INTERNAL, err)
	}

	resp, err := l.core.Srv().AI().DescribeImage(l.ctx, GetContentByClientLanguage(l.ctx, "English", "中文"), url)
	if err != nil {
		return "", errors.New("KnowledgeLogic.DescribeImage.Query", i18n.ERROR_INTERNAL, err)
	}
	if resp.Usage != nil {
		process.NewRecordUsageRequest(resp.Model, types.USAGE_TYPE_SYSTEM, types.USAGE_SUB_TYPE_DESCRIBE_IMAGE, "", l.GetUserInfo().User, resp.Usage)
	}

	return resp.Message(), nil
}

// func describeImage(ctx context.Context, driver srv.VisionAI, imageURL string) (ai.GenerateResponse, error) {
// 	opts := driver.NewVisionQuery(ctx, []*types.MessageContext{
// 		{
// 			Role: types.USER_ROLE_USER,
// 			MultiContent: []openai.ChatMessagePart{
// 				{
// 					Type: openai.ChatMessagePartTypeImageURL,
// 					ImageURL: &openai.ChatMessageImageURL{
// 						URL: imageURL,
// 					},
// 				},
// 			},
// 		},
// 	})

// 	opts.WithPrompt(lo.If(driver.Lang() == ai.MODEL_BASE_LANGUAGE_CN, ai.IMAGE_GENERATE_PROMPT_CN).Else(ai.IMAGE_GENERATE_PROMPT_EN))
// 	opts.WithVar("{lang}", GetContentByClientLanguage(ctx, "English", "中文"))
// 	resp, err := opts.Query()
// 	if err != nil {
// 		return resp, err
// 	}

// 	return resp, nil
// }
