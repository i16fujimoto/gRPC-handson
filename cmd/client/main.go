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
)

var (
	scanner *bufio.Scanner
	client hellopb.GreetingServiceClient
)

func main() {
	fmt.Println("start gRPC client")

	scanner = bufio.NewScanner(os.Stdin)

	address := "localhost:8080"
	conn, err := grpc.Dial(
		address,
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
		fmt.Println("4: exit")
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
	// NewGreetingServiceClient関数で生成したクライアントは、サービスのHelloメソッドにリクエストを送るためのメソッドHelloを持っている
	res, err := client.Hello(context.Background(), req)
	if err != nil {
		log.Fatalf("could not greet: %v", err)
		return
	}
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
