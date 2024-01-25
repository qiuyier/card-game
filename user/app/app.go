package app

import (
	"common/config"
	"common/logs"
	"context"
	"google.golang.org/grpc"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Run 启动程序
func Run(ctx context.Context) error {
	// 初始化日志库
	logs.InitLogger(config.Conf.AppName)

	// 启动 grpc 服务端
	server := grpc.NewServer()

	go func() {
		listen, err := net.Listen("tcp", config.Conf.Grpc.Addr)
		if err != nil {
			logs.Fatal("user grpc server listen err: %v", err)
		}

		logs.Info("user grpc server started listen on %s\n", config.Conf.Grpc.Addr)

		if err = server.Serve(listen); err != nil {
			logs.Fatal("user grpc server run failed err: %v", err)
		}
	}()

	stop := func() {
		server.Stop()
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
				logs.Info("user app quit")
				return nil
			case syscall.SIGHUP:
				stop()
				logs.Info("hang up!! user app quit")
				return nil
			default:
				return nil
			}
		}
	}
}
