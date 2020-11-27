package monitoring

import (
	"context"
	"os"

	"go.uber.org/fx"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/DoomSentinel/scheduler/config"
)

func NewLogger(lc fx.Lifecycle, debug config.DebugConfig) *zap.Logger {
	logLevel := zapcore.DebugLevel
	if !debug.Debug {
		logLevel = zapcore.InfoLevel
	}
	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderCfg.TimeKey = "time"
	encoderCfg.MessageKey = "message"

	core := zapcore.NewTee(
		zapcore.NewCore(zapcore.NewJSONEncoder(encoderCfg), zapcore.Lock(os.Stdout),
			zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
				return lvl >= logLevel && lvl < zapcore.ErrorLevel
			}),
		),
		zapcore.NewCore(zapcore.NewJSONEncoder(encoderCfg), zapcore.Lock(os.Stderr), zapcore.ErrorLevel),
	)

	logger := zap.New(core, zap.AddCaller())

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error { _ = logger.Sync(); return nil },
	})

	return logger
}

func LoggerValid(logger *zap.Logger) bool {
	return logger != nil && logger.Core() != nil
}
