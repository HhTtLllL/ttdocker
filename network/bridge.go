package network

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"net"
	"os/exec"
	"strings"
	"time"
)

type BridgeNetworkDriver struct {

}

func (d *BridgeNetworkDriver) Name () string {
	return "bridge"
}

//创建网络的方法
//输入是  网络的网段, 网关及网络名          返回的是创建好的网络对象
func (d *BridgeNetworkDriver) Create(subnet string, name string) (*Network, error ){
	//通过net 包中的net.ParseCIDR 方法, 取到网段 的字符串中的网关IP地址和网络IP段
	ip, ipRange, _ := net.ParseCIDR(subnet)
	ipRange.IP = ip

	//初始化网络对象
	n := &Network {
		Name: name,
		IpRange: ipRange,
		Driver: d.Name(),
	}

	//配合Linux Bridge
	err := d.initBridge(n)
	if err != nil {
		log.Errorf("error init beidge: %v", err)
	}
	//返回配置好的网络
	return n, err
}
/*
	输入的是网络对象, 执行时会删除网络所对应的网络设备, 而在Bridge Driver 中, 就是删除网络对应的Linux Bridge的设备.
*/
func (d * BridgeNetworkDriver) Delete(network Network) error {
	//网络名即Linux Bridge 的设备名
	bridgeName := network.Name
	//通过netlink库的LinkByName 找到对应的seeing
	br, err := netlink.LinkByName(bridgeName)
	if err != nil {
		return err
	}

	//删除网络对应的 Linux Bridge 设备

	return netlink.LinkDel(br)
}

//连接容器网络端点到Linux Bridge,   连接一个网络和网络端点
/*
先将 Veth 的一端先和 Bridge 连接.
然后再用这一端 和 另外一端连接, 然后再把另外一端移入 namespace
*/
/*
	通过调用Connect 的方法,容器的网络端点已经挂载到了Bridge 网络的 Linux Bridge
*/
func (d *BridgeNetworkDriver) Connect (network *Network, endpoint *Endpoint) error {

	//获取网络名, 即linux Bridge 的接口名
	bridgeName := network.Name
	//通过接口名获取到linux Bridge 接口的对象的接口属性
	br, err := netlink.LinkByName(bridgeName)
	if err != nil {
		return err
	}

	//创建Veth 接口的配置
	la := netlink.NewLinkAttrs()
	//由于Linux 接口名的限制, 名字取endpoint ID 的前五位
	la.Name = endpoint.ID[:5] // 12345

	//通过设置Veth接口的master属性, 设置这个Veth 的一端挂载到网络对应的Linux Bridge 上
	//MasterIndex  must be the index of a bridge
	// 等价于 ip link set dev endpoint.ID[:5] master testbridge
	la.MasterIndex = br.Attrs().Index

	//创建Veth 对象, 通过PeerName 配置Veth另外一端的接口名
	//配置Veth 另外一端的名字 cif - {endpoint ID 的前5位}
	endpoint.Device = netlink.Veth{
		LinkAttrs: la,    //Veth 一端的接口名
		PeerName: "cif-" + endpoint.ID[:5],    //Veth 另外一端的接口名
	}

	/*
		调用netlink 的LinkAdd 方法创建出这个 Veth 接口
		因为上面指定了link 的MasterIndex 是网络对应的Linux Bridge, 所以Veth的一端就已经挂载到了网络对应的Linux Bridge 上
		== ip link add endpoint.ID[:5] type veth peer name cif-12345
		在这里创建出一对 Veth
	*/
	if err = netlink.LinkAdd(&endpoint.Device); err != nil {
		return fmt.Errorf("error add endpoint device :#{err} ")
	}

	//调用netlink 的LinkSetUp 方法, 设置Veth启动
	//相当于 ip link set xxx up 的命令
	// == ip link set 12345 up
	if err = netlink.LinkSetUp(&endpoint.Device); err != nil {
		return fmt.Errorf("error add endpoint device:#{err}")
	}

	return nil
}

func (d *BridgeNetworkDriver) Disconnect(network Network, endpoint *Endpoint) error {

	return nil
}

//初始化Bridge 设备
func (d *BridgeNetworkDriver) initBridge(n *Network) error {
	//创建Bridge虚拟设备
	// try to get bridge by name, if it already exists then just exit
	bridgeName := n.Name
	if err := createBridgeInterface(bridgeName); err != nil {

		return fmt.Errorf("Error add btidge :: %s , Error: %v", bridgeName, err )
	}

	//设置Bridge 设备的地址和路由
	//set bridge IP
	gatewayIP := *n.IpRange
	gatewayIP.IP = n.IpRange.IP
	if err := setInterfaceIP(bridgeName, gatewayIP.String()); err != nil {

		return fmt.Errorf("Error assigning address: %s on bridge:: %s with an error of :%v", gatewayIP, bridgeName, err)
	}

	//地洞Bridge 设备
	if err := setInterfaceUP(bridgeName); err != nil {

		return fmt.Errorf("Error set bridge up : %s, error: %v", bridgeName, err)
	}

	//设置iptabels 的SNAT规则
	//Setup iptables
	if err := setupIPTables(bridgeName, n.IpRange); err != nil {

		return fmt.Errorf("Error setting iptables for %s: %v", bridgeName, err)
	}

	return nil
}


func (d *BridgeNetworkDriver)deleteBridge(n *Network) error {

	bridgeName := n.Name

	//get the link
	l, err := netlink.LinkByName(bridgeName)
	if err != nil {
		return fmt.Errorf("getting link with name %s failed:: %v", bridgeName, err)
	}

	//delete the link
	if err := netlink.LinkDel(l); err != nil {
		return fmt.Errorf("failed to remove bridge interface %s delete:: %v", bridgeName, err)
	}

	return nil
}


//创建Linux Bridge 设备
// ip link add name testbridge type bridge
func createBridgeInterface(bridgeName string) error {
	//先检查是否已经存在了同名的Bridge 设备
	//InterfaceByName --- InterfaceByName返回指定名字的网络接口。
	_, err := net.InterfaceByName(bridgeName)
	//如果存在或者报错则返回创建错误
	if err == nil || !strings.Contains(err.Error(), "no such network interface"){

		return err
	}

	//初始化一个 netlink 的Linux基础对象, Link的名字即Bridge虚拟设备的名字
	//create *netlink.Bridge object
	//创建新的 连接属性  接口的配置
	la := netlink.NewLinkAttrs()
	la.Name = bridgeName
	//使用刚才创建的 Link 的属性创建 netlink 的Bridge 对象
	br := &netlink.Bridge{la}
	//调用netlink的Linkadd方法, 创建 Bridge虚拟网络设备
	// netlink 的Linkadd 方法是用来创建虚拟网络设备的 相当于 ip link add xxxx
	if err := netlink.LinkAdd(br); err != nil {

		return fmt.Errorf("Bridge creation failed for bridge %s : %v", bridgeName, err)
	}

	return nil
}

//设置网络接口为UP 状态
/*
	Linux 的网络设备只有设置成UP 状态后才能处理和转发请求. 通过netlnk 的LinkSetUp方法, 将创建的Linux Bridge 设置成UP状态
*/
func setInterfaceUP(interfaceName string) error {

	iface, err := netlink.LinkByName(interfaceName)
	if err != nil {
		return fmt.Errorf("Error retrieving a link named [ %s ]: %v", iface.Attrs().Name, err)
	}

	//通过"netlink" 的LinkSetUp 方法设置接口状态为Up状态
	//等价于 ip link set xxx up 命令
	// ip link set testbridge up
	if err := netlink.LinkSetUp(iface); err != nil {

		return fmt.Errorf("Error enabling interface for %s : %v", interfaceName, err)
	}

	return nil
}


// 设置Bridge 设备的地址和路由
// Set the IP addr of a netlink interface
//设置一个网络接口的IP地址,  力图 setinterfaceIP("testbridge, "192.168.0.1/24")
// 设置网桥地址
//等于 ip addr add
func setInterfaceIP(name string, rawIP string) error {
	retries := 2;

	var iface netlink.Link
	var err error

	for i := 0; i < retries; i ++ {
		//通过 netlink 的LinkByName 方法找到需要设置的网络接口
		//根据名字找到设备
		iface, err = netlink.LinkByName(name)
		if err == nil {
			break
		}

		log.Debugf("error retrieving ne bridge netlink [%s]...retrying", name)
		time.Sleep(2 * time.Second)
	}

	if err != nil {

		return fmt.Errorf("Abandoning retrieving the new bridge link from netlink, Run [ ip link ] to troubleshoot the error: %v", err)
	}
	/*
		由于 netlink.ParseIPNet 是对net.ParesCIDR 的一个封装, 因此可以将net.ParseCIDR 的返回值中的IP 和 net 整合
		返回值中的ipNet既包含了网段的信息, 192.168.0.0/24 也包含了原始的ip 192.168.0.1
	*/
	ipNet, err := netlink.ParseIPNet(rawIP)
	if err != nil {
		return err
	}

	/*
		通过netlink.AddrAdd 个网络接口配置地址, 相当于 ip addr add xxx 的命令
		同时如果配置了地址所在网段的信息, 例如192.168.0.0/24
		还回配置路由表 192.168.0.0/24 转发到这个  testbridge 的网络接口上面
		通过调用 netlink 的AddrAdd方法,配置Linux Bridge 的地址和路由表
	*/
	addr := &netlink.Addr{ ipNet, "", 0, 0, nil}

	//等价于 ip addr 192.xxx.xxx.xxx/24 dev testbridge
	return netlink.AddrAdd(iface, addr)
}


/*
	设置 iptables Linux SNAT 规则

	通过直接执行iptables 命令,创建 SNAT规则, 只要是从这个网桥上出来的包, 都会对其做源IP的转换,
	保证了容器经过宿主机访问到宿主机外部网络请求的包转换成机器IP, 从而能正确的送达和接受
*/

//设置路由
func setupIPTables(bridgeName string, subnet *net.IPNet) error {
	/*
		由于Go语言没有直接操控 iptables 操作的库, 所以需要通过命令的方式来配置
		创建iptables 的命令

		iptables -t nat -A POSTROUTING  -s <bridgeName> ! -o <bridgeName> -j MASQUERADE
	*/
	iptablesCmd := fmt.Sprintf("-t nat -A POSTROUTING -s %s ! -o %s -j MASQUERADE", subnet.String(), bridgeName)
	cmd := exec.Command("iptables", strings.Split(iptablesCmd, " ")...)
	//执行 iptables 命令配置 SNAT 规则
	output, err := cmd.Output()
	if err != nil {
		log.Errorf("iptables Output, %v", output)
	}

	return nil
}