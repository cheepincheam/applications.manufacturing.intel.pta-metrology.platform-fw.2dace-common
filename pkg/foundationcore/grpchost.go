//
//                  INTEL CORPORATION PROPRIETARY INFORMATION
//     This software is supplied under the terms of a license agreement or
//     nondisclosure agreement with Intel Corporation and may not be copied
//     or disclosed except in accordance with the terms of that agreement.
//          Copyright(c) 2009-2019 Intel Corporation. All Rights Reserved.
//
//

package foundationcore

import (
	"2dacecommon/pkg/foundationcore/grpcweb"
	"context"
	"net"
	"net/http"

	"google.golang.org/grpc"
)

const GigabyteMsgSize = 1073741824

// IGrpcSvc - interface that defines an grpc service for our cloud grpc services as the foundationcore automatically handle the hosting aspects.
//	Due to the way grpc generate registration code, we need to use auto generated registration API code. Hence, the Register interface method is for
//	custom grpc svc to call that registration API for us.
type IGrpcSvc interface {
	Register(*grpc.Server)
}

// GrpcHost - An instance grpc service.
type GrpcHost struct {
	grpcSvr  *grpc.Server // GRPC server instace
	svcName  string       // service name
	hostUrl  string       // default grpc hosting url
	log      *Logger      // logger instance
	listener net.Listener // listener for the gprc transport
	// Grpc Web variables
	enabledGrpcWeb    bool                       // do we enable grpcweb protocol
	grpcWebHostingUrl string                     // GRPC web hosting url
	grpcWebSvr        *grpcweb.WrappedGrpcServer // grpc web server instance
	httpSvr           *http.Server               // http server instance
	httpListener      net.Listener               // http transport listener
}

// NewGrpcHost - return an instance of the grpc host.
func NewGrpcHost(svcName string, hostingUrl string, enabledGrpcWeb bool, grpcWebHostingUrl string) *GrpcHost {
	var err error
	var log *Logger
	var l net.Listener

	log, err = NewDefaultLogger()
	if err != nil {
		log.Error("Failed to create logger\n")
	}
	l, err = net.Listen("tcp", hostingUrl)
	if err != nil {
		log.Error("Failed to listen on %v\n", err)
	}

	options := grpc.MaxRecvMsgSize(GigabyteMsgSize)
	grpcSvr := grpc.NewServer(options)
	obj := &GrpcHost{
		grpcSvr:           grpcSvr,
		svcName:           svcName,
		hostUrl:           hostingUrl,
		log:               log,
		listener:          l,
		enabledGrpcWeb:    enabledGrpcWeb,
		grpcWebHostingUrl: grpcWebHostingUrl}
	return obj
}

// Start - start hosting the grpc service. This method takes an instance IGrpcSvc interface implementation instance.
//	It will register the server and start serving grpc. If grpcweb is enabled, it will serve that too
func (o *GrpcHost) Start(svc IGrpcSvc) {
	var err error
	svc.Register(o.grpcSvr)
	go o.grpcSvr.Serve(o.listener)
	o.log.Info("GRPC Server for %s hosted on %s\n", o.svcName, o.hostUrl)
	if o.enabledGrpcWeb {
		o.grpcWebSvr = grpcweb.WrapServer(o.grpcSvr,
			grpcweb.WithOriginFunc(func(s string) bool {
				return true
			}),
			grpcweb.WithCorsForRegisteredEndpointsOnly(false),
			grpcweb.WithAllowNonRootResource(true),
			grpcweb.WithAllowedRequestHeaders([]string{"*"}),
		)
		o.httpSvr = &http.Server{
			Handler: http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
				o.grpcWebSvr.ServeHTTP(resp, req)
			}),
		}
		/*
			o.httpListener, err = net.Listen("tcp", o.grpcWebHostingUrl)
		*/
		// Above is the original HTTP listener. In effort to implement prevention against CWE-918 Server-Side Request Forgery (SSRF),
		// we utilize ListenerConfig with control routine to mitigate that. More details is at safelistenercontrol.go
		listenerCfg := &net.ListenConfig{
			Control: GrpcWebListenerControl,
		}
		o.httpListener, err = listenerCfg.Listen(context.Background(), "tcp", o.grpcWebHostingUrl)
		if err == nil {
			go func() {
				o.httpSvr.Serve(o.httpListener)
			}()
			o.log.Info("GRPC-Web Server for %s hosted on %s\n", o.svcName, o.grpcWebHostingUrl)
		} else {
			panic(err)
		}
	}
}

// Stop - stop the serving of grpc service. If grpcweb is started earlier, it will be torn down as well
func (o *GrpcHost) Stop() {
	o.grpcSvr.GracefulStop()
	if o.enabledGrpcWeb {
		o.httpSvr.Shutdown(context.Background())
		o.httpListener.Close()
	}
	o.log.Info("%s GRPC & GRPC-WEB (if enabled) has been torn down successfully\n", o.svcName)
}
