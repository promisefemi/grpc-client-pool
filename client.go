package grpc_client_pool

import "google.golang.org/grpc"

type ClientCon struct {
	id   string
	pool *ClientPool
	Conn *grpc.ClientConn
}

func (c *ClientCon) Release() {
	c.pool.put(c)
}
