FROM alpine:latest

COPY . .
RUN apk --update add openjdk8 maven && \
    cd app-sample && \
    mvn clean package

WORKDIR /app-sample
CMD ["java", "-jar", "target/container-cloudlet-app-sample-0.0.1.jar"]
