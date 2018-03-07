![logo de skywire](https://user-images.githubusercontent.com/26845312/32426764-3495e3d8-c282-11e7-8fe8-8e60e90cb906.png)

# Skywire

Aquí está nuestro [Blog](https://blog.skycoin.net/tags/skywire/) sobre Skywire.

Skywire todavía está bajo fuerte desarrollo.


![2018-01-21 10 44 06](https://user-images.githubusercontent.com/1639632/35190261-1ce870e6-fe98-11e7-8018-05f3c10f699a.png)

## Tabla de contenido
* [Requerimientos](#requerimientos)
* [Instalación](#instalación)
* [Ejecutar Skywire](#ejecutar-skywire)
* [Docker](#docker)

### Requerimientos

* golang 1.9+

  https://golang.org/dl/

* git

* setup $GOPATH env (for example: /go)
  https://github.com/golang/go/wiki/SettingGOPATH

## Instalación
### Sistemas Unix

```
mkdir -p $GOPATH/src/github.com/skycoin
cd $GOPATH/src/github.com/skycoin
git clone https://github.com/skycoin/skywire.git
```

Construya los binarios para skywire
```
cd $GOPATH/src/github.com/skycoin/skywire/cmd
go install ./...
```

## Ejecutar Skywire

### Sistemas Unix
```
cd $GOPATH/bin
./manager -web-dir ${GOPATH}/src/github.com/skycoin/skywire/static/skywire-manager
```

Abra una nueva ventana de comando

```
cd $GOPATH/bin
./node -connect-manager -manager-address :5998 -manager-web :8000 -discovery-address messenger.skycoin.net:5999-028667f86c17f1b4120c5bf1e58f276cbc1110a60e80b7dc8bf291c6bec9970e74 -address :5000 -web-port :6001
```
Use el navegador para abrir http://127.0.0.1:8000

## Docker

```
docker build -t skycoin/skywire .
```

### Iniciar el Manager

```
docker run -ti --rm \
  --name=skywire-manager \
  -p 5998:5998 \
  -p 8000:8000 \
  skycoin/skywire
```

Abrir [http://localhost:8000](http://localhost:8000).
La contraseña de inicio de sesión predeterminada para Skywire Manager es **1234**.

### Iniciar un nodo y conéctelo a el manager

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

Abrir [http://localhost:8000](http://localhost:8000).
