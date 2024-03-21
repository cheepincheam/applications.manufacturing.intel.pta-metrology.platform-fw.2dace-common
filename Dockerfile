FROM --platform=amd64 ubuntu:20.04

# proxy settings & environment variables
ENV http_proxy=proxy-chain.intel.com:911
ENV https_proxy=proxy-chain.intel.com:912
ENV no_proxy=host.docker.internal
ENV DEBIAN_FRONTEND=noninteractive

# setup the apt-get proxy
COPY conf/dev-container/02proxy /etc/apt/apt.conf.d/02proxy

# system configuration for 2DACE system
COPY conf/sys_conf.json /etc/intelatcloud/conf/sys_conf.json

# log configuration for all the main-test code
COPY conf/log_conf.json /etc/intelatcloud/conf/main_log_conf.json

# log configuration for all the microservices
COPY conf/log_conf.json /etc/intelatcloud/conf/log_conf.json

# Update apt package list
RUN apt-get update -y

# Install the barebone tools
RUN apt install curl software-properties-common apt-transport-https ca-certificates gnupg wget -y

RUN wget -O - https://apt.kitware.com/keys/kitware-archive-latest.asc 2>/dev/null | gpg --dearmor - | tee /etc/apt/trusted.gpg.d/kitware.gpg >/dev/null
RUN apt-add-repository 'deb https://apt.kitware.com/ubuntu/ focal main'
RUN apt update
RUN apt install cmake -y
RUN apt install build-essential pkg-config git protobuf-compiler -y

# Install go
RUN wget -c https://dl.google.com/go/go1.20.14.linux-amd64.tar.gz -O - | tar -xz -C /usr/local
ENV PATH="$PATH:/usr/local/go/bin"

ENV GOROOT=/usr/local/go
ENV GOPATH=$HOME/go
ENV GOBIN=$GOPATH/bin
ENV PATH=$PATH:$GOROOT:$GOPATH:$GOBIN

# Protocol Buffer compiler
#RUN export GO111MODULE=on
#RUN go get google.golang.org/protobuf/cmd/protoc-gen-go google.golang.org/grpc/cmd/protoc-gen-go-grpc
RUN export PATH="$PATH:$(go env GOPATH)/bin"
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
RUN go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Kafka client
RUN apt-get install librdkafka-dev --assume-yes

# Docker package dependencies
RUN apt-get install -y apt-transport-https ca-certificates software-properties-common gnupg

# Download and add Docker's official public PGP key
RUN curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg

# Add the `stable` channel's Docker upstream repository.
#
# If you want to live on the edge, you can change "stable" below to "test" or
# "nightly". I highly recommend sticking with stable!
RUN echo "deb [arch=amd64 signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null

# Update the apt package list (for the new apt repo).
RUN apt-get update -y

# Install the latest version of Docker CE.
RUN apt-get install -y docker-ce docker-ce-cli containerd.io
RUN echo "export DOCKER_HOST=tcp://host.docker.internal:2375" >> ~/.bashrc && . ~/.bashrc

RUN update-ca-certificates

# Install rust lang
RUN curl https://sh.rustup.rs -sSf | sh -s -- -y
RUN . $HOME/.cargo/env

# Install etcd-client
RUN apt install etcd-client globus-gridftp-server-progs globus-gass-copy-progs libglobus-gridftp-server-dev libglobus-common-dev libglobus-gssapi-gsi-dev cmake build-essential pkg-config git autotools-dev autoconf -y

EXPOSE 8000