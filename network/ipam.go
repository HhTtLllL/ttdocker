package network

import (
	"encoding/json"
	log "github.com/Sirupsen/logrus"
	"net"
	"os"
	"path"
	"strings"
)

//实现网络中IP地址的分配， 即如何管理网段中IP地址的分配与释放
const ipamDefaultAllocatorPath = "/var/run/ttdocker/network/ipam/subnet.json"

/*
	IPAM 也是网络功能中的一个组件，用于网络IP地址的分配和释放， 包括容器的IP地址和网络网关的IP地址
	主要功能:
		IPAM.Allocate(subnet *net.IPNet) 从指定的subnet 网段中分配IP地址
		IPAM.Release(subnet net.IPNet, ipaddr net.IP) 从指定的subnet 网段中释放掉指定的IP地址
*/

//存放 IP 地址分配信息
type IPAM struct {
	//分配文件存放位置
	SubnetAllocatorPath string
	//网段和位图算法的数组 map,key是网段,value 是分配的位图数组
	Subnets *map[string]string
}

//初始化一个IPAM 的对象, 默认使用 "/var/run/ttdocker/network/ipam/subnet.json" 作为分配信息存储位置
var ipAllocator = &IPAM {
	SubnetAllocatorPath: ipamDefaultAllocatorPath,
}
//fixme
// 此处使用的位图是使用string 中的一个字符标示一个状态为,实际上可以采用一位表示一个是否分配的状态位，这样资源会有更低的消耗

//加载网段地址分配信息
func (ipam *IPAM) load() error {
	//通过os.stat 函数检查存储文件状态, 如果不存在,则说明之前没有分配,则不需要加载
	if _, err := os.Stat(ipam.SubnetAllocatorPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		} else {
			return err
		}
	}

	//打开并读取存储文件
	subnetConfigFile, err := os.Open(ipam.SubnetAllocatorPath)
	defer subnetConfigFile.Close()
	if err != nil {
		return err
	}

	subnetJson := make([]byte, 2000)
	n, err := subnetConfigFile.Read(subnetJson)
	if err != nil {
		return err
	}

	//将文件中的内容反序列化出IP的分配信息
	err = json.Unmarshal(subnetJson[:n], ipam.Subnets)
	if err != nil {
		log.Errorf("Error dump allocation info, %v", err)
		return err
	}
	return nil
}

//存储网段地址分配信息
func (ipam *IPAM)dump() error {

	//检查存储文件所在文件夹是否存在,如果不存在则创建,path.Split 函数能够分隔目录和文件
	ipamConfigFileDir, _ := path.Split(ipam.SubnetAllocatorPath)
	if _,err := os.Stat(ipamConfigFileDir); err != nil {
		if os.IsNotExist(err){
			//创建文件夹, os.mkdirall 相当于 mkdir -p <dir> 命令
			os.MkdirAll(ipamConfigFileDir, 0644)
		}else {
			return err
		}
	}

	//打开存储文件,os.O_TRUNC 表示如果存在则清空, os.O_CREATE 表示如果不存在则创建
	subnetConfigFile, err := os.OpenFile(ipam.SubnetAllocatorPath, os.O_TRUNC | os.O_WRONLY | os.O_CREATE, 0644)
	defer subnetConfigFile.Close()
	if err != nil {
		return err
	}

	//序列化ipam 对象到json 串中
	ipamConfigJson, err := json.Marshal(ipam.Subnets)
	if err != nil {
		return err
	}
	//将序列化后的json 串写入到配置文件中
	_, err = subnetConfigFile.Write(ipamConfigJson)
	if err != nil {
		return err
	}

	return nil
}

//通过位图算法 分配IP地址
//这个函数 用来实现在网段中分配一个可用的IP地址,并将IP地址分配信息记录到文件中
//  从指定的subnet 网段中分配IP地址
func (ipam *IPAM) Allocate(subnet *net.IPNet) (ip net.IP, err error ){
	//存放网段中地址分配信息的数组
	ipam.Subnets = &map[string]string{}

	//从文件中加载已经分配的网段信息
	err = ipam.load()
	if err != nil {
		log.Errorf("Error dump allocation info, %v", err)
	}

	_, subnet, _ = net.ParseCIDR(subnet.String())
	/*
		net.IPNet.Mask.Size() 函数会返回网段的子网掩码的总长度和网段前面固定位的长度
		比如： 127.0.0.0/8  网段的子网掩码是 255.0.0.0
		subnet.Mask.Size() 的返回值就是前面 255 所对应的位数的总位数， 即 8 和 24
	*/
	one, size := subnet.Mask.Size()

	//如果之前没有分配过这个网段，则初始化网段的分配配置
	if _, exist := (*ipam.Subnets)[subnet.String()]; !exist{
		/*
			初始化
			用 0 填满这个网段的配置,
			1 << uint8(size- one) 表示这个网段中有多少个可用地址
			size - one 是子网掩码后面的网络位数,2^(size - one) 表示网段中的可用IP数
			而2^(size - one)等价于 1 << uint8(size - one)
		*/
		(*ipam.Subnets)[subnet.String()] = strings.Repeat("0", 1 << uint8(size - one))
	}

	//遍历网段的位图数组
	for c:= range ((*ipam.Subnets)[subnet.String()]) {
		//找到数组中为 0,的项和数组序号,即可以分配的IP
		if (*ipam.Subnets)[subnet.String()][c] == '0' {
			ipalloc := []byte((*ipam.Subnets)[subnet.String()])
			//Go的字符串, 创建以后就不能修改, 所以通过转换成byte 数组, 修改后在转换成字符串赋值
			ipalloc[c] = '1'
			(*ipam.Subnets)[subnet.String()] = string(ipalloc)
			//这里的IP为初始IP, 比如对于网段 192.168.0.0/16,这里就是 192.168.0.0
			ip = subnet.IP

			/*
				通过网段的IP与上面的偏移相加计算出分配的IP地址,由于IP地址是uint的一个数组
				需要通过数组中的每一项加所需要的值,
				举例:
			网段是172.16.0.0/12  数组序号是 65555,
			那么在[172,16,0,0] 上依次加[uint8(65555 >> 24), uint8(65555 >> 16), uint8(65555 >> 8), uint8(65555 >> 0)]
			即[0, 1, 0 ,19], 那么获得的IP就是 172.17.0.19
			*/
			for t := uint(4); t > 0; t -= 1 {
				[]byte(ip)[4-t] += uint8(c >> ((t - 1) * 8))
			}

			//由于此处IP是从1开始分配的, 所以最后在加 1, 最终得到分配的IP是 172.17.0.20
			ip[3] += 1
			break
		}
	}

	//调用dump 将分配结果保存到文件中
	ipam.dump()
	return
}


//从指定的subnet 网段中释放掉指定的IP地址
func (ipam *IPAM) Release(subnet *net.IPNet, ipaddr *net.IP) error {

	ipam.Subnets = &map[string]string{}

	_, subnet, _ = net.ParseCIDR(subnet.String())

	err := ipam.load()
	if err != nil {
		log.Errorf("Error dump allocation info, %v", err)
	}

	c := 0
	releaseIP := ipaddr.To4()
	releaseIP[3] -= 1

	for t := uint(4); t > 0; t -= 1 {
		c += int(releaseIP[t - 1] - subnet.IP[t - 1]) << ((4 - t) * 8)
	}

	ipalloc := []byte((*ipam.Subnets)[subnet.String()])
	ipalloc[c] = '0'
	(*ipam.Subnets)[subnet.String()] = string(ipalloc)

	ipam.dump()
	return nil
}