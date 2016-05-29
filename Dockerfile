# Dockerfile for cmonit.
# Workdir is set to $GOPATH=/go.
# config file can be mapped into the /cmonit volume

FROM golang:1.6
MAINTAINER Baohua Yang <yeasy.github.io>
ENV TZ Asia/Shanghai

VOLUME /cmonit

WORKDIR /cmonit

#RUN cd /tmp \
#        && git clone --single-branch --depth 1 https://github.com/yeasy/cmonit.git \
#        && cd cmonit \
#        && make install \
#        && cp /tmp/cmonit/cmonit.yaml /cmonit/

RUN go get github.com/yeasy/cmonit \
        && cp $GOPATH/src/github.com/yeasy/cmonit/cmonit.yaml /cmonit/ \
        && cp $GOPATH/bin/cmonit /cmonit/

# use this in development
ENTRYPOINT ["sh", "-c", "cmonit", "start"]
