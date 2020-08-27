package container

import (
	log "github.com/Sirupsen/logrus"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

func NewParentProcess(tty bool, volume string) (*exec.Cmd, *os.File) {

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
	NewWorkSpace(rootURL, mntURL, volume)

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
func NewWorkSpace(rootURL string, mntURL string, volume string){

	CreateReadOnlyLayer(rootURL)  //新建busybox 文件夹，将busybox.tar 解压到 busybox 目录下，作为容器的只读层
	CreateWriteLayer(rootURL)    //创建了一个名为 writeLayer　的文件夹，　作为容器唯一的可写层
	CreateMountPoint(rootURL, mntURL) //创建了mnt 文件，作为挂载点，然后啊writeLayer目录和busybox 目录mount 到 mnt 目录下

	if volume != "" {

		//解析volume 串
		volumeURLs := volumeUrlExtract(volume)
		length := len(volumeURLs)

		if length == 2 && volumeURLs[0] != "" && volumeURLs[1] != "" {

			MountVolume(rootURL, mntURL, volumeURLs)
			log.Infof("%q", volumeURLs)
		}else {

			log.Infof("volume parameter input is not correct.")
		}

	}

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

func MountVolume(rootURL string, mntURL string, volumeURLs []string){

	//读取宿主机文件目录 URL, 创建宿主机文件目录
	parentUrl := volumeURLs[0]
	if err := os.Mkdir(parentUrl, 0777); err != nil {

		log.Infof("Mkdir parent dir %s error. %v", parentUrl, err)
	}

	//读取容器挂载点URL,　在容器文件系统里创建挂载点
	containerUrl := volumeURLs[1]
	containerVolumeURL := mntURL + containerUrl
	if err := os.Mkdir(containerVolumeURL, 0777); err != nil {

		log.Infof("mkdir container dir %s error. %v", containerVolumeURL, err)
	}
	dirs := "dirs=" + parentUrl


	//最后把宿主机文件目录挂载到容器挂载点，　这样启动容器的过程，对数据卷的处理也就完成了
	cmd := exec.Command("mount", "-t", "aufs", "-o", dirs, "none", containerVolumeURL)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {

		log.Errorf("mount volume failed. %v", err)
	}

}


//Delete the AUFS filesystem while container exit
//当　容器退出的时候，　删除aufs　文件系统
func DeleteWorkSpace(rootURL string, mntURL string, volume string){

	if volume != "" {

		volumeURLs := volumeUrlExtract(volume)
		length := len(volumeURLs)

		if length == 2 && volumeURLs[0] != "" && volumeURLs[0] != "" {
			DeleteMountPointWithVolume(rootURL, mntURL, volumeURLs)
		}else {
			DeleteMountPoint(rootURL, mntURL)
		}
	}else {
		DeleteMountPoint(rootURL, mntURL)
	}

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

func DeleteMountPointWithVolume(rootURL string, mntURL string, volumeURLs []string){

	containerUrl := mntURL + volumeURLs[1]

	//卸载volume挂载点的文件系统，　保证整个容器的挂载点没有被使用
	cmd := exec.Command("umount", containerUrl)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {

		log.Errorf("umount volume failed. %v", err)
	}

	//卸载整个容器文件系统的挂载点
	cmd = exec.Command("umount", mntURL)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {

		log.Errorf("umount mountpotin failed. %v", err)
	}

	//删除容器文件系统挂载点。
	if err := os.RemoveAll(mntURL); err != nil {

		log.Infof("Remove mountpoint dir %s error %v", mntURL, err)
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

//解析volume 字符串
func volumeUrlExtract(volume string) ([]string) {

	volumeURLs := strings.Split(volume, ":")

	return volumeURLs
}













































