package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"

	hellopb "mygrpc/pkg/grpc"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	"google.golang.org/grpc/metadata"

	// これを忘れると、実行時にditailに[proto: not found]というエラーが出てしまう
	// デシリアライズする際に、protoファイルを参照するために必要
	_ "google.golang.org/genproto/googleapis/rpc/errdetails"
)

var (
	scanner *bufio.Scanner
	client  hellopb.GreetingServiceClient
)

func main() {
	fmt.Println("start gRPC client")

	scanner = bufio.NewScanner(os.Stdin)

	address := "localhost:8080"
	conn, err := grpc.Dial(
		address,

		grpc.WithUnaryInterceptor(myUnaryClientInterceptor1()),
		grpc.WithStreamInterceptor(myStreamClientInterceptor1()),

		// 昔はgrpc.WithInsecure()で同じことをしていましたが、現在google.golang.org/grpcパッケージのWithInsecure()関数はDeprecatedになっています
		grpc.WithTransportCredentials(insecure.NewCredentials()), // insecure: コネクションでSSL/TLSを使用しない
		grpc.WithBlock(), // コネクションが確立されるまで待機する(同期処理をする)
	)
	if err != nil {
		log.Fatalf("did not connect: %v", err)
		return
	}
	defer conn.Close()

	client = hellopb.NewGreetingServiceClient(conn)

	for {
		fmt.Println("1: send Request")
		fmt.Println("2: HelloServerStream")
		fmt.Println("3: HelloClientStream")
		fmt.Println("4: HelloBiStream")
		fmt.Println("5: exit")
		fmt.Print("please enter >")

		scanner.Scan()
		input := scanner.Text()

		switch input {
		case "1":
			Hello()
		case "2":
			HelloServerStream()
		case "3":
			HelloClientStream()
		case "4":
			HelloBiStreams()
		case "5":
			fmt.Println("exit")
			return
		}
	}
}

func Hello() {
	fmt.Print("please enter your name >")
	scanner.Scan()
	name := scanner.Text()

	req := &hellopb.HelloRequest{Name: name}

	ctx := context.Background()
	md := metadata.New(map[string]string{"type": "unary", "from": "client"})
	ctx = metadata.NewOutgoingContext(ctx, md)

	// NewGreetingServiceClient関数で生成したクライアントは、サービスのHelloメソッドにリクエストを送るためのメソッドHelloを持っている
	var header, trailer metadata.MD
	// header, trailerを受け取るためのgrpc.Headerとgrpc.Trailerを使って、ヘッダーとトレーラーを受け取る
	// このCallOption付きでメソッドを呼び出すと、grpc.Header・grpc.Trailer関数に引数として渡したメタデータ型に、レスポンス受信時に取得したヘッダー・トレーラーのデータが格納される
	res, err := client.Hello(ctx, req, grpc.Header(&header), grpc.Trailer(&trailer))

	if err != nil {
		if stat, ok := status.FromError(err); ok {
			fmt.Printf("code: %s\n", stat.Code())
			fmt.Printf("message: %s\n", stat.Message())
			fmt.Printf("details: %s\n", stat.Details())
			return
		} else {
			fmt.Println(err)
			return
		}
	}

	fmt.Println(header)
	fmt.Println(trailer)
	log.Printf("Greeting: %s", res.GetMessage())
}

func HelloServerStream() {
	fmt.Println("please enter your name >")
	scanner.Scan()
	name := scanner.Text()

	req := &hellopb.HelloRequest{Name: name}
	// NewGreetingServiceClient関数で生成したクライアントは、サービスのHelloServerStreamメソッドにリクエストを送るためのメソッドHelloServerStreamを持っている
	// サーバーから複数回レスポンスを受け取るためのストリームを得る
	stream, err := client.HelloServerStream(context.Background(), req)
	if err != nil {
		fmt.Println(err)
		return
	}

	for {
		// サーバーからのレスポンスを受信するためのメソッドRecvを呼び出す
		res, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			fmt.Println("all the responses have already received.")
			break
		}
		if err != nil {
			log.Fatalf("could not greet: %v", err)
			return
		}
		log.Println(res)
	}
}

func HelloClientStream() {
	// サーバーに複数回リクエストを送るためのストリームを得る
	stream, err := client.HelloClientStream(context.Background())
	if err != nil {
		fmt.Println(err)
		return
	}

	sendCount := 5
	fmt.Printf("Please enter %d names.\n", sendCount)
	for i := 0; i < sendCount; i++ {
		scanner.Scan()
		name := scanner.Text()

		req := &hellopb.HelloRequest{Name: name}
		// クライアントからリクエストを送信するためのメソッドSendを呼び出す
		// ストリームを通じてリクエストを送信する
		if err := stream.Send(req); err != nil {
			log.Fatalf("could not send: %v", err)
			return
		}
	}

	// クライアントからのリクエストを全て送信した後、レスポンスを受け取るためのメソッドRecvを呼び出す
	res, err := stream.CloseAndRecv()
	if err != nil {
		log.Fatalf("could not greet: %v", err)
		return
	}
	log.Printf("Greeting: %s\n", res.GetMessage())
}

func HelloBiStreams() {
	ctx := context.Background()
	md := metadata.New(map[string]string{"type": "stream", "from": "client"})
	ctx = metadata.NewOutgoingContext(ctx, md)

	stream, err := client.HelloBiStreams(ctx)
	if err != nil {
		fmt.Println(err)
		return
	}

	sendNum := 5
	sendCount := 0
	fmt.Printf("Please enter %d names.\n", sendNum)
	var sendEnd, recvEnd bool
	for !(sendEnd && recvEnd) {
		if !sendEnd {
			scanner.Scan()
			name := scanner.Text()

			sendCount++

			req := &hellopb.HelloRequest{Name: name}
			if err := stream.Send(req); err != nil {
				log.Fatalf("could not send: %v", err)
				return
			}

			if sendCount == sendNum {
				sendEnd = true
				/*
					client.HelloBiStreamsから得られるストリームは、SendメソッドとRecvメソッド以外にも、
					grpc.ClientStreamインタフェースが持つメソッドセットも使うことができます。
					CloseSendメソッドは、まさにgrpc.ClientStreamインターフェース由来のメソッドです。
				*/
				if err := stream.CloseSend(); err != nil {
					fmt.Println(err)
				}
			}
		}

		var headerMD metadata.MD
		if !recvEnd {
			if headerMD == nil {
				headerMD, err = stream.Header()
				if err != nil {
					fmt.Println(err)
				} else {
					fmt.Println(headerMD)
				}
			}
			res, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				recvEnd = true
			} else if err != nil {
				log.Fatalf("could not greet: %v", err)
				return
			}
			log.Println(res.GetMessage())
		}
	}
	trailerMD := stream.Trailer()
	fmt.Println(trailerMD)
}
