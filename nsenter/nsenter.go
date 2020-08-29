package nsenter

/*
#include <errno.h>
#include <sched.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <fcntl.h>


//　这里__attribute__((constructor)) 是指，一旦这个报被引用，　那么这个函数就会被自动执行
//类似于构造函数，　会在程序一启动的时候运行
__attribute__((constructor)) void enter_namespace(void) {

	char *ttdocker_pid;
//从环境变量中获取需要进入的PID
	ttdocker_pid = getenv("ttdocker_pid");
	if (ttdocker_pid) {
		//fprintf(stdout, "got mydocker_pid=%s\n", mydocker_pid);
	} else {
		//fprintf(stdout, "missing mydocker_pid env skip nsenter");
		return;
	}
	char *ttdocker_cmd;
	ttdocker_cmd = getenv("ttdocker_cmd");
	if (ttdocker_cmd) {
		//fprintf(stdout, "got mydocker_cmd=%s\n", mydocker_cmd);
	} else {
		//fprintf(stdout, "missing mydocker_cmd env skip nsenter");
		return;
	}
	int i;
	char nspath[1024];
	//需要进入的5中　namespace
	char *namespaces[] = { "ipc", "uts", "net", "pid", "mnt" };

	for (i=0; i<5; i++) {
		//拼接对应的路径/proc/pid/ns/ipc, 类似这样
		sprintf(nspath, "/proc/%s/ns/%s", ttdocker_pid, namespaces[i]);
		int fd = open(nspath, O_RDONLY);
		//这里调用setns系统调用进入对应的namespace
		if (setns(fd, 0) == -1) {
			//fprintf(stderr, "setns on %s namespace failed: %s\n", namespaces[i], strerror(errno));
		} else {
			//fprintf(stdout, "setns on %s namespace succeeded\n", namespaces[i]);
		}
		close(fd);
	}
//在进入的namespac 中执行指定的命令
	int res = system(ttdocker_cmd);
	exit(0);
	return;
}

*/
import "C"
