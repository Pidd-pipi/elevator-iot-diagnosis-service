package main

import (
	"io"
	"log/slog"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"example.com/elevator-iot-diagnosis-service/config"
)

// 关闭信号到达后，服务必须停止扫描任务并正常退出，不残留后台 goroutine。
func TestBug002ShutdownStopsSweeper(t *testing.T) {
	cfg := config.Default()
	cfg.Port = "0"
	cfg.DataFile = filepath.Join(t.TempDir(), "state.json")
	cfg.SweepInterval = 10 * time.Millisecond
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	done := make(chan error, 1)
	go func() { done <- run(cfg, logger) }()
	time.Sleep(500 * time.Millisecond)
	_ = syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("优雅关闭返回错误: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("关闭信号后服务未退出，扫描任务未随关闭停止")
	}
}
