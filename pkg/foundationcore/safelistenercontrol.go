//
//                  INTEL CORPORATION PROPRIETARY INFORMATION
//     This software is supplied under the terms of a license agreement or
//     nondisclosure agreement with Intel Corporation and may not be copied
//     or disclosed except in accordance with the terms of that agreement.
//          Copyright(c) 2009-2021 Intel Corporation. All Rights Reserved.
//
//

package foundationcore

import (
	"fmt"
	"net"
	"strings"
	"syscall"
)

var registeredGrpcWebPorts []string = []string{}

const maxNumOfSvc = 100
const grpcWebPortField = "rpcwebhostUrl"

// This is the CWE-918 Prevention Implementation based on https://www.agwa.name/blog/post/preventing_server_side_request_forgery_in_golang
// Our filter logic are slightly enhanced to only allow port number configured in our system configuration files and not port 80.
// We do not filter address type for now
//
func GrpcWebListenerControl(network string, address string, conn syscall.RawConn) error {
	if !(network == "tcp4" || network == "tcp6") {
		return fmt.Errorf("%s is not a safe network type", network)
	}

	_, port, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("%s is not a valid host/port pair: %s", address, err)
	}
	
	// our grpcweb will never be on port 80
	if port == "80" || !containsPort(port) {
		return fmt.Errorf("%s is not a safe port number", port)
	}
	return nil
}

func containsPort(port string) bool {
	ans := false
	if len(registeredGrpcWebPorts) > maxNumOfSvc {
		panic("Exceeded maximum allowed number of microservice")
	}
	for _, p := range registeredGrpcWebPorts {
		if p == port {
			ans = true
			break
		}
	}
	return ans
}

func init() {
	microsvcConfigs := GetSystemConfig().GetConfig(ConfigType_SVC).(map[string]interface{})
	if len(microsvcConfigs) > maxNumOfSvc {
		panic("Exceeded maximum allowed number of microservice")
	}
	for _, svcConf := range microsvcConfigs {
		o := svcConf.(map[string]interface{})
		grpcWebUrl, exists := o[grpcWebPortField]
		if exists {
			tokens := strings.Split(grpcWebUrl.(string), ":")
			if len(tokens) == 2 {
				registeredGrpcWebPorts = append(registeredGrpcWebPorts, tokens[1])
			}
		}
	}
}
