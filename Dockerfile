# Dockerfile for cmonit.
# Workdir is set to $GOPATH=/go.
# config file can be mapped into the /cmonit volume

FROM golang:1.6
MAINTAINER Baohua Yang <yeasy.github.io>
ENV TZ Asia/Shanghai

RUN go get github.com/yeasy/cmonit

RUN ln -s /$GOPATH/src/github.com/yeasy/cmonit /cmonit

VOLUME /$GOPATH/src/github.com/yeasy/cmonit

WORKDIR /$GOPATH/src/github.com/yeasy/cmonit

# use this in development
ENTRYPOINT ["sh", "-c", "cmonit"]
