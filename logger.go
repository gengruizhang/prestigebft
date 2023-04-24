package main

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"os"
	"path"
	"runtime"
	"strconv"
)

//configure log format
func saveLog() {
	e := os.RemoveAll("./logs/s_" + strconv.Itoa(ThisServerID))
	if e != nil {
		log.Fatal(e)
	}
	fmt.Println(">> old logs removed")
	if err := os.Mkdir("logs", os.ModePerm); err != nil {
		log.Error(err)
	}
	fmt.Println(">> new ./logs created")

	log.Out = os.Stdout
	fileName := fmt.Sprintf("./logs/server_%d_n_%d_b%d_d%d.log", ThisServerID, NumOfServers, BatchSize, Delay)
	file, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		log.Out = file
	} else {
		log.Info("Failed to log to file, using default stderr")
	}
}

func setLogger() {
	if NaiveStorage {
		saveLog()
	}
	log.SetReportCaller(true)
	//log.SetFormatter(&log.JSONFormatter{})
	PopcornLogFormat := &logrus.TextFormatter{
		//CallerPrettyfier: func(f *runtime.Frame) (string, string) {
		//	filename := path.Base(f.File)
		//	return fmt.Sprintf("%s:%d", filename, f.Line), time.Now().Format(time.RFC3339)
		//},

		CallerPrettyfier: func(f *runtime.Frame) (string, string) {
			filename := path.Base(f.File)
			return fmt.Sprintf("%s:%d", filename, f.Line), " "
		},
	}
	log.SetFormatter(PopcornLogFormat)

	log.SetLevel(logrus.Level(LogLevel))
}
