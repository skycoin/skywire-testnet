# Skywire scalable multi-node docker containers
The associated `docker-compose.yml` file can be used setup and run a single instance of the Skywire Manager (currently the old UI), and scale up any number of subordinate Nodes using features of `docker-compose`. 

By default (without specifying a scale flag) you will get a Single Manager and a Single Node. This has been (lightly) tested  on MacOS and Linux.

## Prerequisits
You must have both `Docker` and `docker-compose` installed on the target machine to use this process

## Steps
The following provide details for use:
1. Copy the `docker-compose.yml` file to a new empty folder on a machine which already has Docker installed.
2. Change into the new folder where the `docker-compose.yml` file resides.
3. To start a single instance of both Manager and Node (in interactive mode on the terminal) run the following command:
```sh
docker-compose up
```
4. Test that the Manager is running and that the node has connected to it. The Manager UI will be avaiable at http://127.0.0.1:8000 on the machine where the Docker containers are now running. If everything worked, you should be able to access the manager using the default password (you will be asked to change this), and there should be a single Node shown.
5. To shutdown the containers (which are running interactivly), press `CTRL + C`
6. Test stating the Manager with 5 Nodes using the following command:
```sh
docker-compose up --scale swnode=5
```

## Notes and Tips

### Stop the Manager and Node(s)
```sh
docker-compose down
```

### Start Manager and X Nodes
```sh
docker-compose up --scale swnode=X
```

### Get IP address for a running container
```sh
docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' container_or_image_name_here
```

### Monitor Docker stats (Memory, CPU usage,etc)
```sh
docker stats
```