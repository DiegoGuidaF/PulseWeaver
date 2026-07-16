package policy

import (
	"github.com/DiegoGuidaF/PulseWeaver/internal/database"
)

// Repository backs the policy domain's DB-reading views (e.g. the policy-map
// audit). The forward-auth hot path itself never touches the database
// directly — it reads the in-memory cache maintained by Service.
type Repository struct {
	db *database.DB
}

func NewRepository(db *database.DB) *Repository {
	return &Repository{db: db}
}
