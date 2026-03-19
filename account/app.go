package main

import (
	"github.com/rermrf/mall/account/events"
	"github.com/rermrf/mall/pkg/grpcx"
)

type App struct {
	Server    *grpcx.Server
	Consumers []events.Consumer
}
