package main

import (
	"github.com/promisefemi/grpc-client-pooling/proto"
	"google.golang.org/grpc"
	"log"
)

func main() {
	address := ":9000"
	conn, err := grpc.Dial(address)
	if err != nil {
		log.Fatalf("unable to connect to the client -- %s", err)
	}

	uc := proto.NewUserClient(conn)
	//uc.Set(context.Backgrouond())
	_ = uc
}
