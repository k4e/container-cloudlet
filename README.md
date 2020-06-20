# Container-based Cloudlet

## Setup

### Server

1. Install Go
    ```
    $ sudo add-apt-repository ppa:longsleep/golang-backports
    $ sudo apt update
    $ sudo apt install -y golang-go
    ```

1. Install Docker
    ```
    $ sudo apt update
    $ sudo apt install -y apt-transport-https ca-certificates curl gnupg-agent software-properties-common
    $ curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo apt-key add -
    $ sudo add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable"
    $ sudo apt update
    $ sudo apt install -y docker-ce
    $ sudo systemctl status docker
    ```

1. Install kubectl
    ```
    $ sudo apt-get update && sudo apt-get install -y apt-transport-https
    $ curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo apt-key add -
    $ echo "deb https://apt.kubernetes.io/ kubernetes-xenial main" | sudo tee -a /etc/apt/sources.list.d/kubernetes.list
    $ sudo apt update
    $ sudo apt install -y kubectl
    ```

1. Install minikube
    ```
    $ curl -Lo minikube https://storage.googleapis.com/minikube/releases/latest/minikube-linux-amd64 && chmod +x minikube
    $ sudo mkdir -p /usr/local/bin/
    $ sudo install minikube /usr/local/bin/
    $ sudo apt install -y conntrack
    ```


1. Add below to `.bashrc` and `source .bashrc`
    ```
    export GOPATH=$HOME/go
    export GO111MODULE=on
    ```

1. Run `sudo docker login`

1. Run `sudo minikube start --vm-driver none`

1. Run `sudo chown -R $USER $HOME/.kube $HOME/.minikube`

1. Run:
    ```
    $ sudo kubectl create secret generic regcred --from-file=.dockerconfigjson=$HOME/.docker/config.json --type=kubernetes.io/dockerconfigjson
    ```

### Client

1. Install OpenJDK and Maven
    ```
    $ sudo apt install -y openjdk-11-jdk maven
    ```

1. Compile
    ```
    $ cd container-cloudlet/client
    $ mvn clean compile
    ```

## Usage

### Server

1. `cd container-cloudlet/server`

1. Run `go run .`

### Client

- Create a sample app on server
    ```
    $ mvn exec:java -Dexec.args="$(sudo minikube ip) 9999 create"
    ```

- Send a message to the sample app on server
    ```
    $ mvn exec:java -Dexec.args="$(sudo minikube ip) 30088 send hello?"
    ```

- Delete the sample app on server
    ```
    $ mvn exec:java -Dexec.args="$(sudo minikube ip) 9999 delete"
    ```