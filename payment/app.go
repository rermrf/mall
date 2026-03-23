package main

import (
	"github.com/rermrf/mall/payment/service"
	"github.com/rermrf/mall/pkg/grpcx"
)

type App struct {
	Server *grpcx.Server
	ReconJob *service.ReconciliationJob
}
