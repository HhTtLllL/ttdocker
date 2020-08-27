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

	//切换到　/root/busybox　目录
	//cmd.Dir = "/root/busybox"
	mntURL := "/root/mnt"
	rootURL := "/root/"
	NewWorkSpace(rootURL, mntURL)

	cmd.Dir = mntURL
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

//Create a AUFS filesystem as container root workspace
//创建一个aufs 文件系统作为容器的根　的工作目录
func NewWorkSpace(rootURL string, mntURL string){

	CreateReadOnlyLayer(rootURL)
	CreateWriteLayer(rootURL)
	CreateMountPoint(rootURL, mntURL)


}

func CreateReadOnlyLayer(rootURL string) {
	busyboxURL := rootURL + "busybox/"
	busyboxTarURL := rootURL + "busyox.tar"

	//判断这个 busybox 目录是否存在
	exist, err := PathExists(busyboxURL)

	if err != nil {
		log.Infof("Fail to judge whether dir %s exists %v", busyboxURL, err)
	}

	if exist == false {
		if err := os.Mkdir(busyboxURL, 0777); err != nil {
			log.Errorf("mkdir dis %s error %v", busyboxURL, err)
		}

		if _, err := exec.Command("tar", "-xvf", busyboxTarURL, "-c", busyboxURL).CombinedOutput(); err != nil {
			log.Errorf("Untar dis %s error %v", busyboxURL, err)
		}
	}
}


func CreateWriteLayer(rootURL string){

	writeURL := rootURL + "writeLayer/"

	if err := os.Mkdir(writeURL, 0777); err != nil {

		log.Errorf("mkdir dis %s error %v", writeURL, err)
	}
}

func CreateMountPoint(rootURL string, mntURL string){

	if err := os.Mkdir(mntURL, 0777); err != nil {

		log.Errorf("mkdir dir %s is error %v", mntURL, err)
	}

	dirs := "dirs=" + rootURL + "writeLayer:" + rootURL + "busybox"

	cmd := exec.Command("mount", "-t", "aufs", "-o", dirs, "none", mntURL)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr;

	//执行 command 命令
	if err := cmd.Run(); err != nil {
		log.Errorf("%v", err)
	}
}

//Delete the AUFS filesystem while container exit
//当　容器退出的时候，　删除aufs　文件系统
func DeleteWorkSpace(rootURL string, mntURL string){
	DeleteMountPoint(rootURL, mntURL)
	DeleteWriteLayer(rootURL)
}

//删除挂载点
func DeleteMountPoint(rootURL string, mntURL string){

	cmd := exec.Command("umount", mntURL)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		log.Errorf("%v", err)
	}

	if err := os.RemoveAll(mntURL); err != nil {

		log.Errorf("Remove dir %s error %v", mntURL, err)
	}
}

func DeleteWriteLayer(rootURL string){

	writeURL := rootURL + "writeLayer/"

	if err := os.RemoveAll(writeURL); err != nil {

		log.Errorf("remove dir %s error %v", writeURL, err)
	}
}


func PathExists(path string ) (bool, error ){

	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}

	return false, err
}














































