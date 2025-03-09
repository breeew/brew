package types

type SpaceApplication struct {
	ID        string `json:"id" db:"id"`
	SpaceID   string `json:"space_id" db:"space_id"`
	UserID    string `json:"user_id" db:"user_id"`
	Desc      string `json:"desc" db:"desc"`
	Status    string `json:"status" db:"status"`
	UpdatedAt int64  `json:"updated_at" db:"updated_at"`
	CreatedAt int64  `json:"created_at" db:"created_at"`
}

const (
	SPACE_APPLICATION_ACCESS  = "access"
	SPACE_APPLICATION_WAITING = "waiting"
	SPACE_APPLICATION_REFUSE  = "refuse"
)
