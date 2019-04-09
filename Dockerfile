FROM hub.digi-sky.com/base/centos:7.5
COPY ./build/app /data/app
WORKDIR /data
ENTRYPOINT /data/app