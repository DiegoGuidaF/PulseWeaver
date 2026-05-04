package device

import "github.com/DiegoGuidaF/PulseWeaver/internal/database"

type Repository struct {
	db *database.DB
}

func NewRepository(db *database.DB) *Repository {
	return &Repository{
		db: db,
	}
}
