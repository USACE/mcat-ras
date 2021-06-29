FROM osgeo/gdal:alpine-normal-3.2.1

COPY --from=golang:1.16.5-alpine3.13 /usr/local/go/ /usr/local/go/

RUN apk add --no-cache \
    pkgconfig \
    gcc \
    libc-dev \
    git

ENV GOROOT=/usr/local/go
ENV GOPATH=/go
ENV PATH=$PATH:$GOROOT/bin:$GOPATH/bin

RUN go get github.com/githubnemo/CompileDaemon

COPY ./ /app
WORKDIR /app
