package main

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/urfave/cli"
	"os"
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
		cli.StringFlag{
			Name: "v",
			Usage: "volume",
		},
		cli.BoolFlag{
			Name: "d",
			Usage: "detach container",
		},
		//　提供 run 后面的 -name 指定容器名字参数
		cli.StringFlag{
			Name: "name",
			Usage: "container name",
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

			fmt.Println(context.Args())
			return fmt.Errorf("missing container command")
		}

		var cmdArray []string

		for _, arg := range context.Args(){

			cmdArray = append(cmdArray, arg)
		}
		//fmt.Println(context.Args())
	//	cmd := context.Args().Get(0)
		createTty := context.Bool("ti")
		detach := context.Bool("d")
		volume := context.String("v")

		if createTty && detach {

			return fmt.Errorf("ti and d paramter can not both provided")
		}

		resConf := &subsystems.ResourceConfig{

			//取出各个字段对应的参数值
			MemoryLimit: context.String("m"),
			CpuSet: context.String("cpuset"),
			CpuShare: context.String("cpushare"),
		}

		containerName := context.String("name")
		fmt.Println("name = ", containerName)

		Run(createTty, cmdArray,resConf, volume, containerName)

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

var commitCommand = cli.Command{
	Name: "commit",
	Usage: "commit a container into image",

	Action: func(context *cli.Context) error {

		if len(context.Args()) < 1 {

			return fmt.Errorf("missing container name")
		}
		imageName := context.Args().Get(0)

		commitContainer(imageName)

		return nil
	},
}

var listCommand = cli.Command{
	Name: "ps",
	Usage: "list all the containers",

	Action: func(context *cli.Context) error{

		ListContainers()
		return nil
	},
}

var logCommand = cli.Command{
	Name: "logs",
	Usage: "print logs of a container",
	Action: func(context *cli.Context) error {

		if len(context.Args()) < 1 {
			return fmt.Errorf("please input your container name")
		}

		containerName := context.Args().Get(0)
		logContainer(containerName)

		return nil
	},
}

var execCommand = cli.Command{
	Name: "exec",
	Usage: "exec a command into container",
	Action: func(context *cli.Context) error {
		//This is for callback
		/*if os.Getenv(ENV_EXEC_PID) != "" {

			log.Infof("pid callback pid %s", os.Getgid())
			return nil
		}*/

		fmt.Println("env_pid11 = ",os.Getenv(ENV_EXEC_PID))
		if os.Getenv(ENV_EXEC_PID) != "" {

			log.Infof("pid callback pid %s", os.Getgid())
			return nil
		}

		if len(context.Args()) < 2 {
			return fmt.Errorf("Missing container name or command")
		}
		//获取容器的名字, 从命令行获取
		containerName := context.Args().Get(0)
		//将命令以切片的方式保存
		var commandArray []string

		for _,arg := range context.Args().Tail() {

			commandArray = append(commandArray, arg)
		}

		//执行命令
		fmt.Println("开始执行命令")
		ExecContainer(containerName, commandArray)

		return nil
	},
}

var stopCommand = cli.Command{
	Name: "stop",
	Usage: "stop a container",
	Action: func(context *cli.Context) error {
		if len(context.Args()) < 1 {
			return fmt.Errorf("Miss container name")
		}

		containerName := context.Args().Get(0)
		stopContainer(containerName)

		return nil
	},
}