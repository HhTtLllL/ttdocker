package main

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"os/exec"
	"ttdocker/container"
)

func commitContainer(containerName, imageName string){

	mntURL := fmt.Sprintf(container.MntUrl, containerName)
	mntURL += "/"
	imageTar := container.RootUrl + "/" +  imageName + ".tar"

	// tar -czf /root/imnaeName.tar -C mntURL .
	//将 mntURL 压缩到 .(当前目录)                                                      执行命令并返回标准输出和错误处理合并的切片
//	pwd, _ := os.Getwd()

	if _, err := exec.Command("tar", "-czf", imageTar, "-C", mntURL, ".").CombinedOutput(); err != nil {

		log.Errorf("Tar folder %s error %v", mntURL, err)
	}
}

