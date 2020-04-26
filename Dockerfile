FROM golang:alpine
COPY . /app
WORKDIR /app
ENV GO111MODULE=on
ENV GOPROXY=https://goproxy.cn
RUN go build
EXPOSE 11111
CMD ./bt_crawler
