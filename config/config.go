// Package config 负责加载服务运行所需的全部配置项。
//
// 配置来源优先级：环境变量 > 默认值。所有与业务规则相关的阈值
// （上报周期、困人判定阈值、接单超时时限、评分权重等）都集中在此，
// 避免散落在各业务模块中导致口径不一致。
package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"
)

// Config 汇总服务运行期全部配置。
type Config struct {
	// Port HTTP 监听端口，默认 8080，可用环境变量 PORT 覆盖。
	Port string
	// DataFile JSON 持久化文件路径；为空表示仅使用内存存储。
	DataFile string
	// AutoPersist 为 true 时，超时扫描器每次扫描后主动落盘。
	AutoPersist bool
	// LogLevel 结构化日志级别：debug / info / warn / error。
	LogLevel string

	// ReadTimeout HTTP 读超时。
	ReadTimeout time.Duration
	// WriteTimeout HTTP 写超时。
	WriteTimeout time.Duration
	// IdleTimeout HTTP keep-alive 空闲超时。
	IdleTimeout time.Duration
	// ShutdownTimeout 优雅关闭等待在途请求的最大时长。
	ShutdownTimeout time.Duration

	// ReportPeriod 终端状态上报周期（秒），默认 5 秒。
	ReportPeriod time.Duration
	// EntrapmentThreshold 困人判定持续时间阈值（秒），默认 30 秒。
	EntrapmentThreshold time.Duration
	// AcceptDeadline 接单后允许的最大处置时限，默认 10 分钟。
	AcceptDeadline time.Duration
	// SweepInterval 困人超时扫描任务执行周期，默认 30 秒。
	SweepInterval time.Duration
	// ScoreWindow 健康评分统计窗口，默认 30 天。
	ScoreWindow time.Duration

	// FaultScoreWeight 每发生一次故障扣分。
	FaultScoreWeight int
	// UntimelyScoreWeight 每次未按时处置扣分。
	UntimelyScoreWeight int
	// WatchlistThreshold 健康评分低于等于该值进入重点关注名单。
	WatchlistThreshold int
	// UnknownFaultPrompt 未知故障码的提示文案。
	UnknownFaultPrompt string
}

// Default 返回一份带默认值的配置。
func Default() *Config {
	return &Config{
		Port:                "8080",
		DataFile:            "data/elevator-state.json",
		AutoPersist:         true,
		LogLevel:            "info",
		ReadTimeout:         15 * time.Second,
		WriteTimeout:        30 * time.Second,
		IdleTimeout:         60 * time.Second,
		ShutdownTimeout:     8 * time.Second,
		ReportPeriod:        5 * time.Second,
		EntrapmentThreshold: 30 * time.Second,
		AcceptDeadline:      10 * time.Minute,
		SweepInterval:       30 * time.Second,
		ScoreWindow:         30 * 24 * time.Hour,
		FaultScoreWeight:    2,
		UntimelyScoreWeight: 5,
		WatchlistThreshold:  60,
		UnknownFaultPrompt:  "未知故障码，需人工确认",
	}
}

// Load 从环境变量加载配置并叠加默认值。
//
// 对非法整数类的环境变量，当前实现退回默认值；最终配置统一由
// Validate 校验，保证运行期不会拿到无效阈值。
func Load() *Config {
	cfg := Default()
	if v := os.Getenv("PORT"); v != "" {
		cfg.Port = v
	}
	if v := os.Getenv("DATA_FILE"); v != "" {
		cfg.DataFile = v
	}
	if v := os.Getenv("AUTO_PERSIST"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			cfg.AutoPersist = b
		}
	}
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		cfg.LogLevel = v
	}
	if v := os.Getenv("READ_TIMEOUT_SEC"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.ReadTimeout = time.Duration(n) * time.Second
		}
	}
	if v := os.Getenv("WRITE_TIMEOUT_SEC"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.WriteTimeout = time.Duration(n) * time.Second
		}
	}
	if v := os.Getenv("IDLE_TIMEOUT_SEC"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.IdleTimeout = time.Duration(n) * time.Second
		}
	}
	if v := os.Getenv("SHUTDOWN_TIMEOUT_SEC"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.ShutdownTimeout = time.Duration(n) * time.Second
		}
	}
	if v := os.Getenv("REPORT_PERIOD_SEC"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.ReportPeriod = time.Duration(n) * time.Second
		}
	}
	if v := os.Getenv("ENTRAPMENT_THRESHOLD_SEC"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.EntrapmentThreshold = time.Duration(n) * time.Second
		}
	}
	if v := os.Getenv("ACCEPT_DEADLINE_MIN"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.AcceptDeadline = time.Duration(n) * time.Minute
		}
	}
	if v := os.Getenv("SWEEP_INTERVAL_SEC"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.SweepInterval = time.Duration(n) * time.Second
		}
	}
	if v := os.Getenv("FAULT_SCORE_WEIGHT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			cfg.FaultScoreWeight = n
		}
	}
	if v := os.Getenv("UNTIMELY_SCORE_WEIGHT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			cfg.UntimelyScoreWeight = n
		}
	}
	if v := os.Getenv("WATCHLIST_THRESHOLD"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			cfg.WatchlistThreshold = n
		}
	}
	return cfg
}

// Validate 校验配置项合法性。任何无效配置都返回明确错误，避免服务带病启动。
func (c *Config) Validate() error {
	if port, err := strconv.Atoi(c.Port); err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("PORT 非法: %q（应为 1-65535）", c.Port)
	}
	switch c.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("LOG_LEVEL 非法: %q（应为 debug/info/warn/error）", c.LogLevel)
	}
	for _, d := range []struct {
		name  string
		value time.Duration
	}{
		{"ReadTimeout", c.ReadTimeout},
		{"WriteTimeout", c.WriteTimeout},
		{"IdleTimeout", c.IdleTimeout},
		{"ShutdownTimeout", c.ShutdownTimeout},
		{"ReportPeriod", c.ReportPeriod},
		{"EntrapmentThreshold", c.EntrapmentThreshold},
		{"AcceptDeadline", c.AcceptDeadline},
		{"SweepInterval", c.SweepInterval},
		{"ScoreWindow", c.ScoreWindow},
	} {
		if d.value <= 0 {
			return fmt.Errorf("%s 必须大于 0，当前 %s", d.name, d.value)
		}
	}
	if c.FaultScoreWeight < 0 {
		return fmt.Errorf("FAULT_SCORE_WEIGHT 不能为负，当前 %d", c.FaultScoreWeight)
	}
	if c.UntimelyScoreWeight < 0 {
		return fmt.Errorf("UNTIMELY_SCORE_WEIGHT 不能为负，当前 %d", c.UntimelyScoreWeight)
	}
	if c.WatchlistThreshold < 0 || c.WatchlistThreshold > 100 {
		return fmt.Errorf("WATCHLIST_THRESHOLD 必须在 0-100 之间，当前 %d", c.WatchlistThreshold)
	}
	if c.UnknownFaultPrompt == "" {
		return fmt.Errorf("UnknownFaultPrompt 不能为空")
	}
	return nil
}

// SlogLevel 将配置中的日志级别转换为 slog.Level。
func (c *Config) SlogLevel() slog.Level {
	switch c.LogLevel {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
