package main

import (
	log "github.com/Sirupsen/logrus"
	"github.com/urfave/cli"
	"os"
)

const usage = "test ttdocker"

func main(){

	app := cli.NewApp()
	app.Name = "ttdocker"
	app.Usage = usage

	app.Commands = []cli.Command{
		initCommand,
		runCommand,
		//目的是把运行状态容器的内存存储成镜像保存下来
		commitCommand,
		listCommand,
		logCommand,
		execCommand,
	}


	//初始化 日志配置
	//在app run 执行之前执行的
	app.Before = func(context *cli.Context) error {

		log.SetFormatter(&log.JSONFormatter{})
		log.SetOutput(os.Stdout)

		return nil
	}

	if err := app.Run(os.Args); err!= nil {

		log.Fatal(err)
	}
}
