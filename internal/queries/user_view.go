package queries

import (
	"context"
	"fmt"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/auth"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
)

type UserView struct {
	ID                 ids.UserID  `db:"id"`
	Username           string      `db:"username"`
	DisplayName        string      `db:"display_name"`
	Email              string      `db:"email"`
	Role               auth.Role   `db:"role"`
	MustChangePassword bool        `db:"must_change_password"`
	BypassHostCheck    bool        `db:"bypass_host_check"`
	CreatedBy          *ids.UserID `db:"created_by"`
	CreatedAt          time.Time   `db:"created_at"`
}

func (r *Repository) GetAllUsers(ctx context.Context) ([]UserView, error) {
	const query = `
		SELECT
			u.id, u.username, u.display_name, u.email, u.role,
			u.must_change_password, u.created_by, u.created_at,
			COALESCE(uhs.bypass_host_check, 0) AS bypass_host_check
		FROM users u
		LEFT JOIN user_host_settings uhs ON uhs.user_id = u.id
		WHERE u.deleted_at IS NULL
		ORDER BY u.created_at DESC
	`
	var users []UserView
	if err := r.db.SelectContext(ctx, &users, query); err != nil {
		return nil, fmt.Errorf("get all users: %w", err)
	}
	if users == nil {
		users = []UserView{}
	}
	return users, nil
}
