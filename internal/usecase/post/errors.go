package post

import "errors"

var ErrForbidden = errors.New("forbidden: not the owner of this post")
