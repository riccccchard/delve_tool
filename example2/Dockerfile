FROM golang:1.14 as builder

WORKDIR /build

COPY . .

ENV GO111MODULE=on \
        CGO_ENABLED=0 \
        GOOS=linux \
        GOARCH=amd64

RUN go build -o httpxorm

WORKDIR /dist

RUN cp /build/httpxorm .

EXPOSE 9101

CMD ["/dist/httpxorm"]

