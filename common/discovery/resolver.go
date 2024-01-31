package discovery

import (
	"common/config"
	"common/logs"
	"context"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc/attributes"
	"google.golang.org/grpc/resolver"
	"time"
)

type Resolver struct {
	conf         config.EtcdConf
	etcdClient   *clientv3.Client
	DialTimeout  int
	closeCh      chan struct{}
	key          string
	cc           resolver.ClientConn
	srvAdderList []resolver.Address
	watchCh      clientv3.WatchChan
}

// Build 当 grpc.dial的时候 同步调用该方法
func (r Resolver) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	// 获取到调用的 key 连接 etcd 获取其 value

	// 1.连接 etcd
	r.cc = cc

	var err error
	r.etcdClient, err = clientv3.New(clientv3.Config{
		Endpoints:   r.conf.Addrs,
		DialTimeout: time.Duration(r.DialTimeout) * time.Second,
	})
	if err != nil {
		logs.Fatal("grpc client connect etcd err: %v", err)
	}

	r.closeCh = make(chan struct{})

	// 2.根据 key去 value
	r.key = target.URL.Path

	if err = r.sync(); err != nil {
		return nil, err
	}

	// 3.当节点变动时，监听实时更新信息
	go r.watch()

	return nil, nil
}

func (r Resolver) Scheme() string {
	//TODO implement me
	return "etcd"
}

func (r Resolver) sync() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(r.conf.RWTimeout)*time.Second)
	defer cancel()

	res, err := r.etcdClient.Get(ctx, r.key, clientv3.WithPrefix())
	if err != nil {
		logs.Error("grpc client get etcd failed, name=%s, err: %v", r.key, err)
		return err
	}

	r.srvAdderList = make([]resolver.Address, res.Count)

	for _, v := range res.Kvs {
		server, err := ParseValue(v.Value)
		if err != nil {
			logs.Error("grpc client parse etcd value failed, name=%s, err: %v", r.key, err)
			continue
		}

		r.srvAdderList = append(r.srvAdderList, resolver.Address{
			Addr:       server.Addr,
			Attributes: attributes.New("weight", server.Weight),
		})
	}

	if len(r.srvAdderList) == 0 {
		return nil
	}

	err = r.cc.UpdateState(resolver.State{
		Addresses: r.srvAdderList,
	})
	if err != nil {
		logs.Error("grpc client update state value failed, name=%s, err: %v", r.key, err)
		return err
	}

	return nil
}

func (r Resolver) watch() {
	// 1.定时同步数据
	// 2.监听节点的事件 从而触发不同操作
	// 3.监听 close 事件 关闭 etcd

	r.watchCh = r.etcdClient.Watch(context.Background(), r.key, clientv3.WithPrefix())

	ticker := time.NewTicker(time.Minute)

	for {
		select {
		case <-r.closeCh:
			r.Close()
		case res, ok := <-r.watchCh:
			if ok {
				//
				r.update(res.Events)
			}

		case <-ticker.C:
			if err := r.sync(); err != nil {
				logs.Error("watch sync failed,err: %v", err)
			}
		}
	}
}

func (r Resolver) update(events []*clientv3.Event) {
	for _, ev := range events {
		switch ev.Type {
		case clientv3.EventTypePut:
			// put key value
			server, err := ParseValue(ev.Kv.Value)
			if err != nil {
				logs.Error("grpc client update(EventTypePut) parse etcd value failed, name=%s, err: %v", r.key, err)
			}

			addr := resolver.Address{
				Addr:       server.Addr,
				Attributes: attributes.New("weight", server.Weight),
			}

			// 节点不存在就添加
			if !Exist(r.srvAdderList, addr) {
				r.srvAdderList = append(r.srvAdderList, addr)

				err = r.cc.UpdateState(resolver.State{
					Addresses: r.srvAdderList,
				})
				if err != nil {
					logs.Error("grpc client update(EventTypePut) state value failed, name=%s, err: %v", r.key, err)
				}
			}

		case clientv3.EventTypeDelete:
			// 接收到 delete操作 删除r.srvAddrList 其中匹配的值
			server, err := ParseKey(string(ev.Kv.Key))
			if err != nil {
				logs.Error("grpc client update(EventTypeDelete) parse etcd value failed, name=%s, err: %v", r.key, err)
			}

			addr := resolver.Address{Addr: server.Addr}

			if list, ok := Remove(r.srvAdderList, addr); ok {
				r.srvAdderList = list

				err = r.cc.UpdateState(resolver.State{
					Addresses: r.srvAdderList,
				})
				if err != nil {
					logs.Error("grpc client update(EventTypeDelete) state value failed, name=%s, err: %v", r.key, err)
				}
			}
		}
	}
}

func (r Resolver) Close() {
	if r.etcdClient != nil {
		err := r.etcdClient.Close()
		if err != nil {
			logs.Error("resolver close etcd err:%v", err)
		}
	}
}

func Exist(list []resolver.Address, addr resolver.Address) bool {
	for i := range list {
		if list[i].Addr == addr.Addr {
			return true
		}
	}

	return false
}

func Remove(list []resolver.Address, addr resolver.Address) ([]resolver.Address, bool) {
	for i := range list {
		if list[i].Addr == addr.Addr {
			list[i] = list[len(list)-1]
			return list[:len(list)-1], true
		}
	}
	return nil, false
}

func NewResolver(conf config.EtcdConf) *Resolver {
	return &Resolver{
		conf: conf,
	}
}
