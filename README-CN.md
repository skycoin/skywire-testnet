![skywire logo](https://user-images.githubusercontent.com/26845312/32426764-3495e3d8-c282-11e7-8fe8-8e60e90cb906.png)

# Skywire
Skywire当前还处于开发阶段,可以浏览我们的[博客](https://blog.skycoin.net/tags/skywire/)了解更多关于Skywire的消息

### 运行环境
* golang 1.9+

  https://golang.org/dl/

* git

* setup $GOPATH env (for example: /go)
  https://github.com/golang/go/wiki/SettingGOPATH

## 安装过程
### Linux/Mac Unix系统

#### 打开终端命令行
```
mkdir -p $GOPATH/src/github.com/skycoin
cd $GOPATH/src/github.com/skycoin
git clone https://github.com/skycoin/skywire.git
```

编译Skywire
```
cd $GOPATH/src/github.com/skycoin/skywire/cmd
go install ./...
```

#### 编译Skywire
```
cd $GOPATH/src/github.com/skycoin/skywire/cmd
go install ./...
```
编译好的Skywire程序在$GOPATH/bin

## 运行 Skywire

### Linux/Mac Unix系统
```
cd $GOPATH/bin
./manager -web-dir ${GOPATH}/src/github.com/skycoin/skywire/static/skywire-manager
```

#### 新建一个新的终端命令行

```
cd $GOPATH/bin
./node -connect-manager -manager-address :5998 -manager-web :8000 -discovery-address messenger.skycoin.net:5999-028667f86c17f1b4120c5bf1e58f276cbc1110a60e80b7dc8bf291c6bec9970e74 -address :5000 -web-port :6001
```


## 打开 Skywire 管理页面

浏览器打开 "http://127.0.0.1:8000"
打开管理页面需要登录,默认密码:**1234**,可以通过登录后首页面点击Update Password进行更改密码

## 使用 Skywire 管理页面连接App

### 连接可用App (当前可用App: Shadowsocks)
浏览器打开 "http://127.0.0.1:8000",点击已经启动的Node进入Node信息页,找到Shadowsocks Client并点击"Enter the key for node and app"处输入Node Key和App Key,也可以点击"Search services"进行搜索可用APP

### 使用
默认正常启动后,App会显示**可用端口** (如:9443)

### 使用Firefox浏览器

#### 安装 FoxyProxy Standard
打开Firefox浏览器,地址栏输入"https://addons.mozilla.org/zh-CN/firefox/addon/foxyproxy-standard/", 点击"添加到 Firefox"按钮按照提示进行安装

#### 配置 FoxyProxy Standard
安装完成后,Firefox浏览地址栏输入"about:addons"进入插件页面,找到"FoxyProxy Standard"并点击首选项进入配置页面<br>
选择"Use Enabled Proxies By Patterns and Priority"启用FoxyProxy<br>
点击"Add"进行添加配置,
```
Proxy Type: SOCKS5
IP address, DNS name, server name: 127.0.0.1
Port: 可用端口
```
最后点击"Save"