package hostaccess

import "errors"

var (
	ErrHostNotFound       = errors.New("host not found")
	ErrHostConflict       = errors.New("host already exists")
	ErrHostFQDNImmutable  = errors.New("host fqdn is immutable; rename = delete + create")
	ErrDuplicateHostID    = errors.New("duplicate host id in reconcile request")
	ErrDuplicateHostFQDN  = errors.New("duplicate fqdn in reconcile request")
	ErrHostGroupNotFound  = errors.New("host group not found")
	ErrHostGroupConflict  = errors.New("host group name already exists")
	ErrSuggestionNotFound = errors.New("ignored suggestion not found")
	ErrSuggestionConflict = errors.New("suggestion already ignored")
	ErrReferenceNotFound  = errors.New("referenced entity not found")
	ErrGroupNameRequired  = errors.New("group name is required")
	ErrDuplicateGroupID   = errors.New("duplicate group id")
)
