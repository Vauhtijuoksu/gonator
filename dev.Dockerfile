FROM golang:1.15.7-alpine3.12

ENV GO111MODULE=on

WORKDIR /app

COPY . .

RUN go get -u github.com/cosmtrek/air

ENTRYPOINT [ "air" ]