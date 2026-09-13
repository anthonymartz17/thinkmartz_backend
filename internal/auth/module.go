package auth

import "go.uber.org/fx"

// Module bundles the auth package's constructors as fx providers.
var Module = fx.Module("auth",

	fx.Provide(fx.Annotate(NewRepository, fx.As(new(UserRepository)))),
	fx.Provide(fx.Annotate(NewTokenService, fx.As(new(TokenIssuer)))),
	fx.Provide(fx.Annotate(NewService, fx.As(new(Authenticator)))),
	fx.Provide(NewHandler),
)
