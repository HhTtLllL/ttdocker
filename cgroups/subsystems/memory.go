package subsystems

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"
)

type MemorySubSystem struct {

}

//设置 cgroupPath 对应的 cgroup 的内存资源限制
// 这里的cgroupPaht 就是 这个 hierardcy 的根节点 mydockr-cgroup
func (s *MemorySubSystem) Set(cgroupPath string, res *ResourceConfig) error {

	//GetCgroupPath 的作用是获取当前 subsystem 在虚拟文件系统中的路径，
	if subsysCgroupPath, err := GetCgroupPath(s.Name(), cgroupPath, true); err == nil {

		if res.MemoryLimit != "" {

			//writefile 函数向filename指定的文件中写入数据。如果文件不存在将按给出的权限创建文件，否则在写入数据之前清空文件。
			// Join 讲任意数量的路径元素放入一个单一路径里 sbusysCgroupPath/memory.limit_in_bytes
			if err := ioutil.WriteFile(path.Join(subsysCgroupPath, "memory.limit_in_bytes"), []byte(res.MemoryLimit), 0644); err != nil {

				return fmt.Errorf("set cgrup memory fail #{err}")
			}
		}

		return nil
	}else {

		return err
	}
}

//将一个迸程加入到 cgroupPath 对应的 cgroup 中
func (s *MemorySubSystem) Apply(cgroupPath string, pid int) error {

	if subsysCgroupPath, err := GetCgroupPath(s.Name(), cgroupPath, false); err == nil {

		//把进程的pid 写到cgroup 的虚拟文件系统对应目录下的 task 文件中
		if err := ioutil.WriteFile(path.Join(subsysCgroupPath, "tasks"), []byte(strconv.Itoa(pid)), 0644); err != nil {

			return fmt.Errorf("set group proc fail #{err}")
		}

		return nil
	}else {

		return fmt.Errorf("get cgroup #{cgroupPath} error :#{err}")
	}
}

//删除对应 cgroupPath 对应的cgroup
func (s *MemorySubSystem) Remove(cgroupPath string) error {

	if subsysCgoupPath, err := GetCgroupPath(s.Name(), cgroupPath, false); err == nil {

		//删除cgroup 便是删除对应的cgroupPath目录
		return os.RemoveAll(subsysCgoupPath)
	}else {

		return err
	}
}

//返回 cgroup 的名字
func (s *MemorySubSystem) Name() string {

	return "memory"
}
