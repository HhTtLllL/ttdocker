package subsystems

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"strings"
)

//通过  /proc/self/rnountinfo 找出挂载了某个 subsystem 的 hierarchy cgroup 根节点所在的目录
func FindCgroupMountpoint(subsystem string) string {

	f, err := os.Open("/proc/self/mountinfo")
	if err != nil {

		return ""
	}
	defer f.Close()

	//创建并返回一个 从 f 读取数据的 Scanner， 默认的分割函数是 ScanLines
	scanner := bufio.NewScanner(f)

	for scanner.Scan() { //当读完文件中的内容时　退出

		txt := scanner.Text()
		fields := strings.Split(txt, " ") //去掉 txt 中间的空格

		for _, opt := range strings.Split(fields[len(fields) - 1], ","){

			if opt == subsystem {

				return fields[4]
			}
		}
	}
	if err := scanner.Err(); err != nil {

		return ""
	}

	return ""
}

//得到cgroup 在文件系统中的绝对路径
//					subsystem 是 s.name
func GetCgroupPath(subsystem string, cgroupPath string, autoCreate bool) (string, error ){
	/*
			举个例子， 如果这里的 subsystem = memory， 即 即将设置memory对应的资源，
		    对应的cgroupRoot 对应的值 为 /sys/fs/cgroup/memory
			cpuset 则对应的为 /sys/fd/cgroup/cpuset

			然后控制资源 就是在 对应的路径下面创建对应的文件 然后把 限制资源的多少 写入到对应的文件中
		举例： memory  首先要进入当前 hierarchy, 就是进入当前manager 对应的的目录，这个目录在run 期间已经建立，名为 mydaocker-cgroup
			然后就是在 下面创建 memory.limit_in_bytes 这个文件，然后把限制资源的 大小写入文件中就星
	*/
	//stat返回一个描述name指定的文件对象的FileInfo。,如果不存在，根据autocreate 创建一个
	cgroupRoot := FindCgroupMountpoint(subsystem)

	if _, err := os.Stat(path.Join(cgroupRoot, cgroupPath)); err == nil || (autoCreate && os.IsNotExist(err)) {

		if os.IsNotExist(err) {

			if err := os.Mkdir(path.Join(cgroupRoot, cgroupPath), 0755); err == nil {

			} else {

				return "", fmt.Errorf("error create cgroup #{err}")
			}
		}

		return path.Join(cgroupRoot, cgroupPath), nil
	}else{

		return "", fmt.Errorf("cgroup path error #{err}")
	}
}