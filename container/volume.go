package container

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"os"
	"os/exec"
	"strings"
)

//Create a AUFS filesystem as container root workspace
//创建一个aufs 文件系统作为容器的根　的工作目录
//为每个容器创建文件系统
func NewWorkSpace(volume string, imageName string, containerName string){

	//根据用户输入的镜像为每个容器创建只读层
	CreateReadOnlyLayer(imageName)  //新建busybox 文件夹，将busybox.tar 解压到 busybox 目录下，作为容器的只读层
	//为每个容器创建出一个可写层
	CreateWriteLayer(containerName)    //创建了一个名为 writeLayer　的文件夹，　作为容器唯一的可写层
	//创建容器的根目录，然后把镜像层的只读层和容器读写层挂载到容器根目录，成为容器的文件系统
	CreateMountPoint(containerName, imageName) //创建了mnt 文件，作为挂载点，然后啊writeLayer目录和busybox 目录mount 到 mnt 目录下

	if volume != "" {

		//解析volume 串
		volumeURLs := volumeUrlExtract(volume)
		length := len(volumeURLs)

		if length == 2 && volumeURLs[0] != "" && volumeURLs[1] != "" {

			//根据用户输入的volume参数获取相应要挂载的宿主机数据卷URL和容器中的挂载点URL，并挂载数据卷。
			MountVolume(volumeURLs, containerName)
			log.Infof("newworkspace %q", volumeURLs)
		}else {

			log.Infof("volume parameter input is not correct.")
		}
	}
}

// 根据用户输入的镜像为每个容器创建只读层。 镜像解压出来的只读层以RootUrl + imageName 命名
//根据tar 格式的镜像文件作为只读层
func CreateReadOnlyLayer(imageName string)  error {

	unTarFolderUrl := RootUrl + "/" + imageName + "/"
	imageUrl := RootUrl + "/" + imageName + ".tar"

	//判断这个 busybox 目录是否存在
	exist, err := PathExists(unTarFolderUrl)
	if err != nil {

		log.Infof("Fail to judge whether dir %s exists %v", unTarFolderUrl, err)
		return err
	}

	if !exist {
		//如果目录不错在就创建这个目录
		if err := os.MkdirAll(unTarFolderUrl, 0622); err != nil {

			log.Errorf("mkdir dis %s error %v", unTarFolderUrl, err)
			return err
		}

		//解压
		if _, err := exec.Command("tar", "-xvf", imageUrl, "-C", unTarFolderUrl).CombinedOutput(); err != nil {

			log.Errorf("Untar dis %s error %v", unTarFolderUrl, err)
			return err
		}
	}

	return nil
}

//为每一个容器创建一个读写层， 容器的读写层修改成以 WriteLayerUrl + containerName 命名
func CreateWriteLayer(containerName string){

	writeURL := fmt.Sprintf(WriteLayerUrl, containerName)
	if err := os.MkdirAll(writeURL, 0777); err != nil {

		log.Errorf("mkdir dis %s error %v2222", writeURL, err)
	}
}

//创建容器的跟目录,然后把镜像只读层 和 容器读写层挂载到容器根目录，成为容器的文件系统
//把通过镜像解压出来的只读层和容器的可读写层用AUFS联合挂载成为容器的文件系统
func CreateMountPoint(containerName string, imageName string) error {

	mntUrl := fmt.Sprintf(MntUrl, containerName)
	if err := os.MkdirAll(mntUrl, 0777); err != nil {

		log.Errorf("mkdir dir %s is error %v", mntUrl, err)
	}

	tmpWriteLayer := fmt.Sprintf(WriteLayerUrl, containerName)
	tmpImageLocation := RootUrl + "/" + imageName
	mntURL := fmt.Sprintf(MntUrl, containerName)
	dirs := "dirs=" + tmpWriteLayer + ":" + tmpImageLocation

	//把通过镜像解压出来的只读层和容器的可读写层用aufs联合挂载称为容器的文件系统。
	_, err := exec.Command("mount", "-t", "aufs", "-o", dirs, "none", mntURL).CombinedOutput()
	if err != nil {

		log.Errorf("mount volume failed. %v", err)
		return err
	}

	return nil
}

//根据用户输入的volume 参数获取相应要挂载的宿主机 数据卷URL 和容器中的挂载点URL， 并挂载数据卷
//容器的挂载点改为 MntUrl + containerName + containerUrl 命令
func MountVolume(volumeURLs []string, containerName string) error {

	//读取宿主机文件目录 URL, 创建宿主机文件目录
	parentUrl := volumeURLs[0]
	if err := os.Mkdir(parentUrl, 0777); err != nil {

		log.Infof("Mkdir parent dir %s error. %v", parentUrl, err)
	}

	//读取容器挂载点URL,　在容器文件系统里创建挂载点
	containerUrl := volumeURLs[1]
	//containerVolumeURL := mntURL + containerUrl
	mntURL := fmt.Sprintf(MntUrl, containerName)
	containerVolumeURL := mntURL + "/" + containerUrl
	if err := os.Mkdir(containerVolumeURL, 0777); err != nil {

		log.Infof("mkdir container dir %s error. %v", containerVolumeURL, err)
	}
	dirs := "dirs=" + parentUrl

	//最后把宿主机文件目录挂载到容器挂载点，　这样启动容器的过程，对数据卷的处理也就完成了
	_, err := exec.Command("mount", "-t", "aufs", "-o", dirs, "none", containerVolumeURL).CombinedOutput()
	if err != nil {

		log.Errorf("mount volume failed. %v", err)
		return err
	}

	return nil
}


//Delete the AUFS filesystem while container exit
//当　容器退出的时候，　删除aufs　文件系统
func DeleteWorkSpace(volume string, containerName string){

	if volume != "" {

		volumeURLs := strings.Split(volume, ":")
		length := len(volumeURLs)

		if length == 2 && volumeURLs[0] != "" && volumeURLs[1] != "" {

			DeleteMountPointWithVolume(volumeURLs, containerName)
		}else {

			DeleteMountPoint(containerName)
		}
	}else {

		DeleteMountPoint(containerName)
	}

	DeleteWriteLayer(containerName)
}

//删除挂载点
func DeleteMountPoint(containerName string) error {

	mntURL := fmt.Sprintf(MntUrl, containerName)
	_, err := exec.Command("umount", mntURL).CombinedOutput()
	if err != nil {

		log.Errorf("unmount %s error %v", mntURL, err)
		return err
	}

	if err := os.RemoveAll(mntURL); err != nil {

		log.Errorf("Remove dir %s error %v", mntURL, err)
	}

	return nil
}

func DeleteWriteLayer(containerName string){

	writeURL := fmt.Sprintf(WriteLayerUrl, containerName)
	if err := os.RemoveAll(writeURL); err != nil {

		log.Errorf("remove dir %s error %v", writeURL, err)
	}
}

func DeleteMountPointWithVolume(volumeURLs []string, containerName string) error {

	mntURL := fmt.Sprintf(MntUrl, containerName)
	containerUrl := mntURL + "/" + volumeURLs[1]
	//卸载volume挂载点的文件系统，　保证整个容器的挂载点没有被使用
	if _, err := exec.Command("umount", containerUrl).CombinedOutput(); err != nil {

		log.Errorf("umount volume %s failed. %v", containerUrl, err)
		return err
	}

	//卸载整个容器文件系统的挂载点
	if _, err := exec.Command("umount", mntURL).CombinedOutput(); err != nil {

		log.Errorf("umount mountPoint %s failed. %v", mntURL, err)
		return err
	}

	//删除容器文件系统挂载点。
	if  err := os.RemoveAll(mntURL); err != nil {

		log.Infof("Remove mountpoint dir %s error %v", mntURL, err)
	}

	return nil
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

	return strings.Split(volume, ":")
}