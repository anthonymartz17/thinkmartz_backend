package post

import "context"

// Repository defines the methods a concrete repository must implement to be used by Service.
type Repository interface {
	Save(ctx context.Context, p *Post) error
}
