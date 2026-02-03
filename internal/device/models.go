package device

import (
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/database"
)

type Device struct {
	ID        string        `db:"id" json:"id"`
	Name      string        `db:"name" json:"name"`
	CreatedAt database.Time `db:"created_at" json:"created_at"`
}
