package useraccess

import (
	"context"
	"fmt"

	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
)

// GetOwnerRefs returns every non-deleted user as a minimal {id, display_name}
// reference for owner pickers, sorted the way they render.
func (r *Repository) GetOwnerRefs(ctx context.Context) ([]httpapi.OwnerRef, error) {
	type row struct {
		ID          ids.UserID `db:"id"`
		DisplayName string     `db:"display_name"`
	}
	// u.id breaks display-name ties: display names are not unique, so without it
	// two owners sharing one can swap places between refetches.
	const query = `
		SELECT id, display_name
		FROM users
		WHERE deleted_at IS NULL
		ORDER BY display_name, id
	`
	var rows []row
	if err := r.db.SelectContext(ctx, &rows, query); err != nil {
		return nil, fmt.Errorf("get owner refs: %w", err)
	}

	refs := make([]httpapi.OwnerRef, len(rows))
	for i, rw := range rows {
		refs[i] = httpapi.OwnerRef{
			Id:          rw.ID.Int64(),
			DisplayName: rw.DisplayName,
		}
	}
	return refs, nil
}
