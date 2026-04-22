package hostaccess

import "errors"

var (
	ErrKnownHostNotFound  = errors.New("known host not found")
	ErrKnownHostConflict  = errors.New("known host already exists")
	ErrHostGroupNotFound  = errors.New("host group not found")
	ErrHostGroupConflict  = errors.New("host group name already exists")
	ErrSuggestionNotFound = errors.New("ignored suggestion not found")
	ErrSuggestionConflict = errors.New("suggestion already ignored")
	ErrReferenceNotFound  = errors.New("referenced entity not found")
)
