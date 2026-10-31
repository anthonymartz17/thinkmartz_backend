package feed

import (
	"github.com/anthonymartz17/thinkmartz_backend/internal/post"
	"go.uber.org/fx"
)

// Module bundles the feed package's constructors as fx providers.
var Module = fx.Module("feed",
	fx.Provide(fx.Annotate(
		NewRedisRepo,
		fx.As(new(Cache)),
		fx.As(new(post.FeedWriter)),
	)),
	fx.Provide(
		fx.Annotate(NewPostgresRepository,
			fx.As(new(Repository)),
		),
	),
	fx.Provide(fx.Annotate(
		NewService,
		fx.As(new(Reader)),
	)),
	fx.Provide(NewHandler),
)
