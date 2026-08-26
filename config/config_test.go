package config

import "testing"

func TestDefaultConfigValid(t *testing.T) {
	if err := Default().Validate(); err != nil {
		t.Fatalf("默认配置应通过校验，得到 %v", err)
	}
}

func TestValidateRejectsBadPort(t *testing.T) {
	c := Default()
	c.Port = "not-a-port"
	if err := c.Validate(); err == nil {
		t.Fatal("非法 PORT 应校验失败")
	}
}

func TestValidateRejectsBadLogLevel(t *testing.T) {
	c := Default()
	c.LogLevel = "verbose"
	if err := c.Validate(); err == nil {
		t.Fatal("非法 LOG_LEVEL 应校验失败")
	}
}

func TestSlogLevelMapping(t *testing.T) {
	c := Default()
	c.LogLevel = "debug"
	if got := c.SlogLevel().String(); got != "DEBUG" {
		t.Fatalf("debug 应映射为 DEBUG，得到 %s", got)
	}
	c.LogLevel = "error"
	if got := c.SlogLevel().String(); got != "ERROR" {
		t.Fatalf("error 应映射为 ERROR，得到 %s", got)
	}
}
