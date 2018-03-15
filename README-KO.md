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
```
cd $GOPATH/bin
./manager -web-dir ${GOPATH}/src/github.com/skycoin/skywire/static/skywire-manager
```

새 명령창 오픈

```
cd $GOPATH/bin
./node -connect-manager -manager-address :5998 -manager-web :8000 -discovery-address messenger.skycoin.net:5999-028667f86c17f1b4120c5bf1e58f276cbc1110a60e80b7dc8bf291c6bec9970e74 -address :5000 -web-port :6001
```
브라우저로 http://127.0.0.1:8000 오픈

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
