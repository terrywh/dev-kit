package stream

import (
	"context"
	"log"
	"time"

	"github.com/quic-go/quic-go"
)

type Client struct {
	handler ConnectionHandler
	options DialOptions
}

type ClientOptions struct {
	Handler ConnectionHandler
	DialOptions
}

func NewClient(options *ClientOptions) (cli *Client) {
	cli = &Client{
		handler: options.Handler,
		options: options.DialOptions,
	}
	return cli
}

func (cli *Client) Serve(ctx context.Context) {
	var conn quic.Connection
	// var device_id entity.DeviceID
	var err error
SERVING:
	for {
		conn /* device_id */, _, err = DefaultTransport.Dial(ctx, &DialOptions{
			Address:     cli.options.Address, // TODO 公共 REGISTRY 服务
			Certificate: cli.options.Certificate,
			PrivateKey:  cli.options.PrivateKey,
			Retry:       3,
			Backoff:     1200 * time.Millisecond,
		})
		if ctx.Err() != nil {
			break SERVING
		}
		if err != nil {
			log.Println("<Client.Serve> failed to dial registry: ", err)
			continue
		}
		cli.handler.ServeConn(ctx, conn)
	}
}

func (cli *Client) Close() error {
	return cli.handler.Close()
}