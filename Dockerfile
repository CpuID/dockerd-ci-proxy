FROM golang:1.9.2-alpine3.6

RUN mkdir -p /go/src/app
WORKDIR /go/src/app

COPY . /go/src/app
RUN apk add --no-cache git && \
    go-wrapper download && \
    apk del git
RUN go-wrapper install

CMD ["go-wrapper", "run"]
