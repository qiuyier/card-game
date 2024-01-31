package rpc

import (
	"common/config"
	"common/discovery"
	"common/logs"
	"context"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/resolver"
	"user/pb"
)

var (
	UserClient pb.UserServiceClient
)

func Init() {
	// etcd解析器 可以在 grpc 连接的时候触发 通过提供的 addr 地址 去 etcd 中查找
	r := discovery.NewResolver(config.Conf.Etcd)
	resolver.Register(r)

	userDomain := config.Conf.Domain["user"]
	initClient(userDomain.Name, userDomain.LoadBalance, &UserClient)
}

func initClient(name string, loadBalance bool, client any) {
	// 找grpc服务地址
	addr := fmt.Sprintf("etcd:///%s", name)

	// 负载均衡
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials())}
	if loadBalance {
		opts = append(opts, grpc.WithDefaultServiceConfig(fmt.Sprintf(`{"LoadBalancingPolicy": "%s"}`, "round_robin")))
	}

	conn, err := grpc.DialContext(context.TODO(), addr, opts...)
	if err != nil {
		logs.Fatal("rpc connect etcd err: %v", err)
	}

	switch c := client.(type) {
	case *pb.UserServiceClient:
		*c = pb.NewUserServiceClient(conn)
	default:
		logs.Fatal("unsupported client type")
	}
}
