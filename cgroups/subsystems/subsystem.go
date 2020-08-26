package subsystems

//用于传递资源限制配置的结构体，包含内存限制，cup 时间权重， cpu 核心数
type ResourceConfig struct {
	MemoryLimit string
	CpuShare string
	CpuSet string
}

//Subsystem 借口， 每个Subsystem 借口可以实现下面的四个接口
//这里将cgroup 抽象成了path， 原因是cgroup 在 hierarchy 的路径，便是虚拟文件系统中的虚拟路径
type Subsystem interface {
	Name() string  //返回 subsystem 的名字, 如　cpu  memory
	Set(path string, res *ResourceConfig) error  //设置某个cgroup 在这个子系统中的资源
	Apply(path string, pid int) error  //将进程添加到某个 cgroup 中
	Remove(path string) error //移除某个cgroup
}

//通过不同的subsystem 初始化实例 创建资源限制处理链数组

var (
	SubsystemsIns = []Subsystem{
		&CpusetSubSystem{},
		&MemorySubSystem{},
		&CpuSubSystem{},
	}
)


