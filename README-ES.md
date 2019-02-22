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
* Guía del desarrollador
  * [Manager API](docs/api/ManagerAPI.md)
  * [Node API](docs/api/NodeAPI.md)

### Requerimientos

* golang 1.9+

  https://golang.org/dl/

* git

* setup $GOPATH env (for example: /go)
  https://github.com/golang/go/wiki/SettingGOPATH

## Instalación

Primero dale un vistazo al [manual de integración con script](static/script/README.md) para conocer algunos detalles y hechos que le serán de utilidad para entender como se integra skywire con el sistema Unix base, muy importante allí es la parte de las políticas de red.

Se lees este documento te darás cuenta que si usas un segmento de IP diferente al sugerido tendrás que cambiar algunas cosas, nosotros te diremos cuales a lo largo de este documento.

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

#### Configurar la IP del administrador

Si estás usando el segmento de IP sugerido no necesitas hacer nada aquí, sigue al paso siguiente.

Si usas un segmento IP diferente del sugerido deberás modificar el fichero ```static/script/skywire.defaults``` particularmente el valor de la variable llamada ```MANAGER_IP``` que debe apuntar al IP del administrador, en este caso debes modificarla al valor del IP de tu pc administrador.

Solo por precaución y luego de modificar este fichero debes fijarte si existe en tu sistema el fichero ```/etc/default/skywire```, si existe por favor eliminalo. Este será actualizado una vez que ejecutes skywire.

## Ejecutar Skywire

### Sistemas Unix

#### Ejecutar administrador skywire

Abra una ventana de comandos en una PC que será administrador y siga las instrucciones de instalación, entonces para ejecutar el administrador escriba:

```
${GOPATH}/src/github.com/skycoin/skywire/static/script/manager_start
```

`tip: al ejecutar el administrador mediante este script estará ejecutando a la vez un nodo, por lo que no necesita ejecutar un nodo en la misma PC.`

#### Ejecutar el nodo Skywire

Abra una ventana de comandos en una PC que será solo un nodo y siga las instrucciones de instalación, entonces para ejecutar un nodo escriba:

```
${GOPATH}/src/github.com/skycoin/skywire/static/script/node_start
```

#### Detener el administrador y nodo de Skywire

Si inició el administrador o el nodo por la vía anteriormente descrita puedes deternerlo en cada PC con un simple comando en una ventana de comandos:

```
${GOPATH}/src/github.com/skycoin/skywire/static/script/stop
```

Esto verifica el pid de los procesos creados con los scripts de inicio (manager_start y node_start) y los detendrá. Si ejecutaste tanto el administrador como el nodo llamando a los binarios epecíficos y no usando el script sugerido, puedes matar los procesos usando estos comandos:

```
killall node
killall manager
```

##### Instalando el administrador y los nodos como servicios usando systemd

Si usas cualquier sistema operativo moderno linux (lanzado despues de 2017) es seguro que estés usando systemd como gestor de inicio, skywire tiene los ficheros de unit listos para instalar en systemd como servicios.

Por favor ten en cuenta que una instancia administradora inicia su propio nodo en la misma Pc, por lo que debes tener solo una PC como administrador en una red y el resto serán nodos.

###### Instalando los fichero unit de systemd para el administrador

```
cp ${GOPATH}/src/github.com/skycoin/skywire/static/script/upgrade/data/skywire-manager.service /etc/systemd/system/
systemctl enable skywire-manager
systemctl start skywire-manager
```

###### Instalando los fichero unit de systemd para los nodos

```
cp ${GOPATH}/src/github.com/skycoin/skywire/static/script/upgrade/data/skywire-node.service /etc/systemd/system/
systemctl enable skywire-node
systemctl start skywire-node
```

De ahora en adelante puedes manejar (iniciar/parar) los servicios de skywire con systemd, algunos comandos útiles:

```
# para los nodos
systemctl *start* skywire-node
systemctl *stop* skywire-node
systemctl *status* skywire-node
# para el administrador
systemctl *start* skywire-manager
systemctl *stop* skywire-manager
systemctl *status* skywire-manager
```

## Abrir la vista del administrador Skywire
Abrir [http://localhost:8000](http://localhost:8000).
La contraseña de inicio de sesión predeterminada para el administrador Skywire es **1234**.

### Conectarse al nodo
1) Conectarse al nodo —— Buscar servicios —— Conectar

2) Conectarse al nodo —— Ingrese la clave para el nodo y la aplicación —— Conectar

De la primera manera, puede buscar nodos en todo el mundo y seleccionar los nodos a los que desea conectarse; La segunda forma es conectarse al nodo especificado.

#### Usar la aplicación Skywire
Después del inicio normal predeterminado, la aplicación mostrará "**puerto disponible**" (por ejemplo, 9443) después de una conexión exitosa.

#### User el navegador Firefox

#### Instalar el estándar FoxyProxy
Abra el navegador Firefox, ingrese la barra de direcciones "https://addons.mozilla.org/zh-CN/firefox/addon/foxyproxy-standard/", haga clic en el botón "agregar a Firefox" para seguir las instrucciones para instalar.

#### Configurando el FoxyProxy estándar 
Una vez completada la instalación, navegue por la barra de direcciones de Firefox y escriba: "complementos" en la página de complementos, encuentre FoxyProxy "Estándar" y haga clic en las preferencias en la página de configuración <br> seleccione "Usar proxies habilitados por patrones y prioridad" habilitado FoxyProxy <br>
Haga clic en "Agregar" para agregar la configuración,
```
Proxy Type: SOCKS5
IP address, DNS name, server name: 127.0.0.1
Port: 9443
```
Y luego, haga clic en "Guardar"

### Herramienta SSH

#### SSH
Después de abrir este servicio, se generará la clave pública de la aplicación. Basado en la clave pública del nodo y la clave pública, el nodo se puede administrar de forma remota en cualquier máquina que ejecute Skywire.

`Nota: no abra SSH a voluntad y muestre la Clave de nodo y la Clave de la aplicación a extraños.`

#### SSH Client
Ingrese la clave del nodo y la clave de la aplicación. Después de que la conexión sea exitosa, el Puerto (Puerto) se mostrará debajo del botón, por ejemplo, 30001, y finalmente, usar cualquier herramienta de conexión SSH remota.

## Docker

```
docker build -t skycoin/skywire .
```

### Arranque el administrador

```
docker run -ti --rm \
  --name=skywire-manager \
  -p 5998:5998 \
  -p 8000:8000 \
  skycoin/skywire
```

**Nota:**
Las imágenes de Skywire para ARM v5 y v7 están basadas en `busybox`. Los contenedores para las arquitecturas v6 y v8 corren sobre `alpine`.

Abrir [http://localhost:8000](http://localhost:8000).
La contraseña de inicio de sesión predeterminada para Skywire Manager es ** 1234 **.

### Inicie un nodo y conéctelo al administrador

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
      -web-port :6001 \
      -discovery-address discovery.skycoin.net:5999-034b1cd4ebad163e457fb805b3ba43779958bba49f2c5e1e8b062482904bacdb68
```

### Docker Compose

```
docker-compose up
```

Abrir [http://localhost:8000](http://localhost:8000).

Note: puedes agregar más nodos editando el fichero [docker-compose.yml](https://github.com/skycoin/skywire/blob/master/docker-compose.yml)

## Descargar imágenes del sistema

<a name="images"></a>

Note: estas imágenes solo se pueden ejecutar en [Orange Pi Prime](http://www.orangepi.cn/OrangePiPrime/index_cn.html).

### Imágenes de sistema preestablecidas IP

La contraseña predeterminada es 'samos'.

Ejecuta esto **una vez si estás usando las imágenes oficiales** para actualizar a la última versión del código:

```
cd $GOPATH/src/github.com/skycoin/skywire
git remote set-url origin https://github.com/skycoin/skywire.git
git reset --hard
git clean -f -d
git pull origin master
go install -v ./...
```

### Importante:

Estas imágenes bases (todas) tienen un fallo conocido, por favor [lee aquí](https://github.com/skycoin/skywire/issues/80) una vez que hallas actualizado el código para saber como solucionarlo hasta tanto se actualizan las imágenes.

El paquete de imagen del sistema administrador contiene un administrador Skywire y un nodo Skywire, otro paquete de imagen del sistema Nodo solo inicia un nodo.

1) Descargar [Administrador](https://downloads3.skycoin.net/skywire-images/manager.tar.gz) (IP:192.168.0.2)

2) Descargar [Nodo1](https://downloads3.skycoin.net/skywire-images/node-1-03.tar.gz) (IP:192.168.0.3)

3) Descargar [Nodo2](https://downloads3.skycoin.net/skywire-images/node-2-04.tar.gz) (IP:192.168.0.4)

4) Descargar [Nodo3](https://downloads3.skycoin.net/skywire-images/node-3-05.tar.gz) (IP:192.168.0.5)

5) Descargar [Nodo4](https://downloads3.skycoin.net/skywire-images/node-4-06.tar.gz) (IP:192.168.0.6)

6) Descargar [Nodo5](https://downloads3.skycoin.net/skywire-images/node-5-07.tar.gz) (IP:192.168.0.7)

7) Descargar [Nodo6](https://downloads3.skycoin.net/skywire-images/node-6-08.tar.gz) (IP:192.168.0.8)

8) Descargar [Nodo7](https://downloads3.skycoin.net/skywire-images/node-7-09.tar.gz) (IP:192.168.0.9)

### Establecer manualmente la imagen del sistema de IP

`Nota: Esta imagen del sistema solo contiene el entorno básico de Skywire, y necesita configurar IP, etc..`

Descargar [Imagen Pura](https://downloads3.skycoin.net/skywire-images/skywire_pure.tar.gz)

## Construyendo las imágenes de Orange Pi usted mismo

Las imagenes estan en https://github.com/skycoin/Orange-Pi-H5

Las instrucciones para construir las imagenes estan en https://github.com/skycoin/Orange-Pi-H5/wiki/How-to-build-the-images
