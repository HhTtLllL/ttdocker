package main

import (
	log "github.com/Sirupsen/logrus"
	"github.com/urfave/cli"
	"os"
)

const usage = "test ttdocker"

func main(){

	//创建一个命令实例
	app := cli.NewApp()
	app.Name = "ttdocker"
	app.Usage = usage
	app.Version = "0.1.1"

	app.Commands = []cli.Command{
		initCommand,
		runCommand,
		commitCommand,					//把运行状态容器的内存存储成镜像保存下来
		listCommand,
		logCommand,
		execCommand,
		stopCommand,
		removeCommand,
		networkCommand,
	}

	//初始化 日志配置
	//在app run 执行之前执行的
	app.Before = func(context *cli.Context) error {

		log.SetFormatter(&log.JSONFormatter{})
		log.SetOutput(os.Stdout)

		return nil
	}

	//运行app run
	if err := app.Run(os.Args); err!= nil {

		log.Fatal(err)
	}
}
