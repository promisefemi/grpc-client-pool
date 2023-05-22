package main

import (
	"context"
	"fmt"
	"github.com/promisefemi/grpc-client-pooling"
	"github.com/promisefemi/grpc-client-pooling/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"time"
)

func main() {
	poolConfig := &grpc_pooling.PoolConfig{
		MaxOpenConnection:     10,
		MaxIdleConnection:     5,
		ConnectionQueueLength: 10000,
		NewClientDuration:     time.Second * 5,
		Address:               ":9000",
		ConfigOptions: []grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithBlock(),
		},
	}

	connPool := grpc_pooling.NewClientPool(poolConfig)

	for {
		client, err := connPool.Get()
		if err != nil {
			log.Fatalf("%s", err)
		}

		userMessage := &proto.UserMessage{
			FirstName: "Promise",
			LastName:  "Femi",
			Email:     "",
		}

		uc := proto.NewUserClient(client.Conn)
		response, err := uc.Set(context.Background(), userMessage)
		if err != nil {
			fmt.Printf("error unable to set user -- %s -- %d\n", err, connPool.GetNumberOfOpenConnections())
		} else {
			fmt.Printf("%+v -- %d \n", response, connPool.GetNumberOfOpenConnections())
		}
		client.Release()
	}
}
