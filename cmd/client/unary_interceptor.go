package main

import (
	"context"
	"log"

	"google.golang.org/grpc"
)

// invokerはgRPCのクライアントのメソッドを実行する関数
// methodはRPC名
// reqとreplyは対応するリクエストとレスポンス・メッセージ

func myUnaryClientInterceptor1() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		log.Println("[pre] my unary client interceptor 1: ", method)
		err := invoker(ctx, method, req, reply, cc, opts...)
		log.Println("[post] my unary client interceptor 1: ", reply)
		return err
	}
}
