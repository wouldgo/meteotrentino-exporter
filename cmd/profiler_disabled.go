//go:build !profile

package options

import "go.uber.org/zap"

func RunProfiler(addr string, logger *zap.Logger) {
}
