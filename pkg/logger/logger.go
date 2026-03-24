package logger

import (
	"log/slog"
	"os"

	"github.com/grafana/loki-client-go/loki"
	slogloki "github.com/samber/slog-loki/v3"
	slogmulti "github.com/samber/slog-multi"
)

func InitLogger(lokiURL string) {
	stdoutHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	var handler slog.Handler = stdoutHandler

	if lokiURL != "" {
		lokiConfig, err := loki.NewDefaultConfig(lokiURL + "/loki/api/v1/push")
		if err == nil {
			client, err := loki.New(lokiConfig)
			if err == nil {
				lokiOption := slogloki.Option{
					Level:  slog.LevelInfo,
					Client: client,
				}
				lokiHandler := lokiOption.NewLokiHandler()
				handler = slogmulti.Fanout(stdoutHandler, lokiHandler)
			}
		}
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)
}
