FROM golang:1.22-alpine

RUN apk update && apk add --no-cache git && apk add --no-cach bash && apk add build-base

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY . .

RUN go build -o /Task main.go

EXPOSE 8000

CMD [ "/Task" ]