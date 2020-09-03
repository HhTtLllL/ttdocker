package network

import (
	"github.com/vishvananda/netlink"
	"net"
)

var (
	defaultNetworkPath = "/var/run/ttdocker/network/network"
	drivers = map[string]NetworkDriver{}
	networks = map[string]*Network{}
)

type Endpoint struct {
	ID string `json:"id"`
	Device netlink.Veth `json:"dev"`
	IPAddress net.IP `json:"ip"`
	MacAddress net.HardwareAddr `json:"mac"`
	Network * Network
	PortMapping []string
}

type Network struct {
	Name string
	IpRange *net.IPNet
	Driver string
}


type NetworkDriver interface {
	 Name() string
	 Create(subnet string, name string) (*Network, error)
	 Delete(network Network) error
	 Connect(network *Network, ednpoint *Endpoint) error
	 Disconnet(network Network, endpoint *Endpoint) error
}

//创建网络
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
	//通过IPAM分配网关 IP, 获取到网段中第一个IP作为网关的IP

	gatewayIp, err := ipAllocator.Allocate(cidr)
	if err != nil {

		return err
	}
	cidr.IP = gatewayIp

	nw, err := drivers[driver].Create(cidr.String(), name)
	if err != nil {

		return err
	}

	//保存网络信息， 将网络的信息保存在文件系统中， 以便查询和在网络上连接网络端点

	return nw.dump(defaultNetworkPath)
}

func (nw *Network) dump (dumpPath string) error {
	//检查保存的目录是否存在， 不存在则创建
}