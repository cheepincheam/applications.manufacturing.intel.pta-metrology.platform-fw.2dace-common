//
//                  INTEL CORPORATION PROPRIETARY INFORMATION
//     This software is supplied under the terms of a license agreement or
//     nondisclosure agreement with Intel Corporation and may not be copied
//     or disclosed except in accordance with the terms of that agreement.
//          Copyright(c) 2018-2020 Intel Corporation. All Rights Reserved.
//
//

package foundationcore

import (
	"reflect"
	"fmt"
	
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

var grpcErrorCodesRetryMaps = map[codes.Code]bool{
	codes.DeadlineExceeded:   true,
	codes.ResourceExhausted:  true,
	codes.FailedPrecondition: true,
	codes.Aborted:            true,
	codes.Unavailable:        true,
	codes.DataLoss:           true,
}

// GrpcClient - The smart grpc client generic that will handle auto-reconnection and error handling
type GrpcClient struct {
	conn       *grpc.ClientConn
	client     interface{}
	clientName string
	url        string
	opts       []grpc.DialOption
	log        *Logger
}

// NewGrpcClient - Create an new instance GrpcClient
// 	url - url in the format of <hostname>:<port if needed>
//  initializer = the closure that the caller code perform actual GrpcTypedClient = new GrpcTypedClient(c)
//	opts - dial options to be used for dial up the connection
func NewGrpcClient(url string, initializer func(c *grpc.ClientConn) interface{}, opts ...grpc.DialOption) (*GrpcClient, error) {
	defer func() {
		if err := recover(); err !=nil {
			fmt.Println("Recovering panic in NewGrpcClient. Error:", err)
		}
	}()


	l, err := NewDefaultLogger()
	if err != nil {
		panic(err)
	}

	o := &GrpcClient{
		conn: nil,
		url:  url,
		opts: opts,
		log:  l,
	}

	o.conn, err = grpc.Dial(o.url, o.opts...)
	if err != nil {
		o.conn = nil
		return nil, err
	}
	o.client = initializer(o.conn)
	return o, nil
}

// Client - Return the underlying created typed grpc client instance
func (o *GrpcClient) Client() interface{} {
	// we don't want to use reflection all the time due to performance price involved, as caller gets the client instance we get the client type
	// name for logging purpose
	o.clientName = reflect.TypeOf(o.client).String()
	return o.client
}

// RobustGrpcCall - Smart call handler
//	f - the closure where the caller implement their actuall call handler to the target grpc endpoint
//
func (o *GrpcClient) RobustGrpcCall(f func() (interface{}, error)) (interface{}, error) {
	var resp interface{}
	var err error
	//o.log.Info("%s.RobustCall: Invoking f() ... \n", o.clientName)
	resp, err = f()
	if err != nil {
		o.log.Warn("%s.RobustCall error : %s. Depending on the error, we may retry ...\n", o.clientName, err.Error())
		code := grpc.Code(err)
		_, keyExists := grpcErrorCodesRetryMaps[code]
		if keyExists {
			if code == codes.Unavailable {
				o.log.Info("%s.RobustCall: Reconnecting to %s\n", o.clientName, o.url)

				// Target microservice is supposed to be load-balanced. We should be connecting to a virtual address,
				// hence simple reconnect should connect us to another load-balanced microservice
				conn, cerr := grpc.Dial(o.url, o.opts...)
				if cerr == nil {
					o.conn = conn
					o.log.Info("%s.RobustCall: Reconnected to %s\n", o.clientName, o.url)
					o.log.Info("%s.RobustCall: retrying f() ... \n", o.clientName)
					resp, err = f()
					if err != nil {
						o.log.Error("%s.RobustCall error : Retry failed: %s\n", o.clientName, err.Error())
					}
					return resp, err
				}
				err = cerr
				o.log.Error("%s.RobustCall error : Failed to reconnect to %s\n", o.clientName, o.url)
				return nil, err
			} else {
				return resp, err
			}
		} else {
			return resp, err
		}
	}
	//o.log.Info("%s.RobustCall: f() succeeded.\n", o.clientName)
	return resp, err
}

// Close - Closes the grpc connection.
func (o *GrpcClient) Close() error {
	return o.conn.Close()
}
