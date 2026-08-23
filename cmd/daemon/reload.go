package main

import (
	"github.com/o3willard-AI/SSSonector/internal/config"
	"github.com/o3willard-AI/SSSonector/internal/tunnel"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func applyReload(logger *zap.Logger, configFile, flagLevel string, flagSet bool, prev zapcore.Level, atom *zap.AtomicLevel, reloadable tunnel.Reloadable) zapcore.Level {
	newCfg, err := config.LoadConfigFile(configFile)
	if err != nil {
		logger.Error("SIGHUP reload failed: cannot load config",
			zap.String("path", configFile), zap.Error(err))
		return prev
	}
	if err := config.NewValidator().Validate(newCfg); err != nil {
		logger.Error("SIGHUP reload rejected: config validation failed", zap.Error(err))
		return prev
	}

	if reloadable != nil {
		if err := reloadable.ApplyConfig(newCfg); err != nil {
			logger.Error("SIGHUP reload failed while applying runtime settings", zap.Error(err))
			return prev
		}
	}

	cfgLevel := ""
	if newCfg.Config != nil {
		cfgLevel = newCfg.Config.Logging.Level
	}
	newLevel, lerr := resolveLogLevel(flagLevel, flagSet, cfgLevel)
	if lerr != nil {
		logger.Warn("Ignoring invalid logging.level in reloaded config", zap.Error(lerr))
	} else if newLevel != prev {
		if atom != nil {
			atom.SetLevel(newLevel)
		}
		logger.Info("Log level updated",
			zap.String("level", newLevel.String()))
		prev = newLevel
	}

	logger.Info("Configuration reloaded", zap.String("config", configFile))
	return prev
}
