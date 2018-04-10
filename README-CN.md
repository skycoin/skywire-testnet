![skywire logo](https://user-images.githubusercontent.com/26845312/32426764-3495e3d8-c282-11e7-8fe8-8e60e90cb906.png)

# Skywire
Skywire当前还处于开发阶段，如果没有太多技术背景，请等待年后发布的版本。也可以浏览我们的[博客](https://blog.skycoin.net/tags/skywire/)了解更多关于Skywire的消息

## 目录
* [运行环境](#requirements)
* [安装 Skyire](#install-skywire)
* [运行 Skywire](#run-skywire)
* [打开 Skywire 管理页面](#open-skywire-manager)
* [使用 Skywire App](#use-skywire-app)
* [系统镜像下载地址](#images)

<a name="requirements"></a>

## 运行环境
* golang 1.9+

  https://golang.org/dl/

* git

* setup $GOPATH env (for example: /go)
  https://github.com/golang/go/wiki/SettingGOPATH

<a name="install-skywire"></a>

## 安装 Skywire (Linux/Mac Unix系统)

### 打开终端命令行
```
mkdir -p $GOPATH/src/github.com/skycoin
cd $GOPATH/src/github.com/skycoin
git clone https://github.com/skycoin/skywire.git
```

### 编译Skywire
```
cd $GOPATH/src/github.com/skycoin/skywire/cmd
go install ./...
```
编译好的Skywire程序在$GOPATH/bin

<a name="run-skywire"></a>

## 运行 Skywire

### Linux/Mac Unix系统

#### 运行 Skywire Manager
```
cd $GOPATH/bin
./manager -web-dir ${GOPATH}/src/github.com/skycoin/skywire/static/skywire-manager
```
`提示:如果使用以上命令运行,您将不可以关闭当前窗口,否则将会关闭 Skywire Manger。`

如果你需要关闭当前窗口,并继续运行 Skywire Manager，可以使用：

```
cd $GOPATH/bin
nohup ./manager -web-dir ${GOPATH}/src/github.com/skycoin/skywire/static/skywire-manager > /dev/null 2>&1 & echo $! > manager.pid
```

`注意：不要同时执行以上两个命令，只需要选择其中一种方式即可`


#### 运行 Skywire Node

打开一个新的terminal

```
cd $GOPATH/bin
./node -connect-manager -manager-address 127.0.0.1:5998 -manager-web 127.0.0.1:8000
```

`提示:如果使用以上命令运行,您将不可以关闭当前窗口,否则将会关闭 Skywire Node。`

如果你需要关闭当前窗口,并继续运行 Skywire Node，可以使用：

```
cd $GOPATH/bin
nohup ./node -connect-manager -manager-address 127.0.0.1:5998 -manager-web 127.0.0.1:8000 > /dev/null 2>&1 & echo $! > node.pid
```

`提示:127.0.0.1:5998 和 127.0.0.1:8000为配置参数，请以你Skywire Manager的IP和端口设置为准`

#### 关闭Skywire Manager 和 Skywire Node
1) 如果使用一直不关闭terminal窗口方式启动Skywire Manager和Node，请在Manager和Node各自terminal上按下Ctrl + c 结束

2) 使用关闭terminal保持运行方式，请输入:

##### 关闭Skywire Manager
```
cd $GOPATH/bin
pkill -F manager.pid
```

##### 关闭Skywire Node
```
cd $GOPATH/bin
pkill -F node.pid
```
`提示：Windows系统请打开任务管理，并查找manager和node进程进行关闭`


<a name="open-skywire-manager"></a>

## 打开 Skywire 管理页面

浏览器打开 "http://127.0.0.1:8000"<br>打开管理页面需要登录,默认密码:**1234**(可以修改)

### Conect to node

浏览器打开 "http://127.0.0.1:8000", 输入密码后进入，选择列表中其中一个Node进入，然后

1) 连接节点(Connect to node)——搜索服务(Search services)——连接 (Connect)

2) 连接节点(Connect to node)——输入节点公钥与 APP 公钥(Enter the key for node and app)——连接(Connect)

在第一种方式下，你可以搜索到全球的节点，并任意选择你要连接的节点;第二种方式则可连接指定的节点

<a name="use-skywire-app"></a>

#### 使用 Skywire App
默认正常启动后,App成功连接后会显示"**可用端口**" (如:9443)

#### 使用Firefox浏览器

#### 安装 FoxyProxy Standard
打开Firefox浏览器,地址栏输入"https://addons.mozilla.org/zh-CN/firefox/addon/foxyproxy-standard/", 点击"添加到 Firefox"按钮按照提示进行安装

#### 配置 FoxyProxy Standard
安装完成后,Firefox浏览地址栏输入"about:addons"进入插件页面,找到"FoxyProxy Standard"并点击首选项进入配置页面<br>选择"Use Enabled Proxies By Patterns and Priority"启用FoxyProxy<br>
点击"Add"进行添加配置,
```
Proxy Type: SOCKS5
IP address, DNS name, server name: 127.0.0.1
Port: 可用端口
```
最后点击"Save"

### SSH 工具

#### SSH
开启此服务后会生成应用公钥，根据节点公钥与此应用公钥，可在任意运行 Skywire 的机器 中远程管理本节点。

`注意：不要随意开启SSH，并将Node Key 和 App Key 展示给陌生人`

#### SSH Client
要求输入Node Key 和 App Key，连接成功后会在按钮下会显示端口(Port)，如：30001，最后使用任意SSH远程连接工具连接上

## Docker

```
docker build -t skycoin/skywire .
```

### 启动Skywire Manager

```
docker run -ti --rm \
  --name=skywire-manager \
  -p 5998:5998 \
  -p 8000:8000 \
  skycoin/skywire
```

浏览器打开 [http://localhost:8000](http://localhost:8000).
默认密码是: **1234**.

### 启动Skywire Node

```
docker volume create skywire-data
docker run -ti --rm \
  --name=skywire-node \
  -v skywire-data:/root/.skywire \
  --link skywire-manager \
  -p 5000:5000 \
  -p 6001:6001 \
  skycoin/skywire \
    node \
      -connect-manager \
      -manager-address skywire-manager:5998 \
      -manager-web skywire-manager:8000 \
      -address :5000 \
      -web-port :6001
```

### Docker Compose

```
docker-compose up
```

## 系统镜像下载地址

<a name="images"></a>

注意:该系统镜像暂时只可以在[Orange Pi Prime](http://www.orangepi.cn/OrangePiPrime/index_cn.html)运行,以下镜像均已安装好Skywire运行环境

### 预设置IP系统镜像

Manager系统镜像包包含Skywire Manager和一个Skywire Node,其它Node系统镜像包只启动一个Node.因路由器默认占用192.168.0.1,所以Manager到Node7系统镜像从192.168.0.2~192.168.0.9为止

1) [Manager](https://skywire.oss-cn-hongkong.aliyuncs.com/skywire-manager.tar.gz)

2) [Node1](https://skywire.oss-cn-hongkong.aliyuncs.com/skywire_node1.tar.gz)

3) [Node2](https://skywire.oss-cn-hongkong.aliyuncs.com/skywire_node2.tar.gz)

4) [Node3](https://skywire.oss-cn-hongkong.aliyuncs.com/skywire_node3.tar.gz)

5) [Node4](https://skywire.oss-cn-hongkong.aliyuncs.com/skywire_node4.tar.gz)

6) [Node5](https://skywire.oss-cn-hongkong.aliyuncs.com/skywire_node5.tar.gz)

7) [Node6](https://skywire.oss-cn-hongkong.aliyuncs.com/skywire_node6.tar.gz)

8) [Node7](https://skywire.oss-cn-hongkong.aliyuncs.com/skywire_node7.tar.gz)

### 手动配置IP系统镜像

`这个系统镜像只包含运行Skywire的基本环境,需要设置IP等,适合具有相关知识人士使用`

[Pure Image](https://skywire.oss-cn-hongkong.aliyuncs.com/skywire_pure.tar.gz)