FROM hub.digi-sky.com/base/golang:1.12.9

WORKDIR /usr/src/kapp-agent
COPY . .

ENV http_proxy 192.168.2.49:8080
ENV https_proxy 192.168.2.49:8080
ENV GOOS linux

RUN go build -o /usr/src/kapp-agent/build/app main.go

FROM hub.digi-sky.com/base/centos:7.5
COPY --from=0 /usr/src/kapp-agent/build/app /data/app
WORKDIR /data
ENTRYPOINT /data/app