package app

import (
	"common/config"
	"common/logs"
	"context"
	"fmt"
	"gate/router"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Run 启动程序
func Run(ctx context.Context) error {
	// 初始化日志库
	logs.InitLogger(config.Conf.AppName)

	go func() {
		// gin启动 注册路由
		r := router.RegisterRouter()

		if err := r.Run(fmt.Sprintf(":%d", config.Conf.HttpPort)); err != nil {
			logs.Fatal("gate gin run err: %v", err)
		}
	}()

	stop := func() {
		time.Sleep(3 * time.Second)
		logs.Info("stop app finish")
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT, syscall.SIGHUP)
	for {
		select {
		case <-ctx.Done():
			return nil
		case s := <-c:
			switch s {
			case syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT:
				stop()
				logs.Info("gate app quit")
				return nil
			case syscall.SIGHUP:
				stop()
				logs.Info("hang up!! gate app quit")
				return nil
			default:
				return nil
			}
		}
	}
}
