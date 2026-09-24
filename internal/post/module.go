package post

import "go.uber.org/fx"

// Module bundles the post package's constructors as fx providers.
var Module = fx.Module("post",
	fx.Provide(fx.Annotate(NewPostgresRepository, fx.As(new(Repository)))),
	fx.Provide(fx.Annotate(NewRedisRepo, fx.As(new(FeedRepository)))),
	fx.Provide(NewService),
	fx.Provide(NewHandler),
)
