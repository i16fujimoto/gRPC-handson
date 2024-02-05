package main

import (
	"context"
	"errors"
	"io"
	"log"

	"google.golang.org/grpc"
)

/*-----------------------------------
ccは、RPCが呼び出されたClientConnである。
streamerは、ClientStreamを作成するハンドラであり、これを呼び出すのはインターセプタの責任である。

StreamClientInterceptorは、すべてのI/O操作をインターセプトするカスタムClientStreamを返すことができる。
返されるエラーは、ステータス・パッケージと互換性がなければならない。
-----------------------------------*/

/*-----------------------------------
クライアントストリームが担う処理
インターセプタによって得られるクライアントストリームは、主に以下の処理を担うことになります。

・リクエスト送信処理
・レスポンス受信処理
・ストリームclose処理
--------------------------------------*/

type myClientStreamWrapper1 struct {
	grpc.ClientStream
}

func myStreamClientInterceptor1() grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		// ストリームがopenされる前に行われる前処理
		log.Println("[pre] my stream client interceptor 1", method)

		// ストリームを生成 -> 返り値として返す
		// このストリームを用いて、クライアントは送受信処理を行う
		cs, err := streamer(ctx, desc, cc, method, opts...)
		return &myClientStreamWrapper1{cs}, err
	}
}

func (s *myClientStreamWrapper1) RecvMsg(m interface{}) error {
	// レスポンス受信処理
	err := s.ClientStream.RecvMsg(m)
	// レスポンス受信後に割り込ませる処理
	if !errors.Is(err, io.EOF) {
		log.Println("[post message] my stream client interceptor 1: ", m)
	}
	return err
}

func (s *myClientStreamWrapper1) SendMsg(m interface{}) error {
	// リクエスト送信前に割り込ませる処理
	log.Println("[pre message] my stream client interceptor 1: ", m)
	// リクエスト送信処理
	return s.ClientStream.SendMsg(m)
}

func (s *myClientStreamWrapper1) CloseSend() error {
	// ストリームをclose
	err := s.ClientStream.CloseSend()
	// ストリームがcloseされた後に行われる後処理
	log.Println("[post] my stream client interceptor 1")
	return err
}
