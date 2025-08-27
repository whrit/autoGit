package logs

import (
	"io"
	"log"
	"os"

	"github.com/natefinch/lumberjack"
	"github.com/whrit/autoGit/internal/config"
)

func Setup(cfg config.Config) error {
	if cfg.LogPath == "" {
		// leave default logger to stderr
		return nil
	}
	lj := &lumberjack.Logger{
		Filename:   cfg.LogPath,
		MaxSize:    max(1, cfg.LogMaxSize),
		MaxBackups: max(0, cfg.LogMaxBackups),
		MaxAge:     max(1, cfg.LogMaxAge),
		Compress:   true,
	}
	log.SetOutput(io.MultiWriter(os.Stdout, lj))
	return nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
