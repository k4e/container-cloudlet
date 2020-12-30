FROM alpine:3.10

RUN apk --update --no-cache add openjdk8 openssh rsync iptables
RUN apk --update --no-cache add -X http://dl-cdn.alpinelinux.org/alpine/edge/main libnet
RUN apk --update --no-cache add -X http://dl-cdn.alpinelinux.org/alpine/edge/testing criu
COPY . .
RUN chmod +x /lmsupport.sh

WORKDIR /
CMD ["/bin/sh", "-c", "java -jar -Xmx1536m /app-sample/target/container-cloudlet-app-sample-0.0.1-jar-with-dependencies.jar & /lmsupport.sh $!"]
