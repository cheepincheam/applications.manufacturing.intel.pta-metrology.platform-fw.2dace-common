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
	"fmt"
	"plugin"
	"runtime"	
)

var loaderConfigLoc string = ""
var sharedLibLoc string = ""

// DynamicTypeLoader is essentially a wrapper over Go plugin package that is
// designed to aid dependency injection mechanism
//
// Dependency Injection Mechanism
//
// 1. Each library that aspired to be injected, needs to implement a constructor
// function named NewXXX that bear the signature as below:
//	func NewXXX(<non or variable argurments>) *xxxInteface
//
// 2. Currently only interface bound object are supported for DI, hence an inteface
// is required for DI to work. i.e for database driver, A factory type and a interface
// specification is needed so that each db driver can be implemented according to
// the interface specification
//
// 3. The NewObject constructor function is responsible to create and return an instance
// of the abstract interface instance that represent the interface specification

// DependencyInjecterFactory is the factory that create DI injecter.
type DependencyInjecterFactory interface {
	Create() interface{}
}

// DynamicTypeLoader - wrapper over Go plugin package that is designed to aid dependency injection mechanism
type DynamicTypeLoader struct {
	instance *plugin.Plugin
	vtable   map[string]plugin.Symbol
}

// load - method that load a specific shared library
func (o *DynamicTypeLoader) load(sharedLibFileName string) {
	obj, lerr := plugin.Open(sharedLibLoc + sharedLibFileName)
	if lerr != nil {
		panic(lerr)
	}
	o.instance = obj
	o.vtable = make(map[string]plugin.Symbol)
	fmt.Println("NewTypeLoader load completed.")
}

// Lookup - lookup the function based on its name.
func (o *DynamicTypeLoader) Lookup(functionName string) *plugin.Symbol {
	var err error
	var s plugin.Symbol
	var lookupNeeded bool = false

	fmt.Println("NewTypeLoader Lookup started.", len(o.vtable))

	if len(o.vtable) > 1 {
		var ok bool
		s, ok = o.vtable[functionName]
		if ok == false {
			lookupNeeded = true
		}
	} else {
		lookupNeeded = true
	}
	if lookupNeeded == true {
		s, err = o.instance.Lookup(functionName)
		if err != nil {
			panic(err)
		}
		o.vtable[functionName] = s
		fmt.Println("Object inserted")
	}
	return &s
}

// NewTypeLoader - create an instance of DynamicTypeLoader.
func NewTypeLoader(sharedLibFileName string) *DynamicTypeLoader {
	defer func() {
		if err := recover(); err !=nil {
			fmt.Println("Recovering panic in NewTypeLoader. Error:", err)
		}
	}()

	p := new(DynamicTypeLoader)
	p.load(sharedLibFileName)
	fmt.Println("NewTypeLoader completed.")
	return p
}

func init() {
	// Since linux OS is our defacto environment, we only use config file to designate
	// the location of shared library location. In windows we hardcore it for now
	//
	if runtime.GOOS == "linux" {
		readObjects := GetSystemConfig().GetConfig(ConfigType_SO).(map[string]interface{})

		// parse conf json content and read in the path
		sharedLibLoc = readObjects["path"].(string)
	} else {
		sharedLibLoc = ""
	}
}
