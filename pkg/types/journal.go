package types

type Journal struct {
	ID        string           `json:"id" db:"id"`
	SpaceID   string           `json:"space_id" db:"space_id"`
	UserID    string           `json:"user_id" db:"user_id"`
	Date      string           `json:"date" db:"date"`
	Content   KnowledgeContent `json:"content" db:"content"`
	CreatedAt int64            `json:"created_at" db:"created_at"`
	UpdatedAt int64            `json:"updated_at" db:"updated_at"`
}
