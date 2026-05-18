package hosts

import "errors"

var (
	ErrBadRequest        = errors.New("bad request")
	ErrHostNotFound      = errors.New("host not found")
	ErrHostConflict      = errors.New("host already exists")
	ErrHostFQDNImmutable = errors.New("host fqdn is immutable; rename = delete + create")
	ErrDuplicateHostID   = errors.New("duplicate host id in reconcile request")
	ErrDuplicateHostFQDN = errors.New("duplicate fqdn in reconcile request")

	ErrHostGroupNotFound = errors.New("host group not found")
	ErrHostGroupConflict = errors.New("host group name already exists")
	ErrGroupNameRequired = errors.New("group name is required")
	ErrInvalidGroupColor = errors.New("group color must be a 6-digit hex color (e.g. #4C6EF5)")
	ErrDuplicateGroupID  = errors.New("duplicate group id")

	ErrSuggestionNotFound = errors.New("ignored suggestion not found")
	ErrSuggestionConflict = errors.New("suggestion already ignored")

	ErrReferenceNotFound = errors.New("referenced entity not found")
)
