package http

import "go.uber.org/fx"

// Module bundles the http package's constructors as fx providers.
var Module = fx.Module("httpTransport",

	fx.Provide(NewHTTPServer),
	fx.Provide(NewRouter),
)
