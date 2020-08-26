package container

import (
	log "github.com/Sirupsen/logrus"
	"os"
	"os/exec"
	"syscall"
)

func NewParentProcess(tty bool) (*exec.Cmd, *os.File) {

	readPipe, writePipe, err := NewPipe()

	if err != nil {
		log.Errorf("new pipe error %v",err)

		return nil, nil
	}



//	args := []string{"init", command}

	cmd := exec.Command("/proc/self/exe", "init")

	cmd.SysProcAttr = &syscall.SysProcAttr{

		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS |
			syscall.CLONE_NEWNET | syscall.CLONE_NEWIPC | syscall.CLONE_NEWUSER,

			UidMappings: []syscall.SysProcIDMap{{ContainerID: 0, HostID: syscall.Getuid(), Size: 1,},},
			GidMappings: []syscall.SysProcIDMap{{ContainerID: 0,HostID: syscall.Getuid(),Size: 1,},},
	}

	if tty {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	//这个属性的意思是会外带着这个文件句柄去创建子进程
	cmd.ExtraFiles = []*os.File{readPipe}

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