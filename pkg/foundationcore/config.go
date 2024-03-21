//
//                  INTEL CORPORATION PROPRIETARY INFORMATION
//     This software is supplied under the terms of a license agreement or
//     nondisclosure agreement with Intel Corporation and may not be copied
//     or disclosed except in accordance with the terms of that agreement.
//          Copyright(c) 2009-2019 Intel Corporation. All Rights Reserved.
//

package foundationcore

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"runtime"	
)

const defaultLinuxConfPath = "/etc/intelatcloud/conf/sys_conf.json"
//const defaultWinConfPath = "C:\\IAFW\\ATTDPFIASDK\\atcloud\\conf\\sys_conf.json"
const defaultWinConfPath = "C:\\Program Files\\Intel\\2dace\\conf\\sys_conf.json"
const envConfVarName = "ATCLOUD_CONF"
const envConsoleName = "LOG_TO_CONSOLE"
const sysFileName = "sys_conf.json"

// ConfigType - string type that specify a type pf configuration. i.e db or pubsub.
type ConfigType string

// Configuration type string definitions
const (
	ConfigType_DB           		ConfigType = "db"
	ConfigType_PUBSUB       		ConfigType = "pubsub"
	ConfigType_SO           		ConfigType = "so"
	ConfigType_SVC          		ConfigType = "microservices"
	ConfigType_ORCHESTRATOR 		ConfigType = "orchestratorclient"
	ConfigType_REPOSITORY   		ConfigType = "repository"
	ConfigType_CLUSTER      		ConfigType = "cluster"
	ConfigType_LDAP         		ConfigType = "ldap"
	ConfigType_CONTAINERREGISTRY 	ConfigType = "containerregistry"
)

// our singleton instance of ConfigMgr
var configMgr *ConfigMgr

// our singleton instance of config path
var configPath string

// ConfigMgr - stores the map of configuration details from sys_conf.json
type ConfigMgr struct {
	configMap map[ConfigType]interface{}
}

// GetConfig - return the configuration object by configuration type.
func (o *ConfigMgr) GetConfig(confType ConfigType) interface{} {
	conf, ok := o.configMap[confType]
	if ok {
		return conf
	} else {
		return nil
	}
}

func (o *ConfigMgr) Write() error {
	writeBytes, err := json.Marshal(o.configMap)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(configPath, writeBytes, 0644)
	if err != nil {
		fmt.Println("ConfigMgr.Write error :", err)
	}
	return err
}

// IsConfigSectionExists - check if a specific configuration type exists.
func (o *ConfigMgr) IsConfigSectionExists(confType ConfigType) bool {
	_, exists := o.configMap[confType]
	return exists
}

// GetSystemConfig - Get hold of the instance of ConfigMgr
func GetSystemConfig() *ConfigMgr {
	return configMgr
}

// validateFilePath - check if the file path specified is accessable.
func validateFilePath(filePath string) error {
	var err error
	if _, ferr := os.Stat(filePath); ferr != nil {
		if os.IsNotExist(ferr) {
			err = fmt.Errorf("%s does not exists", filePath)
		} else if os.IsPermission(ferr) {
			err = fmt.Errorf("%s access is denied", filePath)
		} else {
			err = ferr
		}
	}
	return err
}

func init() {
	defer func() {
		if err := recover(); err !=nil {
			fmt.Println("Recovering panic in reading sys_conf.json. Error:", err)
		}
	}()
	if confEnv := os.Getenv(envConfVarName); len(confEnv) > 0 {
		if validateFilePath(confEnv) == nil {
			configPath = confEnv + "\\" + sysFileName
		}
	}
	if configPath == "" {
		if runtime.GOOS == "linux" {
			if err := validateFilePath(defaultLinuxConfPath); err != nil {
				panic(err)
			} else {
				configPath = defaultLinuxConfPath
			}
		} else {
			configPath = defaultWinConfPath
			if err := validateFilePath(defaultWinConfPath); err != nil {
				panic(err)
			} else {
				configPath = defaultWinConfPath
			}
		}
	}

	// read the sys_conf.json
	//fmt.Println("Config is at ", configPath)
	conf, cerr := os.Open(configPath)
	if cerr != nil {
		panic(cerr)
	}
	defer conf.Close()
	var readBytes []byte
	readBytes, cerr = ioutil.ReadAll(conf)
	if cerr != nil {
		panic(cerr)
	}
	// parse log json content and read in the path
	var readObjects map[string]interface{}
	cerr = json.Unmarshal([]byte(readBytes), &readObjects)
	if cerr != nil {
		panic(cerr)
	}
	configMgr = &ConfigMgr{}
	configMgr.configMap = make(map[ConfigType]interface{})
	for k, v := range readObjects {
		configMgr.configMap[ConfigType(k)] = v
	}
}
