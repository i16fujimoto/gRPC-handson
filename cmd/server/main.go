package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"time"
	"errors"
	"io"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/genproto/googleapis/rpc/errdetails"

	hellopb "mygrpc/pkg/grpc"
)

type myServer struct {
	hellopb.UnimplementedGreetingServiceServer
}

func (s *myServer) Hello(ctx context.Context, in *hellopb.HelloRequest) (*hellopb.HelloResponse, error) {
	// log.Printf("received: %v\n", in.GetName())
	// return &hellopb.HelloResponse{Message: fmt.Sprintf("Hello, %s!", in.GetName())}, nil

	stat := status.New(codes.Unknown, "unknown error occurred")
	stat, _ = stat.WithDetails(&errdetails.DebugInfo{
		Detail: "detail reason of err",
	})
	err := stat.Err()
	return nil, err
}

func (s *myServer) HelloServerStream(in *hellopb.HelloRequest, stream hellopb.GreetingService_HelloServerStreamServer) error {
	resCount := 5
	for i := 0; i < resCount; i++ {
		// レスポンスを返したいときには、Sendメソッドの引数にHelloResponse型を渡すことでそれがクライアントに送信される
		if err := stream.Send(&hellopb.HelloResponse{Message: fmt.Sprintf("Hello, %s! [%d]", in.GetName(), i)}); err != nil {
			return err
		}
		time.Sleep(time.Second * 1)
	}
	// return文でメソッドを終了させる=ストリームの終わり
	return nil
}

func (s *myServer) HelloClientStream(stream hellopb.GreetingService_HelloClientStreamServer) error {
	nameList := make([]string, 0)
	for {
		// streamのRecvメソッドを呼び出してリクエスト内容を取得する
		req, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			// リクエストを全て受け取った後の処理
			message := fmt.Sprintf("Hello, %s!", nameList)
			return stream.SendAndClose(&hellopb.HelloResponse{Message: message})
		}
		if err != nil {
			return err
		}
		nameList = append(nameList, req.GetName())
	}
}

func (s *myServer) HelloBiStreams(stream hellopb.GreetingService_HelloBiStreamsServer) error {
	for {
		// クライアントからのリクエストを受け取るためのメソッドRecvを呼び出す
		req, err := stream.Recv()
		// 得られたエラーがio.EOFならばもうリクエストは送られてこない
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		log.Printf("received: %v\n", req.GetName())
		// サーバーからのレスポンスを送信するためのメソッドSendを呼び出す
		if err := stream.Send(&hellopb.HelloResponse{Message: fmt.Sprintf("Hello, %s!", req.GetName())}); err != nil {
			return err
		}
	}
}

func NewMyServer() *myServer {
	return &myServer{}
}

func main() {
	port := "8080"
	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		panic(err)
	}

	server := grpc.NewServer()

	// Register Service
	hellopb.RegisterGreetingServiceServer(server, NewMyServer())

	// Register Reflection Service
	/*-------------------------------------------------------------
	元からprotoファイルによるメッセージ型の定義を知らないgRPCurlコマンドは、
	代わりに「gRPCサーバーそのものから、protoファイルの情報を取得する」ことで
	「シリアライズのルール」を知り通信します。
	そしてその「gRPCサーバーそのものから、protoファイルの情報を取得する」ための機能がサーバーリフレクション
	-------------------------------------------------------------*/
	reflection.Register(server)

	go func() {
		log.Printf("start gRPC server on port %s", port)
		_ = server.Serve(listener)
	}()

	// Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	log.Println("stopping gRPC server...")
	server.GracefulStop()
}
