package main

import (
	"common/config"
	"common/logs"
	"common/metrics"
	"context"
	"flag"
	"fmt"
	"os"
	"user/app"
)

var configFile = flag.String("config", "application.yml", "config file")

func main() {
	// 1.加载配置
	flag.Parse()
	config.InitConfig(*configFile)
	// 2.启动监控
	go func() {
		err := metrics.Serve(fmt.Sprintf("0.0.0.0:%d", config.Conf.MetricPort))
		if err != nil {
			panic(err)
		}
	}()
	// 3.启动 grpc 服务端
	err := app.Run(context.Background())
	if err != nil {
		logs.Info("grpc start failed,err: ", err)
		os.Exit(-1)
	}
}
