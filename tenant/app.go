package main

import "github.com/rermrf/mall/pkg/grpcx"

// App 聚合 gRPC Server，方便 Wire 注入
type App struct {
	Server *grpcx.Server
}
