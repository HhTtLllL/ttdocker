package main

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/urfave/cli"
	"./container"
)

var runCommand = cli.Command{

	Name: "run",
	Usage: `Create a container with namespace and cgroup limit ttdocker run -ti [command]`,

	Flags: []cli.Flag{

		cli.BoolFlag{
			Name: "ti",
			Usage: "enable tty",
		},

	},
	Action: func(context *cli.Context) error {

		if len(context.Args()) < 1 {

			return fmt.Errorf("missing container command")
		}
		//fmt.Println(context.Args())

		cmd := context.Args().Get(0)
		tty := context.Bool("ti")

		Run(tty, cmd)

		return nil
	},
}

var initCommand = cli.Command{
	Name: "init",

	Usage: "init container process run user's process in container. ",
	Action: func(context *cli.Context) error {
		log.Infof("init come on")
		cmd := context.Args().Get(0)
		log.Infof("command %s", cmd)
		err := container.RunContainerInitProcess(cmd, nil)

		return err
	},
}

