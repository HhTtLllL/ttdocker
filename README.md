
### 编译 && 运行
编译: go build 
运行:　 ./ttdocker run 
参数说明
 
 

 - -ti 			以交互模式运行
 - -d			后台模式运行
 - --name 给容器指定一个名称
 - -m 设置内存最大值
 - -cpushare 限制CPU时间片片分配比例
 - -volume 指定一个数据卷
 - -p 指定端口映射
 - -e 指定环境变量下运行

其他命令

 - ./ttdocker commit 			镜像打包
 - ./ttdocker ps 					显示所有容器
 - ./ttdocker logs  [容器名]					输出容器日志
 - ./ttdocker exec					重新进入后台运行容器
 - ./ttdocker stop [容器名]	停止容器
 - ./ttdocker rm				删除容器
 - ./ttdocker network create 	创建网络
 - ./ttdocker network list 列举创建的网络
 - ./ttdocker network remove 删除网络
 
 项目介绍: 
 使用Golang语言编写，实现了镜像打包，运行镜像等功能。



项目技术: 

 - 使用Namesoace 进行资源隔离
 - 使用Cgroup进行资源限制
 - 使用AUFS技术进行文件管理
 - 使用Cgo中的exec()实现重新进入容器
 - 使用Veth连接不同的网络Namespace 
 - 使用LinuxBridge 来连接不容的网络设备
 - 通过定义路由表来确定某个网络Namespace中包的流向
 - 使用MASQUERADE 和 DNAT进行网络地址得到转换




ps:此项目是从其他仓库搬迁而来

源仓库链接：[https://github.com/HhTtLllL/go/tree/master/src/ttdocker](https://github.com/HhTtLllL/go/tree/master/src/ttdocker)
