package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	hellopb "mygrpc/pkg/grpc"
)

type myServer struct {
	hellopb.UnimplementedGreetingServiceServer
}

func (s *myServer) Hello(ctx context.Context, in *hellopb.HelloRequest) (*hellopb.HelloResponse, error) {
	log.Printf("received: %v\n", in.GetName())
	return &hellopb.HelloResponse{Message: fmt.Sprintf("Hello, %s!", in.GetName())}, nil
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
	reflection.Register(server)

	go func() {
		log.Printf("start gRPC server on port %s", port)
		server.Serve(listener)
	}()

	// Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	log.Println("stopping gRPC server...")
	server.GracefulStop()
}
