/*
//
//                  INTEL CORPORATION PROPRIETARY INFORMATION
//     This software is supplied under the terms of a license agreement or
//     nondisclosure agreement with Intel Corporation and may not be copied
//     or disclosed except in accordance with the terms of that agreement.
//          Copyright(c) 2009-2019 Intel Corporation. All Rights Reserved.
//
*/
package foundationcore

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"runtime"
	"strconv"
	"strings"
	"time"

	"go.elastic.co/ecszap"
	zap "go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.design/x/lockfree"
	"gopkg.in/natefinch/lumberjack.v2"
)

const linuxLogConfigPath string = "/etc/intelatcloud/conf/%s_log_conf.json"
const winLogConfigPath string = "C:\\IAFW\\ATTDPFIASDK\\atcloud\\conf\\log_conf.json"
const logFileName string = "log_conf.json"
const logContextKey string = "requestContext"

type LogContext struct {
	Namespace        string
	Operation        string
	Application      string
	RequestId        string
	CustomProperties map[string]string
}

func (o *LogContext) ToString() string {
	// s := fmt.Sprintf("{\n\t\"application\":\"%s\",\n\t\"operation\":\"%s\",\n\t\"request_id\":\"%s\",\n\t\"namespace\":\"%s\"", o.Application, o.Operation, o.RequestId, o.Namespace)
	s := fmt.Sprintf("{\n\t\"app.name\":\"%s\",\n\t\"2dace.operation\":\"%s\",\n\t\"2dace.txn.id\":\"%s\",\n\t\"kubernetes.namespace\":\"%s\"", o.Application, o.Operation, o.RequestId, o.Namespace)
	for k, v := range o.CustomProperties {
		s += fmt.Sprintf(",\n\t\"%s\":\"%s\"", k, v)
	}
	s += "\n}"
	return s
}

func SyslogTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	t.Local()
	layout := "2006-01-02 15:04:05.000"
	enc.AppendString(t.Format(layout))
}

var configLoc string

// Logger type - wraps around Uber Zap SugaredLogger
type Logger struct {
	log       *zap.Logger
	ctx       *LogContext
	ctxStack  *lockfree.Stack
	cachedCtx string
}

func (o *Logger) Enter(opr string) {
	c := &LogContext{
		Namespace:        o.ctx.Namespace,
		Operation:        opr,
		Application:      o.ctx.Application,
		RequestId:        o.ctx.RequestId,
		CustomProperties: o.ctx.CustomProperties,
	}
	o.ctxStack.Push(o.ctx)
	o.ctx = c
}

func (o *Logger) EnterWithId(reqId string, opr string) {
	c := &LogContext{
		Namespace:        o.ctx.Namespace,
		Operation:        opr,
		Application:      o.ctx.Application,
		RequestId:        reqId,
		CustomProperties: o.ctx.CustomProperties,
	}
	o.ctxStack.Push(o.ctx)
	o.ctx = c
}

func (o *Logger) Exit() {
	o.ctx = o.ctxStack.Pop().(*LogContext)
}

func (o *Logger) Info(template string, args ...interface{}) {
	o.log.Info(fmt.Sprintf(template, args...), zap.String(logContextKey, o.cachedCtx))
}
func (o *Logger) Error(template string, args ...interface{}) {
	o.log.Error(fmt.Sprintf(template, args...), zap.String(logContextKey, o.cachedCtx))
}
func (o *Logger) Warn(template string, args ...interface{}) {
	o.log.Warn(fmt.Sprintf(template, args...), zap.String(logContextKey, o.cachedCtx))
}
func (o *Logger) Debug(template string, args ...interface{}) {
	o.log.Debug(fmt.Sprintf(template, args...), zap.String(logContextKey, o.cachedCtx))
}
func (o *Logger) Panic(template string, args ...interface{}) {
	o.log.Panic(fmt.Sprintf(template, args...), zap.String(logContextKey, o.cachedCtx))
}

// NewLogger - Create an instance of logger
func NewLogger(logConfigFilePath string, log_ctx *LogContext) (*Logger, error) {

	// Read the log config content
	logConfigFile, fileErr := os.Open(logConfigFilePath)
	if fileErr != nil {
		return nil, fileErr
	}
	defer logConfigFile.Close()
	readBytes, readErr := ioutil.ReadAll(logConfigFile)
	if readErr != nil {
		return nil, readErr
	}

	// parse log json content
	var readObjects map[string]interface{}
	jerr := json.Unmarshal([]byte(readBytes), &readObjects)
	if jerr != nil {
		return nil, jerr
	}

	// for now, we just need to know where to log, we will use the standard production setting provided
	// by Uber zap logger. The log config would bear the format of path/*.log, we need replace * with the actual process name
	var logDestination string
	if runtime.GOOS == "linux" {
		logDestination = readObjects["FileChannelSettings"].(map[string]interface{})["Linux_Path"].(string)
	} else if runtime.GOOS == "windows" {
		logDestination = readObjects["FileChannelSettings"].(map[string]interface{})["Win_Path"].(string)
	} else {
		panic(errors.New("Not supported OS"))
	}

	logDestination = strings.Replace(logDestination, "*", path.Base(os.Args[0]), 1)

	if _, err := os.Stat(logDestination); err != nil {
		if os.IsNotExist(err) {
			err = fmt.Errorf("Log Directory Does not exists.")
		}
	}

	levelConfig := readObjects["Level"]
	var logLevel zapcore.Level = zapcore.InfoLevel
	switch levelConfig {
	case "information":
		logLevel = zapcore.InfoLevel
	case "debug":
		logLevel = zapcore.DebugLevel
	case "warn":
		logLevel = zapcore.WarnLevel
	// if no value is set, default is infolevel
	default:
		logLevel = zapcore.InfoLevel
	}
	ecsEncoderConfig := ecszap.NewDefaultEncoderConfig()
	ecsEncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	encoderConfig := ecsEncoderConfig.ToZapCoreEncoderConfig()
	encoderConfig.EncodeTime = SyslogTimeEncoder

	logConsole := os.Getenv(envConsoleName)
	var rotate_bool bool = false
	if logConsole != "1" {
		logrotate_config := readObjects["FileChannelSettings"].(map[string]interface{})["Rotation"].(string)
		logrotate_config = strings.ToLower(logrotate_config)

		if logrotate_config == "true" {
			if runtime.GOOS == "windows" {
				winLogRollingPath := readObjects["FileChannelSettings"].(map[string]interface{})["Win_Rotate_Path"].(string)
				LogRollingPath := winLogRollingPath + os.Args[0] + ".log"
				logDestination = LogRollingPath
			}
			if _, err := os.Stat(logDestination); err != nil {
				if os.IsNotExist(err) {
					err = fmt.Errorf("Log Directory Does not exists.")
				}
			}
			rotate_bool = true
		}
	}

	// then our log wrapper
	var logger *Logger
	logger = new(Logger)
	logger.ctx = log_ctx
	logger.ctxStack = lockfree.NewStack()
	if logger.ctx != nil {
		logger.cachedCtx = logger.ctx.ToString()
	}

	if rotate_bool == true {
		println("rotate_bool = true")
		max_size_config := readObjects["FileChannelSettings"].(map[string]interface{})["Maxsize"].(string)
		max_size, _ := strconv.Atoi(max_size_config)
		maxbackup_config := readObjects["FileChannelSettings"].(map[string]interface{})["MaxBackups"].(string)
		max_back_up, _ := strconv.Atoi(maxbackup_config)
		max_age_config := readObjects["FileChannelSettings"].(map[string]interface{})["MaxAge"].(string)
		max_age, _ := strconv.Atoi(max_age_config)
		compress_config := readObjects["FileChannelSettings"].(map[string]interface{})["Compress"].(bool)
		compress_localtime := readObjects["FileChannelSettings"].(map[string]interface{})["Compress_Local_Time"].(bool)
		w := zapcore.AddSync(&lumberjack.Logger{
			Filename:   logDestination,
			MaxSize:    max_size,           //megabytes
			MaxBackups: max_back_up,        //days
			MaxAge:     max_age,            //days
			Compress:   compress_config,    // true or false
			LocalTime:  compress_localtime, // once compress, the time stamp put on the old log file
		})
		core := ecszap.WrapCore(zapcore.NewCore(zapcore.NewJSONEncoder(encoderConfig), w, logLevel))
		logger.log = zap.New(core)
	} else {
		core := ecszap.WrapCore(zapcore.NewCore(zapcore.NewJSONEncoder(encoderConfig), zapcore.AddSync(os.Stdout), logLevel))
		logger.log = zap.New(core)
	}
	return logger, nil
}

// NewLogger - Create an instance of logger
func NewDefaultLogger() (*Logger, error) {
	return NewLogger(configLoc, &LogContext{
		Namespace:   "",
		Operation:   "",
		Application: "",
	})
}

func NewLoggerWith(log_ctx *LogContext) (*Logger, error) {
	return NewLogger(configLoc, log_ctx)
}

func init() {

	// Determine the OS and figure conf location
	procName := os.Args[0]
	if runtime.GOOS == "linux" {
		configLoc = fmt.Sprintf(linuxLogConfigPath, path.Base(procName))
	} else {
		defer func() {
			if err := recover(); err != nil {
				fmt.Println("Recovering panic in init of NewDefaultLogger. Error:", err)
			}
		}()

		if confEnv := os.Getenv(envConfVarName); len(confEnv) > 0 {
			if err := validateFilePath(confEnv); err == nil {
				configLoc = confEnv + "\\" + logFileName
			} else {
				panic(err)
			}
		}
		if configLoc == "" {
			if err := validateFilePath(winLogConfigPath); err == nil {
				configLoc = winLogConfigPath
			} else {
				panic(err)
			}
		}
	}
}
