package network

import (
	"encoding/json"
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
	"net"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"text/tabwriter"
	"ttdocker/container"
)

var (
	defaultNetworkPath = "/var/run/ttdocker/network/network"
	drivers 		   = map[string]NetworkDriver{}
	networks  		   = map[string]*Network{}
)
/*
网络端点是用来连接容器与网络的, 保证容器内部与网络的通信
*/
type Endpoint struct {
	ID 				string `json:"id"`
	Device 			netlink.Veth `json:"dev"`
	IPAddress 		net.IP `json:"ip"`
	MacAddress 		net.HardwareAddr `json:"mac"`
	Network 		*Network
	PortMapping 	[]string                         //端口映射
}

/*
网络的抽象
网络是容器的一个集合在这个网络上的容器可以通过这个网络互相通信，就像挂载到同
一个 Linux Bridge 设备上的网络设备一样，可以直接通过 Bridge 设备实现网络互连 ； 连接到同
一个网络中的容器也可 以通过这个网络和网络中别的容器互连 。 网络中会包括这个网络相关的
配置，比如网络的容器地址段、网络操作所调用的网络驱动等信息 。
*/
type Network struct {
	Name 		string         //网络名
	IpRange 	*net.IPNet     //地址段
	Driver 		string   	   // 网络驱动名
}


/*
网络驱动
网络驱动是一个网络功能中的组件,
*/
type NetworkDriver interface {
	 Name() string												//驱动名
	 Create(subnet string, name string) (*Network, error)		//创建网络
	 Delete(network Network) error								//删除网络
	 Connect(network *Network, ednpoint *Endpoint) error		//连接容器网络端点到网络
	 Disconnect(network Network, endpoint *Endpoint) error		//从网络上移除容器网络端点
}



//将这个网络的配置信息保存在文件系统中
func (nw *Network) dump (dumpPath string) error {
	//检查保存的目录是否存在， 不存在则创建
	if _, err := os.Stat(dumpPath); err != nil {
		if os.IsNotExist(err) {
			os.MkdirAll(dumpPath, 0644)
		} else {
			return err
		}
	}

	//保存的文件名是网络的名字
	nwPath := path.Join(dumpPath, nw.Name)
	//打开保存的文件用于写入， 后面打开的模式参数分别是存在内容清空、只写入、不存在则创建
	nwFile, err := os.OpenFile(nwPath, os.O_TRUNC | os.O_WRONLY | os.O_CREATE, 0644)
	if err != nil {
		logrus.Errorf("error :", err)
		return err
	}
	defer nwFile.Close()

	//通过json的库序列网络对象到json 的字符串
	nwJson, err := json.Marshal(nw)
	if err != nil {
		logrus.Errorf("error: ", err)
		return err
	}

	//将网络配置json字符串写入到文件中
	_, err = nwFile.Write(nwJson)
	if err != nil {
		logrus.Errorf("error: ", err)
		return err
	}

	return nil
}

//从网络的配置目录中的文件读取到网络的配置， 以便网络查询及在这个网络上连接网络端点
func (nw *Network) load(dumpPath string) error {
	//打开配置文件
	nwConfigFile, err := os.Open(dumpPath)
	defer nwConfigFile.Close()
	if err != nil {

		return err
	}
	//从配置文件中读取网络的配置json 字符串
	nwJson := make([]byte, 2000)
	n, err := nwConfigFile.Read(nwJson)
	if err != nil {
		return err
	}

	//通过json 字符串反序列化出网络
	err = json.Unmarshal(nwJson[:n], nw)
	if err != nil {
		logrus.Errorf("Error load nw info", err)
		return err
	}

	return nil
}
func (nw *Network) remove(dumpPath string) error {

	//网络对应的配置文件,即配置目录下的网络名文件
	//检查文件状态,如果文件已经不存在就直接返回
	if _, err := os.Stat(path.Join(dumpPath, nw.Name)); err != nil {
		if os.IsNotExist(err) {
			fmt.Println("已经被删除")
			return nil
		} else {
			return nil
		}
	}else {
		//调用os.reomve 删除这个网络对应的配置文件
		return os.Remove(path.Join(dumpPath, nw.Name))
	}
}


func Init() error{

	//加载网络驱动
	var bridgeDriver = BridgeNetworkDriver{}
	//drivers[bridge]
	drivers[bridgeDriver.Name()] = &bridgeDriver

	//判断网络的配置目录是否存在，不存在则创建
	if _, err := os.Stat(defaultNetworkPath); err != nil {
		if os.IsNotExist(err){
			os.MkdirAll(defaultNetworkPath, 0644)
		}else {
			return err
		}
	}

	//检查网络配置目录中的所有文件
	//filepath.walk(path,func(string, os.fileInfo, error)) 函数会遍历指定的path 目录
	//并且执行第二个参数中的函数指针去处理目录下的每一个文件
	filepath.Walk(defaultNetworkPath, func(nwPath string, info os.FileInfo, err error) error {

		//如果是目录则跳过
		/*if info.IsDir() {
			return nil
		}*/

		//func HasSuffix(s, suffix string) bool
		//判断 s 串中是否包含 suffix 子串
		//如果是目录则跳过
		if strings.HasSuffix(nwPath, "/"){
			return nil
		}

		//加载文件名作为网络名
		//func Split(path string) (dir, file string)
		//Split函数将路径从最后一个斜杠后面位置分隔为两个部分（dir和file） 并返回。
		//如果路径中没有斜杠，函数返回值dir会设为空字符串，
		//file会设为path。两个返回值满足path == dir+file。
		_, nwName := path.Split(nwPath)
		nw := &Network{
			Name: nwName,
		}

		//调用前面介绍的Network.load 方法加载网络的配置信息
		if err := nw.load(nwPath); err != nil  {

			logrus.Errorf("error load network ::%s", err)
		}
		//将网络的配置信息加入到networks 的字典中
		networks[nwName] = nw

		return nil
	})

	return nil
}





//创建网络 			bridge        192.168.0.0/24    testbridge
func CreateNetwork(driver string, subnet string, name string) error {
	//ParseCIDR 是 Golang net 包的函数， 功能是将网段的字符转换成 net.IPNet 的对象
	/*
		func ParseCIDR(s string) (IP, *IPNet, error)
		本函数会返回IP地址和该IP所在的网络和掩码。
		例如，ParseCIDR("192.168.100.1/16")
		会返回IP地址192.168.100.1和IP网络192.168.0.0/16。
		ParseCIDR 将 s 作为一个CIDR 的IP地址和掩码字符窜
	*/
	_, cidr, _ := net.ParseCIDR(subnet)
	//通过IPAM分配网关IP， 获取到网段中第一个IP作为网关的IP，
	gatewayIp, err := ipAllocator.Allocate(cidr)
	if err != nil {

		return err
	}
	cidr.IP = gatewayIp

	//调用指定的网络驱动创建网络， 这里的drivers 字典是各个网络驱动的实例字典,通过调用网络驱动的
	//Create 方法创建网络， 后面会议 Bridge 驱动为例，介绍它的实现
	//drivers[driver] 返回的是一个 NetDriver 网络驱动, 网络驱动创建一个网络   nw
	nw, err := drivers[driver].Create(cidr.String(), name)
	if err != nil {

		return err
	}

	//保存网络信息， 将网络的信息保存在文件系统中， 以便查询和在网络上连接网络端点
	return nw.dump(defaultNetworkPath)
}

// init()已经把网络配置目录中的所有配置文件加载到了networks 字典中
//这里只需要通过遍历这个字典来展示创建的网络
func ListNetwork() {
	// 通过前面 ttdocker ps 时介绍的tabwriter 的库去展示网络
	w := tabwriter.NewWriter(os.Stdout, 12,1,3,' ', 0)
	fmt.Fprint(w, "NAME\tIpRange\tDriver\n")

	//遍历网络信息
	for _, nw := range networks{

		fmt.Fprintf(w, "%s\t%s\t%s\n",
			nw.Name,
			nw.IpRange.String(),
			nw.Driver,
		)
	}
	//输出到标准输出
	if err := w.Flush(); err != nil {
		logrus.Errorf("Flush error %v", err)
		return
	}
}
/*
	删除网络,网关IP
	删除网络对应的网络设备
	删除网络配置文件
*/
func DeleteNetwork(networkName string) error {

	//查找网络是否存在
	nw, ok := networks[networkName]
	if !ok {
		return fmt.Errorf("no such network::%s", networkName)
	}

	//调用IPAM的实例ipAllocator 释放网络网关的IP
	if err := ipAllocator.Release(nw.IpRange, &nw.IpRange.IP); err != nil {
		return fmt.Errorf("Error remove betwork gageway ip:: %s", err)
	}

	//调用网络驱动删除网络创建的设备与配置,后面会以birdge 驱动删除网络为例子介绍如何实现网络驱动删除网络
	if err := drivers[nw.Driver].Delete(*nw); err != nil {
		return fmt.Errorf("Error remove network DriverError::%s", err)
	}

	//从网络的配置目录中删除该网络对应的配置文件
	return nw.remove(defaultNetworkPath)
}


/*
	使容器网络端点的Veth容器端, 以及后续的配置都在容器的Net Namespace 中执行

	将容器的网络端点加入到容器的网络空间中
	并锁定当前程序所执行的线程, 使当前线程进入到容器的网络空间
	返回值是一个函数指针, 执行这个返回函数才会退出容器的网络空间, 回归到宿主机的网络空间
*/
// enLink  为 peerLink
func enterContainerNetns(enLink *netlink.Link, cinfo *container.ContainerInfo) func() {

	/*
		找到容器的Net Namespace
		/proc/[pid]/ns/net 打开这个文件的文件描述符就可以来操作Net Namespace
		而 containerInfo 中的PID, 即容器在宿主机上映射的进程ID
		它对应的/proc/[pid]/ns/net 就是容器内部的Net Namespace
	*/
	f, err := os.OpenFile(fmt.Sprintf("/proc/%s/ns/net", cinfo.Pid), os.O_RDONLY, 0)
	if err != nil {
		logrus.Errorf("error get container net namespace, %v", err)
	}

	//取到文件的文件描述符
	nsFD := f.Fd()
	/*
		锁定当前程序所执行的检查那个, 如果不锁定操作系统线程的话
		Go语言的goroutine 可能会被调度到别的线程上去, 就不能保证一直在所需要的网络空间中了
		所以要调用runtime.LockOSThread 时要先锁定当前程序执行的线程
	*/
	runtime.LockOSThread()

	//修改veth peer 另外一端到容器的namespace 中
	//修改网络端点Veth的另外一端, 将其移动到容器 Net Namespace 中
	if err = netlink.LinkSetNsFd(*enLink, int(nsFD)); err != nil {
		logrus.Errorf("error set link netns, %v", err)
	}

	//获取当前网络的namespace
	//通过 netns.Get()   方法获取当前网络的Net Namespace
	//以便后面从容器的Net Namespace 中退出, 回到原本网络的 Net Namespace 中
	origns, err := netns.Get()
	if err != nil {
		logrus.Errorf("error get current netns, %v", err)
	}

	//调用netns.set 方法,将当前进程加入容器的 Net Namespace
	//设置当前进程到新的网络namespace ,并在函数执行完成之后在恢复到之前的namespace
	if err = netns.Set(netns.NsHandle(nsFD)); err != nil {
		logrus.Errorf("error set netns, %v", err)
	}


	//返回之前Net Namespace 的函数
	//在容器的网络空间中, 执行完容器配置之后调用此函数就可以将程序恢复到原生的 Net Namespace
	return func() {
		//恢复到上面获取到的之前的 Net Namespace
		netns.Set(origns)
		//关闭Namespace 文件
		origns.Close()
		//取消对当前程序的线程锁定
		runtime.UnlockOSThread()
		//关闭Namespace 文件
		f.Close()
	}
}


//配置容器Namespace 中的网络 设备 及 路由
/*
	容器有自己独立的NetNamespace, 需要将网络端点的Veth设备的另外一端移到这个Net Namespace中并配置, 才能给容器插上网线
*/
func configEndpointIpAddressAndRoute(ep *Endpoint, cinfo *container.ContainerInfo) error {

	//通过网络端点中 "Veth" 的另一端
	peerLink, err := netlink.LinkByName(ep.Device.PeerName)
	if err != nil {
		return fmt.Errorf("fail config endpoint: %v", err)
	}

	/*
		将容器的网络端点加入到容器的网络空间中
		并使这个函数下面的操作都在这个网络空间中进行
		执行完函数后,恢复为默认的网络空间
	*/
	defer enterContainerNetns(&peerLink, cinfo)()

	/*
		获取到容器的IP地址及网段, 用于配置容器内部接口地址
		比如容器IP 是 192.168.1.2, 而网络的网段是 192.168.1.0/24
		那么这个产出的IP字符串就是 192.168.1.2/24, 用于容器内 Veth 端点配置
	*/
	interfaceIP := *ep.Network.IpRange
	interfaceIP.IP = ep.IPAddress

	//调用setInterfaceIP 函数设置容器内Veth 端点的IP
	if err = setInterfaceIP(ep.Device.PeerName, interfaceIP.String()); err != nil {
		return fmt.Errorf("%v, %s", ep.Network, err)
	}

	//启动容器内的Veth 端点
	if err = setInterfaceUP(ep.Device.PeerName); err != nil {
		return err
	}

	//Net Namespace 中默认本地地址为127.0.01 的"lo" 网卡是关闭状态的
	//启动它以保证容器访问自己的请求
	if err = setInterfaceUP("lo"); err != nil {
		return err
	}

	//设置容器内的外部请求都通过容器内的Veth 端点访问
	//0.0.0.0/0 的网段, 表示所有的IP地址段
	_, cidr, _ := net.ParseCIDR("0.0.0.0/0")

	//构建要添加的路由数据, 包括网络设备, 网关IP 及 目的网段
	//相当于 route add -net 0.0.0.0/0 gw {Bridge  网桥地址} dev {容器内的 Veth 端点设备}
	defaultRoute := &netlink.Route {
		LinkIndex: peerLink.Attrs().Index,
		Gw: ep.Network.IpRange.IP,
		Dst: cidr,
	}

	//调用netlink的 RouteAdd 添加路由到容器的网络空间
	//RouteAdd 函数相当于 route add 命令
	if err = netlink.RouteAdd(defaultRoute); err != nil {
		return err
	}

	return nil

}

//配置端口映射
func configPortMapping(ep *Endpoint, cinfo *container.ContainerInfo) error {

	//遍历容器端口映射列表
	for _, pm := range ep.PortMapping{
		//分割成宿主机的端口和容器的端口
		portMapping := strings.Split(pm, ":")
		if len(portMapping) != 2 {
			logrus.Errorf("port mapping format error, %v", pm)
			continue
		}

		//由于iptables 没有Go语言版本的实现, 所以采用exec.command 的方式直接调用命令配置
		//在 iptables 的PEROUTING 中添加DNAT 规则
		//将宿主机的端口请求转发到容器的地址和端口上
		iptablesCmd := fmt.Sprintf("-t nat -A PREROUTING -p tcp -m tcp --dport %s -j DNAT --to-destination %s:%s",
			portMapping[0], ep.IPAddress.String(), portMapping[1])


		//执行iptables 命令, 添加端口映射转发规则
		cmd := exec.Command("iptables", strings.Split(iptablesCmd, " ")...)

		output, err := cmd.Output()
		if err != nil {
			logrus.Errorf("iptables output, %v", output)
			continue
		}
	}

	return nil
}

//挂载容器端点流程的调用分解
func Connect(networkName string, cinfo *container.ContainerInfo) error {

	//从networks 字典中取到容器连接的网络信息， networks 字典中保存了当前已经创建的网络
	//从network 数组中取到网络的配置信息,如果找不到网络则返回错误
	network, ok := networks[networkName]
	if !ok {
		return fmt.Errorf("No such network ::%s", networkName)
	}

	//分配容器IP地址 从网络的IP段, 分配容器IP地址
	//通过调用IPAM 从网络的网段中获取可用的IP作为容器的IP地址
	ip, err := ipAllocator.Allocate(network.IpRange)
	if err != nil {
		return err
	}

	//创建网络端点
	//设置网络端点的IP, 网络和端口映射信息, 以供下面配置调用
	ep := &Endpoint{
		ID: fmt.Sprintf("%s-%s", cinfo.Id, networkName),
		IPAddress: ip,
		Network: network,
		PortMapping: cinfo.PortMapping,
	}

	//调用网络对应的网络驱动挂载和配置网络端点
	/*
		调用网络驱动的 " connect " 方法去连接和配置网络端点
	*/
	if err = drivers[network.Driver].Connect(network, ep); err != nil {
		return err
	}

	//进入到容器的网络namespace 配置容器网络设备的IP地址和路由
	//进入到容器网络的namespace 配置容器网络, 设备IP地址和路由信息
	if err = configEndpointIpAddressAndRoute(ep, cinfo); err != nil {
		return err
	}

	//配置容器到宿主机的端口映射
	//配置端口映射信息, 例如 ttdocker run -p 8080:80
	return configPortMapping(ep, cinfo)
}

func Disconnect(networkName string, cinfo *container.ContainerInfo) error {

	return nil
}