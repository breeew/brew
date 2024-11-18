package sqlstore

import (
	"context"

	sq "github.com/Masterminds/squirrel"
	"github.com/breeew/brew-api/pkg/register"
	"github.com/breeew/brew-api/pkg/types"
)

func init() {
	register.RegisterFunc(registerKey{}, func() {
		provider.stores.KnowledgeChunkStore = NewKnowledgeChunkStore(provider)
	})
}

type FileManagementStore struct {
	CommonFields
}

func NewFileManagementStore(provider SqlProviderAchieve) *FileManagementStore {
	repo := &FileManagementStore{}
	repo.SetProvider(provider)
	repo.SetTable(types.TABLE_FILE_MANAGEMENT)
	repo.SetAllColumns("id", "space_id", "user_id", "file", "file_size", "object_type", "kind", "status", "created_at")
	return repo
}

// Create 创建新的文件记录
func (s *FileManagementStore) Create(ctx context.Context, data types.FileManagement) error {
	query := sq.Insert(s.GetTable()).
		Columns("space_id", "user_id", "file", "file_size", "object_type", "kind", "status", "created_at").
		Values(data.SpaceID, data.UserID, data.File, data.FileSize, data.ObjectType, data.Kind, data.Status, data.CreatedAt)

	queryString, args, err := query.ToSql()
	if err != nil {
		return errorSqlBuild(err)
	}

	_, err = s.GetMaster(ctx).Exec(queryString, args...)
	if err != nil {
		return err
	}
	return nil
}

// GetByID 根据ID获取文件记录
func (s *FileManagementStore) GetByID(ctx context.Context, userID, file string) (*types.FileManagement, error) {
	query := sq.Select(s.GetAllColumns()...).From(s.GetTable()).Where(sq.Eq{"user_id": userID, "file": file})

	queryString, args, err := query.ToSql()
	if err != nil {
		return nil, errorSqlBuild(err)
	}

	var res types.FileManagement
	if err = s.GetReplica(ctx).Get(&res, queryString, args...); err != nil {
		return nil, err
	}
	return &res, nil
}

// Delete 根据ID删除文件记录
func (s *FileManagementStore) Delete(ctx context.Context, userID, file string) error {
	query := sq.Delete(s.GetTable()).Where(sq.Eq{"user_id": userID, "file": file})

	queryString, args, err := query.ToSql()
	if err != nil {
		return errorSqlBuild(err)
	}

	_, err = s.GetMaster(ctx).Exec(queryString, args...)
	return err
}

// ListByObjectID
func (s *FileManagementStore) ListByObjectID(ctx context.Context, userID, objectID, objectType string) ([]types.FileManagement, error) {
	query := sq.Select(s.GetAllColumns()...).From(s.GetTable()).
		Where(sq.Eq{"user_id": userID, "object_id": objectID, "object_type": objectType})

	queryString, args, err := query.ToSql()
	if err != nil {
		return nil, errorSqlBuild(err)
	}

	var res []types.FileManagement
	if err = s.GetReplica(ctx).Select(&res, queryString, args...); err != nil {
		return nil, err
	}
	return res, nil
}