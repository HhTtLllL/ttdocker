package main

import (
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"io/ioutil"
	"os"
	"ttdocker/container"
	"strconv"
	"syscall"
)

func stopContainer(containerName string){

	//根据容器名获取对应的主进程 PID
	pid, err := GetContainerPidByName(containerName)
	if err != nil {

		log.Errorf("get container pid by name %s error %v", containerName, err)
		return
	}

	//将 string 类型的PID转换为 int 类型
	pidInt, err := strconv.Atoi(pid)
	if err != nil {

		log.Errorf("conver pid from string to int error %v", err)
		return
	}

	//调用系统diaoyong kill 可以发送信号给进程, 通过传递syscall.SIGTERM 信号，去杀掉容器主进程
	if err := syscall.Kill(pidInt, syscall.SIGTERM); err != nil {

		log.Errorf("stop container %s error %v", containerName, err)
		return
	}

	//根据容器名获取对应信息对象
	containerInfo, err := getContainerInfoByName(containerName)
	if err != nil{

		log.Errorf("get container %s info error %v",err)
	}

	//至此，容器进程已经被kill， 所以下面需要修改容器状态，PID可以置为空
	containerInfo.Status = container.STOP
	containerInfo.Pid = " "

	//将修改后的信息序列化成 json 的字符串
	newContentBytes, err := json.Marshal(containerInfo)
	if err != nil {

		log.Errorf("Json marshal %s error %v", containerName, err)
		return
	}

	dirURL := fmt.Sprintf(container.DefaultInfoLocation, containerName)
	configFilePath := dirURL + container.ConfigName
	//重新写入新的数据 覆盖原来的信息
	if err := ioutil.WriteFile(configFilePath, newContentBytes, 0622); err != nil {

		log.Errorf("write file %s error", configFilePath, err)
	}

}

//调用方式 mydocker stop 容器名
//根据容器名获取对应的 struct 结构
func getContainerInfoByName(containerName string) (* container.ContainerInfo, error){

	//构造存放容器信息的路径
	dirURL := fmt.Sprintf(container.DefaultInfoLocation, containerName)
	configFilePath := dirURL + container.ConfigName
	contentBytes, err := ioutil.ReadFile(configFilePath)
	if err != nil {

		log.Errorf("Read file % error %v", configFilePath, err)
		return nil, err
	}

	var containerInfo container.ContainerInfo
	//将容器信息字符串反序列化成对应的对象
	if err := json.Unmarshal(contentBytes, &containerInfo); err != nil {

		log.Errorf("getContainerInfoByName unmarshal error %v", err)
		return &containerInfo, err
	}

	return &containerInfo, nil
}


func removeContainer(containerName string){

	//根据荣启明获取容器对应的信息
	containerInfo, err := getContainerInfoByName(containerName)
	if err != nil {

		log.Errorf("get container %s info error %v", containerName, err)
		return
	}

	//只删除处于停止状态的容器
	if containerInfo.Status != container.STOP{

		log.Errorf("couldn't remove running container")
		return
	}

	//找到对应存储容器信息的文件路径
	dirURL := fmt.Sprintf(container.DefaultInfoLocation, containerName)
	//将所有信息包括子目录都一出
	if err := os.RemoveAll(dirURL); err != nil {

		log.Errorf("remove file %s error %v", dirURL, err)
		return
	}

	//删除工作环境
	container.DeleteWorkSpace(containerInfo.Volume, containerName)
}

