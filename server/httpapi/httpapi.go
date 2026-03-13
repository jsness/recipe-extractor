package httpapi

import (
	"log"
	"net/http"

	"recipe-extractor/server/core"
	internalapi "recipe-extractor/server/internal/api"
)

type Config struct {
	FrontendDevProxyURL string
}

func NewHandler(cfg Config, app *core.App, logger *log.Logger) http.Handler {
	return internalapi.NewHandler(internalapi.Config{
		FrontendDevProxyURL: cfg.FrontendDevProxyURL,
	}, app, logger).Routes()
}
