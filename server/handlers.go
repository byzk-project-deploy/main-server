package server

import (
	"bufio"
	"net"

	"github.com/golang/protobuf/proto"
	"github.com/lucas-clemente/quic-go"
	"github.com/teamManagement/common/errors"
	"github.com/teamManagement/common/utils"
)

type ServiceHandle func(stream *StreamWrapper, conn *ConnWrapper) error

func Registry(cmd string, fn ServiceHandle) {
	serviceMap[cmd] = fn
}

var (
	serviceMap = make(map[string]ServiceHandle, 16)
)

type ConnWrapper struct {
	conn net.Conn
	rw   *bufio.ReadWriter
}

func (c *ConnWrapper) handle(stream quic.Stream) {
	defer stream.Close()
	streamWrapper := newStreamWrapper(stream)

	msg, success, err := streamWrapper.ReceiveMsg()
	if err != nil || !success {
		return
	}

	handle, ok := serviceMap[string(msg)]
	if !ok {
		_ = streamWrapper.WriteError(errors.ErrCodeCmdNotFound.Errorf("未识别的命令: %s", msg))
		return
	}

	if err = streamWrapper.WriteMsg(nil); err != nil {
		return
	}

	if err = handle(streamWrapper, c); err != nil {
		switch e := err.(type) {
		case *errors.Error:
			_ = streamWrapper.WriteError(e)
		default:
			_ = streamWrapper.WriteError(errors.ErrCodeUnknown.Error(e.Error()))
		}
		return
	}

	_ = streamWrapper.WriteMsg(nil)
}

func (c *ConnWrapper) Run() {
	if c.conn == nil {
		return
	}
	defer c.conn.Close()

	c.rw = bufio.NewReadWriter(bufio.NewReader(c.conn), bufio.NewWriter(c.conn))
	stream := newStreamWrapper(c.rw)
	for {
		msg, success, err := stream.ReceiveMsg()
		if err != nil {
			return
		}

		if !success {
			continue
		}

		cmdStr := string(msg)

		if err = stream.WriteMsg(nil); err != nil {
			continue
		}

		// 	stream, err := c.conn.AcceptStream(context.Background())
		// 	if err != nil {
		// 		return
		// 	}
		// 	go c.handle(stream)
	}
}
