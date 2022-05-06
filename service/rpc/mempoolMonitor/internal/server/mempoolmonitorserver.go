// Code generated by goctl. DO NOT EDIT!
// Source: mempoolmonitor.proto

package server

import (
	"context"

	"github.com/zecrey-labs/zecrey-legend/service/rpc/mempoolMonitor/internal/logic"
	"github.com/zecrey-labs/zecrey-legend/service/rpc/mempoolMonitor/internal/svc"
	"github.com/zecrey-labs/zecrey-legend/service/rpc/mempoolMonitor/mempoolMonitor"
)

type MempoolMonitorServer struct {
	svcCtx *svc.ServiceContext
	mempoolmonitor.UnimplementedMempoolMonitorServer
}

func NewMempoolMonitorServer(svcCtx *svc.ServiceContext) *MempoolMonitorServer {
	return &MempoolMonitorServer{
		svcCtx: svcCtx,
	}
}

func (s *MempoolMonitorServer) Ping(ctx context.Context, in *mempoolmonitor.Request) (*mempoolmonitor.Response, error) {
	l := logic.NewPingLogic(ctx, s.svcCtx)
	return l.Ping(in)
}
