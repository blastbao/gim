package connect

import (
	"context"
	"gim/config"
	"gim/pkg/logger"
	"gim/pkg/pb"
	"gim/pkg/rpc"
	"time"

	"go.uber.org/zap"

	"github.com/alberliu/gn"
)

var encoder = gn.NewHeaderLenEncoder(2, 1024)

var server *gn.Server

// StartTCPServer 启动TCP服务器
func StartTCPServer() {
	gn.SetLogger(logger.Sugar)

	var err error
	server, err = gn.NewServer(config.Config.TCPListenAddr, &handler{},
		gn.WithDecoder(gn.NewHeaderLenDecoder(2)),
		gn.WithEncoder(gn.NewHeaderLenEncoder(2, 1024)),
		gn.WithReadBufferLen(256),
		gn.WithTimeout(11*time.Minute),
		gn.WithAcceptGNum(10),
		gn.WithIOGNum(100))
	if err != nil {
		logger.Sugar.Error(err)
		panic(err)
	}

	server.Run()
}

type handler struct{}

func (*handler) OnConnect(c *gn.Conn) {
	// 初始化连接数据
	conn := &Conn{
		CoonType: CoonTypeTCP,
		TCP:      c,
	}
	c.SetData(conn)
	logger.Logger.Debug("connect:", zap.Int32("fd", c.GetFd()), zap.String("addr", c.GetAddr()))
}

func (*handler) OnMessage(c *gn.Conn, bytes []byte) {
	conn := c.GetData().(*Conn)
	conn.HandleMessage(bytes)
}

func (*handler) OnClose(c *gn.Conn, err error) {
	conn, ok := c.GetData().(*Conn)
	if !ok || conn == nil {
		return
	}
	logger.Logger.Debug("close", zap.String("addr", c.GetAddr()), zap.Int64("user_id", conn.UserId),
		zap.Int64("device_id", conn.DeviceId), zap.Error(err))

	DeleteConn(conn.DeviceId)

	if conn.UserId != 0 {
		_, _ = rpc.GetLogicIntClient().Offline(context.TODO(), &pb.OfflineReq{
			UserId:     conn.UserId,
			DeviceId:   conn.DeviceId,
			ClientAddr: c.GetAddr(),
		})
	}
}
