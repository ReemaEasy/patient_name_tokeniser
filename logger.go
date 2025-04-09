package main

import (
	"context"
	"go.uber.org/zap"
)

func GetLogger(ctx context.Context) *zap.Logger {
	logger, _ := zap.NewDevelopment()
	return logger
}
