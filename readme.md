### GRPC Client Pooling
a simple library to help with creating/managing grpc connections for gRPC client studs

#### Install
```bash
go get github.com/promisefemi/grpc-client-pool
```

#### Usage
```go 
poolConfig := &pool.PoolConfig{
    // Server address
    Address: ":9000",
    //max number of open connections allowed
    MaxOpenConnection: 10,
    // max number of client connections
    MaxIdleConnection: 5,
    // number of connection that can be queued at once
    ConnectionQueueLength: 10000,
    // Duration of client connect (useful when you want to connect to the server with grpc.DialContext)
    NewClientDuration: time.Second * 5,
    // Client connect options
    ConfigOptions: []grpc.DialOption{
        grpc.WithTransportCredentials(insecure.NewCredentials()),
        grpc.WithBlock(),
    },
}

//New client pool
clientPool := pool.NewClientPool(poolConfig)

//Get a new connection type ( idle connection would be reused or new connection would be created )
conn, err := clientPool.Get()
//Handle error
if err != nil {
...
}

//use connection as conn.Conn
uc := proto.NewUserClient(conn.Conn)

// remember to release connection when done
conn.Release()


```