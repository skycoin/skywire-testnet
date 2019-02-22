![skywire logo](https://user-images.githubusercontent.com/26845312/32426764-3495e3d8-c282-11e7-8fe8-8e60e90cb906.png)

# [中文文档](README-CN.md)
# [Spanish Document](README-ES.md)
# [Korean Document](README-KO.md)
# Skywire

스카이와이어 [블로그](https://blog.skycoin.net/tags/skywire/)는 이곳입니다.

스카이와이어는 현재 열심히 개발중입니다.



![2018-01-21 10 44 06](https://user-images.githubusercontent.com/1639632/35190261-1ce870e6-fe98-11e7-8018-05f3c10f699a.png)

## Table of Contents
* [필요사항](#필요사항)
* [설치](#설치)
* [스카이와이어구동](#스카이와이어구동)
* [도커](#도커)
* [시스템 이미지 다운로드 Url](#시스템이미지다운로드)

### 필요사항

* golang 1.9+

  https://golang.org/dl/

* git

* setup $GOPATH env (for example: /go)
  https://github.com/golang/go/wiki/SettingGOPATH

## 설치
### 유닉스 시스템

```
mkdir -p $GOPATH/src/github.com/skycoin
cd $GOPATH/src/github.com/skycoin
git clone https://github.com/skycoin/skywire.git
```

스카이와이어를 위한 바이너리 설치
```
cd $GOPATH/src/github.com/skycoin/skywire/cmd
go install ./...
```

## 스카이와이어구동

### 유닉스 시스템

#### 스카이매니저구동
```
cd $GOPATH/bin
./skywire-manager -web-dir ${GOPATH}/src/github.com/skycoin/skywire/static/skywire-manager
```
`팁:  만약 당신이 위의 명령어를 사용한다면, 현재 열려있는 윈도우를 닫지 못하거나 윈도우 창을 종료할 시 스카이와이어 매니저가 종료될 것입니다.`

만약 현재 열려있는 윈도우창을 닫은 상태에서 스카이와이어 매니저를 구동할 필요가 있다면, 이 명령어를 사용할 수 있습니다.

```
cd $GOPATH/bin
nohup ./skywire-manager -web-dir ${GOPATH}/src/github.com/skycoin/skywire/static/skywire-manager > /dev/null 2>&1 & echo $! > manager.pid

```
`주의: 위의 두 명령어는 동시에 구동할 수 없으며, 둘 중 하나만 사용해야 합니다.`

#### 스카이와이어 노드구동

새로운 윈도우 창을 연다.

```

cd $GOPATH/bin
./skywire-node -connect-manager -manager-address 127.0.0.1:5998 -manager-web 127.0.0.1:8000

```
`팁:  만약 당신이 위의 명령어를 사용한다면, 현재 열려있는 윈도우를 닫지 못하거나 윈도우 창을 종료할 시 스카이와이어 노드가 종료될 것입니다.`

만약 현재 열려있는 윈도우창을 닫은 상태에서 스카이와이어 매니저를 구동할 필요가 있다면, 이 명령어를 사용할 수 있습니다.

```
cd $GOPATH/bin
nohup ./skywire-node -connect-manager -manager-address 127.0.0.1:5998 -manager-web 127.0.0.1:8000 > /dev/null 2>&1 & echo $! > node.pid
```
#### 스카이와이어 매니저 및 노드 중지
1) 만약 스카이와이어 매니저와 노드가 윈도우 터미널에서 실행되고 있는 상태라면, Ctrl+c버튼을 누르면 적용됩니다.
2) 터미널을 계속 사용한 상태에서 종료하려면, 다음을 입력하십시오:

##### 스카이와이어 매니저 종료
```
cd $GOPATH/bin
pkill -F manager.pid
```

##### 스카이와이어 노드 종료
```
cd $GOPATH/bin
pkill -F node.pid
```

## 스카이와이어 매니저 열기

열기 [http://localhost:8000](http://localhost:8000).
스카이와이어 매니저의 초기 패스워드는 **1234** 입니다.

### 노드에 연결
1) 노드에 연결(Connect to node) —— 서비스 검색(Search services) —— 연결(Connect)

2) 노드에 연결(Connect to node) —— 노드/앱 키 입력 —— 연결(Connect)

첫 번째 방식은, 전 세계 노드를 검색한 후, 원하는 노드를 선택하여 연결할 수 있으며; 두번째 방식은 특정한 노드와 연결하는 방식입니다.

#### 스카이와이어 앱 이용
일반적인 경우 응용 프로그램은 연결 성공 시 "** 사용 가능한 포트 **"(예 : 9443)를 표시합니다.

#### 파이어폭스 브라우저 사용

#### FoxyProxy Standard 설치

#### FoxyProxy Standard 설치
파이어폭스 브라우저를 열고, 주소창에 "https://addons.mozilla.org/zh-CN/firefox/addon/foxyproxy-standard/"를 입력 후, "add to Firefox" 버튼을 클릭하여 설치합니다.

#### FoxyProxy Standard 설정
설치가 완료되면 Firefox 주소 표시 줄을 탐색하여 플러그인 페이지에 "addons"를 입력하고 FoxyProxy "Standard"를 찾은 다음 구성 페이지로 환경 설정을 클릭하십시오.< br >사용가능한  "Use Enabled Proxies By Patterns and Priority"을 선택하십시오. <br>
"Add"를 클릭하여 구성을 추가하고,
```
Proxy Type: SOCKS5
IP address, DNS name, server name: 127.0.0.1
Port: 가용포트
```
그리고 마지막으로 "Save"

### SSH 툴
이 서비스가 열리면 응용 프로그램 공개 키가 생성됩니다. 노드의 공용 키와 공용 키를 기반으로 스카이와이어를 실행하는 모든 시스템에서 원격으로 노드를 관리 할 수 있습니다.

`주의: SSH를 열지 않을 결우, 노드 키와 앱 키가 다른사람에게 보여집니다.`

노드 키와 앱 키를 입력하십시오. 연결이 성공하면 단추 아래에 포트 (포트) (예 : 30001)가 표시되고, SSH 원격 연결 도구를 사용하여 연결할 수 있습니다.

## 도커

```
docker build -t skycoin/skywire .
```

### 매니저 실행

```
docker run -ti --rm \
  --name=skywire-manager \
  -p 5998:5998 \
  -p 8000:8000 \
  skycoin/skywire
```
열기 [http://localhost:8000](http://localhost:8000).
스카이와이어 매니저의 초기 패스워드는 **1234** 입니다.


### 노드를 실행시키고, 매니저에 연결합니다.

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

### 도커 구성

```
docker-compose up
```

Open [http://localhost:8000](http://localhost:8000).

참고 : [docker-compose.yml](https://github.com/skycoin/skywire/blob/master/docker-compose.yml) 파일을 편집하는 노드를 더 추가 할 수 있습니다.

## 시스템이미지다운로드
<a name="images"></a>

주의: 이 이미지들은 [오렌지 파이 프라임](http://www.orangepi.cn/OrangePiPrime/index_cn.html)에서만 구동됩니다.

### IP설정 가능 시스템 이미지

매니저 시스템 이미지 패키지에는 스카이와이어 매니저와 스카이와이어 노드가 포함되어 있으며 다른 노드 시스템 이미지 패키지는 노드만 설치되어 있습니다.

1) Download [Manager](https://downloads3.skycoin.net/skywire-images/manager.tar.gz) (IP:192.168.0.2)

2) Download [Node1](https://downloads3.skycoin.net/skywire-images/node-1-03.tar.gz) (IP:192.168.0.3)

3) Download [Node2](https://downloads3.skycoin.net/skywire-images/node-2-04.tar.gz) (IP:192.168.0.4)

4) Download [Node3](https://downloads3.skycoin.net/skywire-images/node-3-05.tar.gz) (IP:192.168.0.5)

5) Download [Node4](https://downloads3.skycoin.net/skywire-images/node-4-06.tar.gz) (IP:192.168.0.6)

6) Download [Node5](https://downloads3.skycoin.net/skywire-images/node-5-07.tar.gz) (IP:192.168.0.7)

7) Download [Node6](https://downloads3.skycoin.net/skywire-images/node-6-08.tar.gz) (IP:192.168.0.8)

8) Download [Node7](https://downloads3.skycoin.net/skywire-images/node-7-09.tar.gz) (IP:192.168.0.9)

### 수동 IP 설정

`주의: 이 시스템 이미지는 단지 기본적인 스카이와이어 설정만을 포함하고 있으며, IP 설정 등이 필요합니다.`

다운로드[원본 이미지](https://downloads3.skycoin.net/skywire-images/skywire_pure.tar.gz)
