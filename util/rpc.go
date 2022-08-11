package util

import (
	"context"
	"reflect"
	"runtime"
	"time"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

func MakeRPC[T any, S any](req T, fn func(context.Context, T, ...grpc.CallOption) (S, error)) S {
	wf := 1
	for {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		r, err := fn(ctx, req)
		if err != nil {
			log.WithFields(log.Fields{
				"rpc name": runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name(),
				"error":    err,
			}).Warn("cannot make rpc call")
			time.Sleep(time.Second * time.Duration(wf))
			wf *= 2
		} else {
			log.WithFields(log.Fields{
				"rpc name": runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name(),
			}).Debug("rpc call")
			return r
		}
	}
}