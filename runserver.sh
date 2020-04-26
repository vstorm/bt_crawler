#!/bin/bash


docker image rm bt_crawler
docker image build -t bt_crawler .
docker run --rm -p 11112:11111 bt_crawler
