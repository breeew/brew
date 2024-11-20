package v1

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"time"

	"github.com/breeew/brew-api/internal/core"
	"github.com/breeew/brew-api/pkg/errors"
	"github.com/breeew/brew-api/pkg/i18n"
	"github.com/breeew/brew-api/pkg/types"
	"github.com/breeew/brew-api/pkg/utils"
)

type UploadLogic struct {
	ctx  context.Context
	core *core.Core
	UserInfo
}

func NewUploadLogic(ctx context.Context, core *core.Core) *UploadLogic {
	l := &UploadLogic{
		ctx:      ctx,
		core:     core,
		UserInfo: setupUserInfo(ctx, core),
	}

	return l
}

type UploadKey struct {
	Key          string `json:"key"`
	FullPath     string `json:"full_path"`
	StaticDomain string `json:"static_domain"`
	Status       string `json:"status"`
}

const (
	UPLOAD_STATUS_EXIST = "exist"
)

func hashFileName(fileName string) string {
	result := strings.Split(fileName, ".")
	var suffix string
	if len(result) > 1 {
		suffix = "." + result[len(result)-1]
		fileName = strings.TrimSuffix(fileName, suffix)
	}

	return utils.MD5(fileName) + suffix
}

func (l *UploadLogic) GenClientUploadKey(objectType, kind, fileName string) (UploadKey, error) {
	userID := l.UserInfo.GetUserInfo().User
	spaceID, _ := InjectSpaceID(l.ctx)
	filePath := genUserFilePath(spaceID, objectType)
	fileName = hashFileName(fileName)

	fullPath := filepath.Join(filePath, fileName)
	exist, err := l.core.Store().FileManagementStore().GetByID(l.ctx, spaceID, fullPath)
	if err != nil && err != sql.ErrNoRows {
		return UploadKey{}, errors.New("UploadLogic.FileManagementStore.GetById", i18n.ERROR_INTERNAL, err)
	}

	if exist != nil {
		return UploadKey{
			Status:       UPLOAD_STATUS_EXIST,
			StaticDomain: l.core.Plugins.FileUploader().GetStaticDomain(),
			FullPath:     fullPath,
		}, nil
	}

	var meta core.UploadFileMeta
	err = l.core.Store().Transaction(l.ctx, func(ctx context.Context) error {
		err := l.core.Store().FileManagementStore().Create(l.ctx, types.FileManagement{
			SpaceID:    spaceID,
			UserID:     userID,
			File:       filepath.Join(filePath, fileName),
			Status:     types.FILE_UPLOAD_STATUS_UNKNOWN,
			Kind:       kind,
			ObjectType: objectType,
			CreatedAt:  time.Now().Unix(),
		})
		if err != nil {
			return errors.New("UploadLogic.GenClientUploadKey.FileManagementStore.Create", i18n.ERROR_INTERNAL, err)
		}

		meta, err = l.core.Plugins.FileUploader().GenUploadFileMeta(filePath, fileName)
		if err != nil {
			return errors.New("UploadLogic.GenClientUploadKey.FileUploader.GenUploadFileMeta", i18n.ERROR_INTERNAL, err)
		}
		return nil
	})
	if err != nil {
		return UploadKey{}, err
	}

	return UploadKey{
		Key:          meta.UploadEndpoint,
		FullPath:     meta.FullPath,
		StaticDomain: l.core.FileUploader().GetStaticDomain(),
	}, nil
}

func genUserFilePath(userID, _type string) string {
	return filepath.Join("/brew/", userID, _type, time.Now().Format("20060102"))
}
