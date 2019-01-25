FROM hub.digi-sky.com/base/centos:7.5
COPY ./app /data/app
WORKDIR /data
ENTRYPOINT /data/app