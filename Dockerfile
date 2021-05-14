FROM golang:1.14 as builder
COPY . /go/src
WORKDIR /go/src
RUN export GO111MODULE=on && \
 export GOPROXY=https://goproxy.cn && \
 go build -o /go/bin/main /go/src/main.go

FROM registry:2
COPY --from=builder /go/bin/main /usr/main
RUN mkdir /lib64 && ln -s /lib/libc.musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2 && \
 sed -i s@/dl-cdn.alpinelinux.org/@/mirrors.aliyun.com/@g /etc/apk/repositories && \
 apk update && \
 apk add fuse
COPY weed /usr/bin/weed
#ENTRYPOINT ["/usr/main"]
EXPOSE 5000
ENTRYPOINT ["sleep"]
CMD ["infinity"]
