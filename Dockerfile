FROM ghcr.io/osgeo/gdal:alpine-normal-3.8.3 as build

COPY --from=golang:1.21.3-alpine /usr/local/go/ /usr/local/go/

RUN apk add --no-cache \
    pkgconfig \
    gcc \
    libc-dev \
    git

ENV GOROOT=/usr/local/go
ENV GOPATH=/go
ENV PATH=$PATH:$GOROOT/bin:$GOPATH/bin


COPY ./ /app
WORKDIR /app

RUN go build main.go

# ENTRYPOINT CompileDaemon --build="go build main.go" --command=./main


FROM ghcr.io/osgeo/gdal:alpine-normal-3.8.3 as local

COPY --from=build /app/main /app/main

RUN wget https://github.com/HydrologicEngineeringCenter/hec-downloads/releases/download/1.0.23/HEC-RAS_62_Example_Projects.zip

RUN unzip HEC-RAS_62_Example_Projects.zip
RUN mkdir mcat-ras-testing
RUN mv /Example_Projects/ /mcat-ras-testing
RUN rm HEC-RAS_62_Example_Projects.zip

ENTRYPOINT /app/main


FROM ghcr.io/osgeo/gdal:alpine-normal-3.8.3 as prod
COPY --from=build /app/main /main
ENTRYPOINT /main