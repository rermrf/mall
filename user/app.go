package main

import (
	"github.com/rermrf/mall/pkg/grpcx"
	"github.com/rermrf/mall/pkg/saramax"
)

// App 聚合 gRPC Server 和 Consumers，方便 Wire 注入
type App struct {
	Server    *grpcx.Server
	Consumers []saramax.Consumer
}
