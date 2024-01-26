package discovery

import (
	"common/config"
	"common/logs"
	"context"
	"encoding/json"
	clientv3 "go.etcd.io/etcd/client/v3"
	"time"
)

type Register struct {
	etcdClient  *clientv3.Client                        // etcd连接
	leaseId     clientv3.LeaseID                        // 租约 ID
	DialTimeout int                                     // 超时时间 秒
	ttl         int                                     // 租约时间 秒
	keepAliveCh <-chan *clientv3.LeaseKeepAliveResponse // 心跳
	info        Server                                  //注册的服务信息
	closeCh     chan struct{}
}

func NewRegister() *Register {
	return &Register{
		DialTimeout: 3,
	}
}

func (r *Register) Register(conf config.EtcdConf) error {
	logs.Info("register etcd...")
	// 注册信息
	info := Server{
		Name:    conf.Register.Name,
		Addr:    conf.Register.Addr,
		Weight:  conf.Register.Weight,
		Version: conf.Register.Version,
		Ttl:     conf.Register.Ttl,
	}
	r.info = info

	// 建立连接
	var err error
	r.etcdClient, err = clientv3.New(clientv3.Config{
		Endpoints:   conf.Addrs,
		DialTimeout: time.Duration(r.DialTimeout) * time.Second,
	})
	if err != nil {
		return err
	}

	if err = r.register(); err != nil {
		return err
	}

	r.closeCh = make(chan struct{})

	// 使用协程 根据心跳结果做相应操作
	go r.watcher()

	return nil
}

func (r *Register) register() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(r.DialTimeout))
	defer cancel()

	var err error
	// 1.创建租约
	if err = r.createLease(ctx, r.info.Ttl); err != nil {
		return err
	}

	// 2.心跳检测
	if r.keepAliveCh, err = r.keepAlive(); err != nil {
		return err
	}

	// 3.绑定租约
	data, _ := json.Marshal(r.info)

	return r.bindLease(ctx, r.info.BuildRegisterKey(), string(data))
}

// createLease 创建租约
func (r *Register) createLease(ctx context.Context, ttl int64) error {
	grant, err := r.etcdClient.Grant(ctx, ttl)
	if err != nil {
		logs.Error("createLease failed,err: %v", err)
		return err
	}
	r.leaseId = grant.ID

	return nil
}

// bindLease 绑定租约
func (r *Register) bindLease(ctx context.Context, key, value string) error {
	_, err := r.etcdClient.Put(ctx, key, value, clientv3.WithLease(r.leaseId))
	if err != nil {
		logs.Error("bindLease failed,err: %v", err)
		return err
	}
	return nil
}

// keepAlive 心跳 确保服务正常
func (r *Register) keepAlive() (<-chan *clientv3.LeaseKeepAliveResponse, error) {
	keepAliveResponses, err := r.etcdClient.KeepAlive(context.Background(), r.leaseId)
	if err != nil {
		logs.Error("keepAlive failed,err: %v", err)
		return keepAliveResponses, err
	}
	return keepAliveResponses, nil
}

// watcher 监听 续租 注销等
func (r *Register) watcher() {
	// 租约到期 检查是否需要自动注册
	ticker := time.NewTicker(time.Duration(r.info.Ttl) * time.Second)

	for {
		select {
		case <-r.closeCh:
			logs.Info("stop register...")

			// 注销
			if err := r.unregister(); err != nil {
				logs.Error("close and unregister failed,err: %v", err)
			}

			// 租约撤销
			if _, err := r.etcdClient.Revoke(context.Background(), r.leaseId); err != nil {
				logs.Error("close and revoke lease failed,err: %v", err)
			}
		case <-r.keepAliveCh:
			// 续租
		case <-ticker.C:
			if r.keepAliveCh == nil {
				// 租约到期 检测是否需要自动注册
				if err := r.register(); err != nil {
					logs.Error("ticker register failed,err: %v", err)
				}
			}
		}

	}
}

func (r *Register) unregister() error {
	_, err := r.etcdClient.Delete(context.Background(), r.info.BuildRegisterKey())
	return err
}

func (r *Register) CloseEtcd() {
	r.closeCh <- struct{}{}
}
