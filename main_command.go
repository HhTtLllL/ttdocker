package main

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/urfave/cli"
	"ttdocker/cgroups/subsystems"
	"ttdocker/container"
)

//定义了runCommand 的Flags， 其作用类似于命令运行时使用 -- 来指定参数
var runCommand = cli.Command{

	Name: "run",
	Usage: `Create a container with namespace and cgroup limit ttdocker run -ti [command]`,

	Flags: []cli.Flag{

		cli.BoolFlag{
			Name: "ti",
			Usage: "enable tty",
		},

		cli.StringFlag{
			Name: "m",
			Usage: "memory limit",
		},
		cli.StringFlag{
			Name: "cpushare",
			Usage: "cpushare limit",
		},

		cli.StringFlag{
			Name: "cputest",
			Usage: "cpuset limit",
		},

	},
	/*
		这里是 run 命令执行的真正函数
		1.判断参数是否包含 command
		2.获取用户指定的command
		3.调用 Run function 去准备启动容器
	*/
	Action: func(context *cli.Context) error {

		if len(context.Args()) < 1 {

			return fmt.Errorf("missing container command")
		}

		var cmdArray []string

		for _, arg := range context.Args(){

			cmdArray = append(cmdArray, arg)
		}
		//fmt.Println(context.Args())
	//	cmd := context.Args().Get(0)
		tty := context.Bool("ti")

		resConf := &subsystems.ResourceConfig{

			//取出各个字段对应的参数值
			MemoryLimit: context.String("m"),
			CpuSet: context.String("cpuset"),
			CpuShare: context.String("cpushare"),
		}


		Run(tty, cmdArray,resConf)

		return nil
	},
}

var initCommand = cli.Command{
	Name: "init",

	Usage: "init container process run user's process in container. ",
	Action: func(context *cli.Context) error {
		log.Infof("init come on")
		//cmd := context.Args().Get(0)
		//log.Infof("command %s", cmd)
		err := container.RunContainerInitProcess()

		return err
	},
}

