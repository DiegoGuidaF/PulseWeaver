package hostaccess

import "errors"

var (
	ErrKnownHostNotFound      = errors.New("known host not found")
	ErrKnownHostConflict      = errors.New("known host already exists")
	ErrKnownHostFQDNImmutable = errors.New("known host fqdn is immutable; rename = delete + create")
	ErrDuplicateKnownHostID   = errors.New("duplicate known host id in reconcile request")
	ErrDuplicateKnownHostFQDN = errors.New("duplicate fqdn in reconcile request")
	ErrHostGroupNotFound      = errors.New("host group not found")
	ErrHostGroupConflict      = errors.New("host group name already exists")
	ErrSuggestionNotFound     = errors.New("ignored suggestion not found")
	ErrSuggestionConflict     = errors.New("suggestion already ignored")
	ErrReferenceNotFound      = errors.New("referenced entity not found")
	ErrGroupNameRequired      = errors.New("group name is required")
	ErrDuplicateGroupID       = errors.New("duplicate group id")
)
