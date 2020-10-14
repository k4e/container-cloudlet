FROM alpine:3.10

RUN apk --update --no-cache add openjdk8 maven openssh rsync
RUN apk --update --no-cache add -X http://dl-cdn.alpinelinux.org/alpine/edge/main libnet
RUN apk --update --no-cache add -X http://dl-cdn.alpinelinux.org/alpine/edge/testing criu
COPY . .
RUN mkdir -p /mnt/tmpfs
RUN cd app-sample && mvn clean package

WORKDIR /app-sample
CMD ["/bin/sh", "-c", "java -jar target/container-cloudlet-app-sample-0.0.1-jar-with-dependencies.jar; sleep 999999"]
