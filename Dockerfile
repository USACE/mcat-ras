FROM osgeo/gdal:alpine-normal-3.2.1 as build

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

RUN go build main.go

ENTRYPOINT CompileDaemon --build="go build main.go" --command=./main

FROM osgeo/gdal:alpine-normal-3.2.1 as prod

COPY --from=build /app/main /app/main

ENTRYPOINT /app/main