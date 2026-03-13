package main

import (
	"github.com/rermrf/mall/pkg/grpcx"
	"github.com/rermrf/mall/pkg/saramax"
)

type App struct {
	Server    *grpcx.Server
	Consumers []saramax.Consumer
}
