package auth

import "github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"

// RoleToAPI converts a domain Role to its API representation. The exhaustive
// linter fails this switch when a new Role value is added without a
// corresponding case here.
func RoleToAPI(r Role) httpapi.UserRole {
	switch r {
	case SuperAdminRole:
		return httpapi.UserRoleSuperadmin
	case AdminRole:
		return httpapi.UserRoleAdmin
	case UserRole:
		return httpapi.UserRoleUser
	default:
		return httpapi.UserRole(r)
	}
}
