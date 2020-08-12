FROM golang:latest
COPY . /app
WORKDIR /app
ENV GO111MODULE=on \
    GOPROXY=https://goproxy.cn
RUN go build
EXPOSE 11111
CMD ./bt_crawler
