package app

import (
	"common/config"
	"context"
	"fmt"
	"google.golang.org/grpc"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Run 启动程序
func Run(ctx context.Context) error {
	// 启动 grpc 服务端
	server := grpc.NewServer()

	go func() {
		listen, err := net.Listen("tcp", config.Conf.Grpc.Addr)
		if err != nil {
			log.Fatalf("user grpc server listen err: %v", err)
		}

		log.Printf("user grpc server started listen on %s\n", config.Conf.Grpc.Addr)

		if err = server.Serve(listen); err != nil {
			log.Fatalf("user grpc server run failed err: %v", err)
		}
	}()

	stop := func() {
		server.Stop()
		time.Sleep(3 * time.Second)
		fmt.Println("stop app finish")
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT, syscall.SIGHUP)
	for {
		select {
		case <-ctx.Done():
			stop()
			return nil
		case s := <-c:
			switch s {
			case syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT:
				stop()
				log.Println("user app quit")
				return nil
			case syscall.SIGHUP:
				stop()
				log.Println("hang up!! user app quit")
				return nil
			default:
				return nil
			}
		}
	}
}
