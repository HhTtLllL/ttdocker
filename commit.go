package main

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"os"
	"os/exec"
)

func commitContainer(imageName string){

	mntURL := "/root/mnt"
	imageTar := "/root/" + imageName + ".tar"

	//fmt.Printf("#{imageTar}")
	// tar -czf /root/imnaeName.tar -C mntURL .
	//将 mntURL 压缩到 .(当前目录)                                                      执行命令并返回标准输出和错误处理合并的切片
	pwd, _ := os.Getwd()
	fmt.Println("pwd  = ",pwd)
	if _, err := exec.Command("tar", "-czf", imageTar, "-C", mntURL, ".").CombinedOutput(); err != nil {

		log.Errorf("Tar folder %s error %v", mntURL, err)
	}
}

