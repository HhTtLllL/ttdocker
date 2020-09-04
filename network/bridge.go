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

func (d * BridgeNetworkDriver) Delete(network Network) error {
	bridgeName := network.Name
	br, err := netlink.LinkByName(bridgeName)
	if err != nil {
		return err
	}

	return netlink.LinkDel(br)
}

func (d *BridgeNetworkDriver) Connect (network *Network, endpoint *Endpoint) error {

	bridgeName := network.Name
	br, err := netlink.LinkByName(bridgeName)
	if err != nil {
		return err
	}

	la := netlink.NewLinkAttrs()
	la.Name = endpoint.ID[:5]
	la.MasterIndex = br.Attrs().Index

	endpoint.Device = netlink.Veth{
		LinkAttrs: la,
		PeerName: "cif-" + endpoint.ID[:5],
	}

	if err = netlink.LinkAdd(&endpoint.Device); err != nil {
		return fmt.Errorf("error add endpoint device :#{err} ")
	}

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
func createBridgeInterface(bridgeName string) error {

	//先检查是否已经存在了同名的Bridge 设备
	_, err := net.InterfaceByName(bridgeName)
	//如果存在或者报错则返回创建错误
	if err == nil || !strings.Contains(err.Error(), "no such network interface ----"){

		return err
	}

	//初始化一个 netlink 的Linux基础对象, Link的名字即Bridge虚拟设备的名字
	//create *netlink.Bridge object
	la := netlink.NewLinkAttrs()
	la.Name = bridgeName
	//使用刚才创建的 Link 的属性创建 netlink 的Bridge 对象
	br := &netlink.Bridge{la}
	//调用netlink的Linkadd方法, 创建 Bridge虚拟网络设备
	// netlink 的Linkadd 方法是用啦床架你虚拟网络设备的 相当于 ip link add xxxx
	if err := netlink.LinkAdd(br); err != nil {

		return fmt.Errorf("Bridge creation failed for bridge %s : %v", bridgeName, err)
	}

	return nil

}

func setInterfaceUP(interfaceName string) error {

	iface, err := netlink.LinkByName(interfaceName)
	if err != nil {
		return fmt.Errorf("Error retrieving a link named [ %s ]: %v", iface.Attrs().Name, err)
	}

	if err := netlink.LinkSetUp(iface); err != nil {

		return fmt.Errorf("Error enabling interface for %s : %v", interfaceName, err)
	}

	return nil
}


// 设置Bridge 设备的地址和路由
// Set the IP addr of a netlink interface
func setInterfaceIP(name string, rawIP string) error {
	retries := 2;

	var iface netlink.Link
	var err error

	for i := 0; i < retries; i ++ {
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

	ipNet, err := netlink.ParseIPNet(rawIP)
	if err != nil {
		return err
	}

	addr := &netlink.Addr{ ipNet, "", 0, 0, nil}

	return netlink.AddrAdd(iface, addr)
}




func setupIPTables(bridgeName string, subnet *net.IPNet) error {
	iptablesCmd := fmt.Sprintf("-t nat -A POSTROUTING -s %s ! -o %s -j MASQUERADE", subnet.String(), bridgeName)
	cmd := exec.Command("iptables", strings.Split(iptablesCmd, " ")...)

	output, err := cmd.Output()
	if err != nil {
		log.Errorf("iptables Output, %v", output)
	}
	return err
}