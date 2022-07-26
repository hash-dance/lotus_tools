/*Package util give common utils
 */
package util

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"
)

// SigTermCancelContext listen cancel signal
func SigTermCancelContext(ctx context.Context) context.Context {
	term := make(chan os.Signal)
	signal.Notify(term, os.Interrupt, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(ctx)

	go func() {
		select {
		case <-term:
			logrus.Infof("Received SIGTERM, canceling")
			cancel()
		case <-ctx.Done():
		}
	}()

	return ctx
}
