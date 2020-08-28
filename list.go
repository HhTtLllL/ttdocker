package main

import (
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"io/ioutil"
	"os"
	"text/tabwriter"
	"ttdocker/container"
)

func ListContainers(){

	//　找到存储容器信息的路径 /var/run/ttdocker
	dirURL := fmt.Sprintf(container.DefaultInfoLocation, "")
	dirURL = dirURL[:len(dirURL) - 1]

	//读取文件夹下的所有内容
	files, err := ioutil.ReadDir(dirURL)
	if err != nil {
		log.Errorf("Read idr %s error %v", dirURL, err)
		return
	}

	var containers []*container.ContainerInfo
	//遍历该文件下的所有文件
	for _, file := range files {
		//根据容器配置文件获取对应的信息，　然后转换成容器信息的对象
		tmpContainer, err := getContainerInfo(file)

		if err != nil {
			log.Errorf("Get container info error %v", err)
			continue
		}
		containers = append(containers, tmpContainer)
	}

	//使用tabWrite.NewWrite 在控制台打印出容器信息
	//tabwrite 是引用 texttabwriter 类库, 用于在控制台打印对齐的表格
	w := tabwriter.NewWriter(os.Stdout, 12, 1, 3, ' ', 0)

	//控制台输出的信息列
	fmt.Fprint(w, "ID\tNAME\tPID\tSTATUS\tCOMMAND\tCREATED\n")

	for _,item := range containers {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			item.Id,
			item.Name,
			item.Pid,
			item.Status,
			item.Command,
			item.CreatedTime)
	}

	//刷新标准输出流缓存区, 将容器里列表打印出来

	if err := w.Flush(); err != nil {

		log.Errorf("Flush error %v", err)
		return
	}
}


func getContainerInfo(file os.FileInfo) (* container.ContainerInfo, error) {

	//获取文件名
	containerName := file.Name()
	//根据文件名生成文件绝对路径
	configFileDir := fmt.Sprintf(container.DefaultInfoLocation, containerName)
	configFileDir = configFileDir + container.ConfigName

	//读取config.json 文件内的容器信息
	content, err := ioutil.ReadFile(configFileDir)

	if err != nil {

		log.Errorf("read file %s error %v", configFileDir, err)
		return nil, err
	}

	var containerInfo container.ContainerInfo
	//将json 文件信息反序列化成容器信息对象

	if err := json.Unmarshal(content, &containerInfo); err != nil {

		log.Errorf("Json umarshal error %v", err)
		return nil, err
	}


	return &containerInfo, nil
}