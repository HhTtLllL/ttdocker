package main

import (
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"io/ioutil"
	_ "ttdocker/nsenter"
	"os"
	"os/exec"
	"strings"
	"ttdocker/container"
)

const ENV_EXEC_PID = "ttdocker_pid"
const ENV_EXEC_CMD = "ttdocker_cmd"

func ExecContainer(containerName string, comArray []string){

	pid, err := getContainerPidByName(containerName)

	if err != nil {
		log.Errorf("exec container getcontainerPidByName %s error %v", containerName, err)
		return
	}

	cmdStr := strings.Join(comArray, " ")
	log.Infof("container pid %s ", pid)
	log.Infof("command %s", cmdStr)

	cmd := exec.Command("/proc/self/exe", "exec")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	os.Setenv(ENV_EXEC_PID, pid)
	os.Setenv(ENV_EXEC_CMD, cmdStr)
	if err := cmd.Run(); err != nil {
		log.Errorf("exec container %s error %v", containerName, err)
	}
}

//根据提供的容器名获取对应容器的ID
func getContainerPidByName(containerName string) (string, error ){

	//拼接存储容器信息的路径
	dirURL := fmt.Sprintf(container.DefaultInfoLocation, containerName)
	configFilePath := dirURL + container.ConfigName

	//读取对应路径下的文件内容
	contentBytes, err := ioutil.ReadFile(configFilePath)

	if err != nil {
		return "", err
	}

	var containerInfo container.ContainerInfo

	//将文件内容反序列化成容器信息对象,然后返回对应的PID
	if err := json.Unmarshal(contentBytes, &containerInfo); err != nil {
		return "", err
	}

	return containerInfo.Pid, nil
}