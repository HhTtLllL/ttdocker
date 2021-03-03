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

	pid, err := GetContainerPidByName(containerName)
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

	//获取对应的PID环境变量， 其实也就是容器的环境变量
	containerEnvs := getEnvsByPid(pid)
	//将宿主机的环境变量和容器的环境变量都放置到 exec 进程内
	cmd.Env = append(os.Environ(), containerEnvs...)

	if err := cmd.Run(); err != nil {

		log.Errorf("exec container %s error %v", containerName, err)
	}
}

//根据提供的容器名获取对应容器的ID
func GetContainerPidByName(containerName string) (string, error ){

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
	fmt.Println("Pid = ", containerInfo.Pid)

	return containerInfo.Pid, nil
}

func getEnvsByPid(pid string)[]string {

	//进程环境变量存放的位置是 /proc/PID/environ
	path := fmt.Sprintf("/proc/%s/environ", pid)
	contengBytes, err := ioutil.ReadFile(path)
	if err != nil {

		log.Errorf("Read file %s error %v", path, err)
		return nil
	}
	//多个环境变量中的分隔符\u0000
	envs := strings.Split(string(contengBytes), "\u0000")

	return envs
}
