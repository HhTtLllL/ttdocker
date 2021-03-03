package main

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"io/ioutil"
	"os"
	"ttdocker/container"
)

func logContainer(containerName string ){

	//找到对应的文件夹的位置
	dirURL := fmt.Sprintf(container.DefaultInfoLocation, containerName)
	logFileLocation := dirURL + container.ContainerLogFile

	//打开日志文件
	file, err := os.Open(logFileLocation)
	defer file.Close()
	if err != nil {

		log.Errorf("log container open file %s error %v", logFileLocation, err)
		return
	}

	//将文件内的内容都读取出来
	content, err := ioutil.ReadAll(file)
	if err != nil {

		log.Errorf("log container read file %s error %v", logFileLocation, err)
		return
	}

	//使用fmt.fprint 函数将读出啦的内容输入到标准输出，也就是控制台上
	fmt.Fprint(os.Stdout, string(content))
}