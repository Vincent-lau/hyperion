package util

import (
	"context"
	"reflect"
	"runtime"
	"time"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

func RetryRPC[T any, S any](req T, fn func(context.Context, T, ...grpc.CallOption) (S, error)) S {
	wf := 1
	for {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		r, err := fn(ctx, req)
		if err != nil {
			log.WithFields(log.Fields{
				"rpc name":          runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name(),
				"error":             err,
				"sleep for seconds": wf,
			}).Warn("cannot make rpc call")
			time.Sleep(time.Second * time.Duration(wf))
			wf *= 2
		} else {
			// log.WithFields(log.Fields{
			// 	"rpc name": runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name(),
			// }).Debug("rpc call")
			return r
		}
	}
}

func MakeRPC[T any, S any](req T, fn func(context.Context, T, ...grpc.CallOption) (S, error)) (S, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	r, err := fn(ctx, req)

	if err != nil {
		log.WithFields(log.Fields{
			"rpc name": runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name(),
			"error":    err,
			"retry":    false,
		}).Warn("cannot make rpc call")
	}

	return r, err
}
