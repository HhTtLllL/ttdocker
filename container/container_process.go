package container

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"os"
	"os/exec"
	"syscall"
)
//一个容器的基本信息
type ContainerInfo struct {
	Pid 		string `json:"pid"`   //容器的init进程在 宿主机上的　PID
	Id 			string `json:"id"`		// 容器ID
	Name 		string `json:"name"`  // 容器名
	Command 	string `json:"command"` //容器内init 进程的运行命令
	CreatedTime string `json:"createTime"` //创建时间
	Status 		string `json:"status"`    //容器的状态
	Volume 		string `json:"volume"`   //容器的数据卷
	PortMapping []string `json:"portmapping"`  //端口映射
}

// 状态  全局变量
var (
	RUNNING 			string = "running"
	STOP 				string = "stopped"
	Exit 				string = "exited"
	DefaultInfoLocation string = "/var/run/ttdocker/%s/"
	ConfigName  		string = "config.json"
	ContainerLogFile 	string = "container.log"
	RootUrl 			string = "/root"
	MntUrl 				string = "/root/mnt/%s"
	WriteLayerUrl 		string = "/root/writeLayer/%s"
)

func NewParentProcess(tty bool, volume string, containerName string, imageName string, envSlice []string) (*exec.Cmd, *os.File) {

	readPipe, writePipe, err := NewPipe()
	if err != nil {

		log.Errorf("new pipe error %v",err)
		return nil, nil
	}
	cmd := exec.Command("/proc/self/exe", "init")
	//fork 出一个新进程
	//在cmd.Run 的时候，会调用系统调用的 clone()。
	cmd.SysProcAttr = &syscall.SysProcAttr{

		//Cloneflags 这个API只有linux 上才有
		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS |
			syscall.CLONE_NEWNET | syscall.CLONE_NEWIPC | syscall.CLONE_NEWUSER,

			UidMappings: []syscall.SysProcIDMap{{ContainerID: 0, HostID: syscall.Getuid(), Size: 1,},},
			GidMappings: []syscall.SysProcIDMap{{ContainerID: 0,HostID: syscall.Getuid(),Size: 1,},},
	}

	if tty {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}else {

		//生成容器对应目录的container. log
		dirURL := fmt.Sprintf(DefaultInfoLocation, containerName)
		if err := os.MkdirAll(dirURL, 0622); err != nil {

			log.Errorf("NewParentProcess mkdir %s error %v", dirURL, err)
			return nil, nil
		}

		stdLogFilePath := dirURL + ContainerLogFile
		stdLogFile, err := os.Create(stdLogFilePath)
		if err != nil {

			log.Errorf("newParentProcess create file %s error %v", stdLogFilePath, err)
			return nil, nil
		}

		cmd.Stdout = stdLogFile
	}

	//这个属性的意思是会外带着这个文件句柄去创建子进程
	cmd.ExtraFiles = []*os.File{readPipe}
	cmd.Env = append(os.Environ(), envSlice...)

	//切换到　/root/busybox　目录
	NewWorkSpace(volume,imageName, containerName)

	cmd.Dir = fmt.Sprintf(MntUrl, containerName)

	return cmd, writePipe
}


func NewPipe() (*os.File, *os.File, error){

	/*
		func Pipe() (r *File, w *File, err error)
		Pipe返回一对关联的文件对象。从r的读取将返回写入w的数据。本函数会返回两个文件对象和可能的错误。
	*/
	read, write, err := os.Pipe()
	if err!= nil {

		return nil, nil, err
	}

	return read,write,nil
}
