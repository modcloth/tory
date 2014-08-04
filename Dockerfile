FROM stackbrew/ubuntu:14.04
MAINTAINER Dan Buch <d.buch@modcloth.com>

ENV DEBIAN_FRONTEND noninteractive
ENV PATH /gopath/bin:/usr/local/go/bin:/usr/local/bin:/usr/local/sbin:/usr/bin:/usr/sbin:/bin:/sbin
ENV GOROOT /usr/local/go
ENV GOPATH /gopath

RUN apt-get update -y
RUN apt-get install -y --no-install-suggests \
    curl make git build-essential mercurial
RUN curl -sL http://golang.org/dl/go1.3.linux-amd64.tar.gz | \
    tar -C /usr/local -xzf -
RUN mkdir -p /gopath/src/github.com/modcloth-labs

ADD . /gopath/src/github.com/modcloth/tory

WORKDIR /gopath/src/github.com/modcloth/tory

RUN go get github.com/tools/godep
RUN make build
RUN apt-get purge -y curl make git build-essential mercurial

CMD ["tory", "serve"]
