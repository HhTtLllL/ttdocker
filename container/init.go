package container

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

//每个包都有init() 函数, 程序如果包括这个包，就先执行这个包里面的init() 函数
//这里的 init 函数是在容器内部执行的，也就是说 ， 代码执行到这里后 ， 容器所在的进程其实就已经创建出来了，
//这是本容器执行的第一个进程。
func RunContainerInitProcess() error {

	//logrus.Infof("command %s" ,command)

	//init 进去读取了 父进程传递过来的参数后，然后在子进程内进行了执行， 完成了将用户指定命令传递给子进程的操作
	cmdArray := readUserCommand()
	if cmdArray == nil || len(cmdArray) == 0 {
		return fmt.Errorf("Run Container get user command error , cmdArray is nil")
	}

	/*  3-1
		//  这里的 MountFlag 的意思如下。
		//。 MS NOEXEC 在本文件系统中不允许运行其他程序。
		//。 MS二NOSUID 在本系统中运行程序的时候， 不允许 set-user-ID 或 set-group-ID 。
		//。 MS NODEV 这个参数是自 从 Linux 2.4 以来，所有 mount 的系统都会默认设定的参数。

	defaultMountFlags := syscall.MS_NOEXEC | syscall.MS_NOSUID | syscall.MS_NODEV
	//挂载proc 系统
	syscall.Mount("proc","/proc","proc",uintptr(defaultMountFlags),"")
	argv := []string{command}

	//exec 实现了完成初始化动作并将用户进程运行起来的操作
	//exec 执行command 对应的程序
	if err := syscall.Exec(command, argv, os.Environ()); err != nil {

		logrus.Errorf(err.Error())
	}
*/
	// 3-2

	//改动，调用 exec.LookPath，可以在系统的 PATH 里面寻找命令的绝对路径
	// 举例： 如果输入的命令为 ls, LookPath  处理后的 为 /bin/ls 然后运行起来
	path, err := exec.LookPath(cmdArray[0])

	if err != nil {

		logrus.Errorf("Exec loop path error %v", err)
		return err
	}

	logrus.Infof("find path %s", path)

	if err := syscall.Exec(path, cmdArray[0:], os.Environ()); err != nil {

		logrus.Errorf(err.Error())
	}


	return  nil

}

func readUserCommand() []string {

	//uintptr(3）就是指 index 为 3 的文件描述符，也就是传递进来的管道的一端
	pipe := os.NewFile(uintptr(3), "pipe")

	msg, err := ioutil.ReadAll(pipe)
	if err != nil {

		logrus.Errorf("init read pipe error %v", err)
	}

	msgStr := string(msg)

	return strings.Split(msgStr, " ")
}