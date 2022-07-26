package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"runtime"

	"github.com/guowenshuai/lotus_tool/baseFee"
	"github.com/guowenshuai/lotus_tool/client"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var (
	// VERSION current version
	VERSION = "v0.1.0"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	formatter := &logrus.TextFormatter{
		FullTimestamp: true,
		// 定义时间戳格式
		TimestampFormat: "2006-01-02 15:04:05",
	}
	logrus.SetFormatter(formatter)

	app := &cli.App{
		Name:    "base fee monitor",
		Version: VERSION,
		Usage:   "base fee watcher",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "api",
				Value: "172.18.5.202:1234",
				Usage: "specify lotus api address",
			},
			&cli.IntFlag{
				Name:  "port",
				Value: 80,
				Usage: "listen port",
			},
		},
	}


	ctx := context.Background()


	app.Action = func(c *cli.Context) error {
		return run(ctx, c)
	}

	err := app.Run(os.Args)
	if err != nil {
		logrus.Fatal(err)
	}
}

func run(ctx context.Context, c *cli.Context) error {
	address := c.String("api")
	listen := c.Int("port")
	nodeApi, closer, err := client.NewFullNodeRPC(ctx, "ws://"+address+"/rpc/v0", http.Header{})
	if err != nil {
		logrus.Fatalf("connecting with lotus failed: %s", err)
	}
	defer closer()
	go func() {
		baseFee.Monitor(ctx, nodeApi, fmt.Sprintf(":%d", listen))
	}()
	<-ctx.Done()
	return nil
}
