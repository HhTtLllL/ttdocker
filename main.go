package main

import(
	"github.com/urfave/cli"
	log "github.com/Sirupsen/logrus"
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
	}


	app.Before = func(context *cli.Context) error {

		log.SetFormatter(&log.JSONFormatter{})
		log.SetOutput(os.Stdout)

		return nil
	}

	if err := app.Run(os.Args); err!= nil {

		log.Fatal(err)
	}





}