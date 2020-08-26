package main

import (
	log "github.com/Sirupsen/logrus"
	"ttdocker/cgroups"
	"ttdocker/cgroups/subsystems"
	"os"
	"strings"
	"ttdocker/container"
)

func Run(tty bool, comArray []string, res *subsystems.ResourceConfig){

	parent, writePipe := container.NewParentProcess(tty)

	//parent := container.NewParentProcess(tty, command)
	//start 调用前面创建好的command 命令
	if parent == nil {

		log.Errorf("new parent process error")
		return
	}

	if err := parent.Start(); err != nil {

		log.Error(err)
	}

	// use mydocker-cgroup as cgroup name
	//创建 cgroup manager ，并通过调用 set 和 apply 设置资源限制并使限制在容器上生效
	cgroupsManager := cgroups.NewCgroupManager("ttdocker-cgroup")
	defer cgroupsManager.Destroy()

	//设置资源限制
	cgroupsManager.Set(res)
	//将容器进程加入到各个 subsystem 挂载对应的 cgroup
	cgroupsManager.Apply(parent.Process.Pid)



	//对容器设置完限制之后，初始化容器
	//发送用户命令
	sendInitCommand(comArray, writePipe)
	parent.Wait()
	//os.Exit(-1)

}

func sendInitCommand(comArray []string, writePipe *os.File){

	command := strings.Join(comArray, " ")
	log.Infof("command all is %s", command)

	writePipe.WriteString(command)
	writePipe.Close()
}
