package pool

import (
	"context"
	"errors"
	"fmt"
	"google.golang.org/grpc"
	"sync"
	"time"
)

var (
	ErrConnectionWaitTimeout error = errors.New("connection queue wait timeout")
)

type queueChan struct {
	connectionChan chan *clientCon
	errorChan      chan error
}

type clientCon struct {
	id   string
	pool *ClientPool
	conn *grpc.ClientConn
}

func (c *clientCon) Release() {
	c.pool.put(c)
}

type ClientPool struct {
	mu sync.Mutex

	address             string
	configOptions       []grpc.DialOption
	maxOpenConnection   int
	maxIdleConnection   int
	idleConnections     map[string]*clientCon
	numOfOpenConnection int
	connectionQueue     chan *queueChan
	clientDuration      time.Duration
}

type PoolConfig struct {
	MaxOpenConnection     int
	MaxIdleConnection     int
	ConnectionQueueLength int
	NewClientDuration     time.Duration
	Address               string
	ConfigOptions         []grpc.DialOption
}

func NewClientPool(config *PoolConfig) *ClientPool {
	clientPool := &ClientPool{
		mu:                  sync.Mutex{},
		address:             config.Address,
		configOptions:       config.ConfigOptions,
		maxOpenConnection:   config.MaxOpenConnection,
		maxIdleConnection:   config.MaxOpenConnection,
		numOfOpenConnection: 0,
		connectionQueue:     make(chan *queueChan, config.ConnectionQueueLength),
		clientDuration:      config.NewClientDuration,
	}

	go clientPool.handleConnectionRequest()
	return clientPool
}

func (cp *ClientPool) put(conn *clientCon) {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	conn.conn.GetState()
	if cp.maxIdleConnection >= len(cp.idleConnections) {
		cp.idleConnections[conn.id] = conn
	} else {
		_ = conn.conn.Close()
		cp.numOfOpenConnection--
	}
}

func (cp *ClientPool) get() (*clientCon, error) {
	cp.mu.Lock()

	//	first find an idle connection and return
	// else check if the number of idle connection is more than maxidleConnections if its more queue connection request
	// or create a new connection

	if len(cp.idleConnections) > 0 {
		for _, val := range cp.idleConnections {
			delete(cp.idleConnections, val.id)
			cp.numOfOpenConnection++
			cp.mu.Unlock()
			return val, nil
		}
	}

	if cp.maxOpenConnection > 0 && cp.numOfOpenConnection >= cp.maxOpenConnection {
		queueRequest := &queueChan{
			connectionChan: make(chan *clientCon),
			errorChan:      make(chan error),
		}

		cp.connectionQueue <- queueRequest

		select {
		case conn := <-queueRequest.connectionChan:
			cp.numOfOpenConnection++
			cp.mu.Unlock()
			return conn, nil
		case err := <-queueRequest.errorChan:
			cp.mu.Unlock()
			return nil, err
		}

	}
	conn, err := cp.openConnection()
	if err != nil {
		return nil, err
	}

	cp.numOfOpenConnection++
	cp.mu.Unlock()
	return conn, nil

}

func (cp *ClientPool) openConnection() (*clientCon, error) {
	var newConn *grpc.ClientConn
	var err error

	//check if user set a timeout when creating new connections
	if cp.clientDuration > time.Duration(0) {
		timeout := time.After(cp.clientDuration)
		ctx, cancel := context.WithTimeout(context.Background(), cp.clientDuration)
		select {
		case <-timeout:
			cancel()
		default:
			newConn, err = grpc.DialContext(ctx, cp.address, cp.configOptions...)
			cancel()
		}
	} else {
		newConn, err = grpc.Dial(cp.address, cp.configOptions...)
	}
	if err != nil {
		return nil, err
	}

	return &clientCon{
		id:   fmt.Sprintf("%v", time.Now().Unix()),
		pool: cp,
		conn: newConn,
	}, nil
}

func (cp *ClientPool) handleConnectionRequest() {
	for rq := range cp.connectionQueue {

		var (
			hasTimedOut  = false
			hasCompleted = false
			timeout      = time.After(time.Duration(3) * time.Second)
		)
		//continually try to get/create a connection until timeout or connection has been returned
		for {

			if hasCompleted || hasTimedOut {
				break
			}
			//continually check for timeout or try to get/create a connection
			select {
			case <-timeout:
				hasTimedOut = true
				rq.errorChan <- ErrConnectionWaitTimeout
			default:
				//	first check if a idle connection is available
				cp.mu.Lock()
				numberOfIdleConnections := len(cp.idleConnections)
				if numberOfIdleConnections > 0 {
					for _, val := range cp.idleConnections {
						delete(cp.idleConnections, val.id)
						cp.mu.Unlock()
						rq.connectionChan <- val
						hasCompleted = true
						break
					}
				} else if cp.maxOpenConnection > 0 && cp.maxOpenConnection > cp.numOfOpenConnection {
					//check if pool has not exceeded number of allowed open connections
					// increase numberOfConnection hoping connection is created
					cp.numOfOpenConnection++
					cp.mu.Unlock()

					conn, err := cp.openConnection()
					//ignoring error because the only error we care about is the timeout
					cp.mu.Lock()
					cp.numOfOpenConnection--
					cp.mu.Unlock()
					if err == nil {
						rq.connectionChan <- conn
						hasCompleted = true
					}

				} else {
					//unlock pool and restart
					cp.mu.Unlock()
				}
			}
		}
	}

}
