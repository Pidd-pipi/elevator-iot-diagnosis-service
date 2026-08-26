// 电梯物联网状态诊断与困人告警服务（elevator-iot-diagnosis-service）。
//
// 基于 Go 实现的电梯物联网 Web 项目，一款后端服务，完成电梯运行状态采集、
// 故障码诊断、困人事件告警、处置闭环与运行健康评分。
// 前端页面通过 go:embed 内嵌 web/ 静态资源，离线可运行。
package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"example.com/elevator-iot-diagnosis-service/config"
	"example.com/elevator-iot-diagnosis-service/httpapi"
	"example.com/elevator-iot-diagnosis-service/service"
	"example.com/elevator-iot-diagnosis-service/store"
)

//go:embed all:web
var embeddedWeb embed.FS

func main() {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		fmt.Fprintln(os.Stderr, "配置校验失败:", err)
		os.Exit(1)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.SlogLevel()}))
	if err := run(cfg, logger); err != nil {
		logger.Error("service exited with error", "err", err)
		os.Exit(1)
	}
}

// run 完成装配、启动与优雅关闭。
func run(cfg *config.Config, logger *slog.Logger) error {
	// 1. 仓储：内存 + JSON 文件持久化。
	st := store.NewStore()
	if cfg.DataFile != "" {
		if err := st.Load(cfg.DataFile); err != nil && !errors.Is(err, fs.ErrNotExist) {
			logger.Warn("load snapshot failed, start with empty store", "file", cfg.DataFile, "err", err)
		} else if err == nil {
			logger.Info("snapshot loaded", "file", cfg.DataFile)
		}
	}

	// 2. 业务服务装配 + 首次启动种子数据。
	svc := service.NewServices(st, cfg, logger)
	if err := svc.Seed.EnsureSeed(); err != nil {
		return err
	}
	scoreCount := svc.Scoring.RefreshAll(time.Now())
	logger.Info("seed ready", "elevators", st.Elevators.Count(), "scored", scoreCount)

	// 3. 定时任务：困人超时扫描 + 周期性落盘。
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	svc.Sweeper.SetOnSweep(func() {
		if cfg.AutoPersist && cfg.DataFile != "" {
			if err := st.Save(cfg.DataFile); err != nil {
				logger.Error("periodic save failed", "err", err)
			}
		}
	})
	sweeperDone := make(chan struct{})
	go func() {
		defer close(sweeperDone)
		svc.Sweeper.Run(ctx)
	}()

	// 4. HTTP 服务：补齐全部超时，避免慢请求/空闲连接占用资源。
	handler := httpapi.Router(svc, cfg, logger, mustSubFS(embeddedWeb))
	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
	}
	serverErr := make(chan error, 1)
	go func() {
		logger.Info("elevator-iot-diagnosis-service listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	// 5. 等待退出信号或启动失败。
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	select {
	case err := <-serverErr:
		return err
	case sig := <-quit:
		logger.Info("shutdown signal received", "signal", sig.String())
	}

	// 6. 优雅关闭：停止接收新请求、等待在途请求、停止扫描任务、最终落盘。
	cancel() // 取消扫描任务，使 Sweeper.Run 退出
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Warn("graceful shutdown incomplete", "err", err)
	}
	<-sweeperDone
	if cfg.DataFile != "" {
		if err := st.Save(cfg.DataFile); err != nil {
			logger.Error("final save failed", "err", err)
		} else {
			logger.Info("snapshot saved", "file", cfg.DataFile)
		}
	}
	logger.Info("service stopped cleanly")
	return nil
}

// mustSubFS 将 embed.FS 约束为根目录视图（web/ 前缀被剥除）。
func mustSubFS(fsys embed.FS) fs.FS {
	sub, err := fs.Sub(fsys, "web")
	if err != nil {
		panic("web 资源嵌入失败: " + err.Error())
	}
	return sub
}
