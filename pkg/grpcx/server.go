package grpcx

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/rermrf/emo/logger"
	"github.com/rermrf/mall/pkg/netx"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/naming/endpoints"
	"google.golang.org/grpc"
)

type Server struct {
	*grpc.Server
	Port      int
	EtcdAddrs []string
	Name      string
	L         logger.Logger

	etcdClient *clientv3.Client
	etcdKey    string
	cancel     context.CancelFunc
}

func (s *Server) Serve() error {
	lis, err := net.Listen("tcp", ":"+strconv.Itoa(s.Port))
	if err != nil {
		return err
	}
	err = s.register()
	if err != nil {
		return err
	}
	return s.Server.Serve(lis)
}

func (s *Server) register() error {
	client, err := clientv3.New(clientv3.Config{
		Endpoints: s.EtcdAddrs,
	})
	if err != nil {
		return err
	}
	s.etcdClient = client

	em, err := endpoints.NewManager(client, "service/"+s.Name)
	if err != nil {
		return err
	}

	ip := netx.GetOutboundIP()
	addr := ip + ":" + strconv.Itoa(s.Port)
	s.etcdKey = "service/" + s.Name + "/" + addr

	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel

	// 租约 TTL 10s
	leaseResp, err := client.Grant(ctx, 10)
	if err != nil {
		cancel()
		return err
	}

	err = em.AddEndpoint(ctx, s.etcdKey, endpoints.Endpoint{
		Addr: addr,
	}, clientv3.WithLease(leaseResp.ID))
	if err != nil {
		cancel()
		return err
	}

	// 自动续约
	ch, err := client.KeepAlive(ctx, leaseResp.ID)
	if err != nil {
		cancel()
		return err
	}
	go func() {
		for {
			select {
			case resp, ok := <-ch:
				if !ok {
					return
				}
				_ = resp
			case <-ctx.Done():
				return
			}
		}
	}()

	s.L.Info("gRPC 服务注册成功",
		logger.String("name", s.Name),
		logger.String("addr", addr),
	)
	return nil
}

func (s *Server) Close() {
	if s.cancel != nil {
		s.cancel()
	}
	if s.etcdClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		em, err := endpoints.NewManager(s.etcdClient, "service/"+s.Name)
		if err == nil {
			_ = em.DeleteEndpoint(ctx, s.etcdKey)
		}
		_ = s.etcdClient.Close()
	}
	s.GracefulStop()
	fmt.Println("gRPC 服务已关闭")
}
