# Build Geth in a stock Go builder container
FROM golang:1.9-alpine as builder

RUN apk add --no-cache make gcc musl-dev linux-headers

ADD . /cryptopepe-worker


RUN apk add --no-cache \
  --repository http://dl-3.alpinelinux.org/alpine/edge/testing \
  gcc g++ make libc6-compat

RUN apk add --no-cache \
  --repository http://dl-3.alpinelinux.org/alpine/edge/testing \
  librsvg-dev glib-dev expat-dev libpng-dev fftw-dev libjpeg-turbo-dev


ARG VIPS_VERSION=8.6.3

ENV VIPS_DIR=/vips
ENV PKG_CONFIG_PATH=${VIPS_DIR}/lib/pkgconfig:$PKG_CONFIG_PATH

RUN apk update && apk add --no-cache openssl ca-certificates && mkdir -p ${GOPATH}/src && \
    wget -O- https://github.com/libvips/libvips/releases/download/v${VIPS_VERSION}/vips-${VIPS_VERSION}.tar.gz | tar xzC /tmp

RUN cd /tmp/vips-${VIPS_VERSION} && \
    ./configure \
        --disable-static \
        --disable-dependency-tracking \
        --without-python \
        --prefix=${VIPS_DIR} && \
    make && \
    make install


RUN cd /cryptopepe-worker && build/env.sh go build -v -o build/cryptopepe-worker .

# Pull Geth into a second stage deploy alpine container
FROM alpine:latest

RUN apk add --no-cache \
  --repository http://dl-3.alpinelinux.org/alpine/edge/testing \
  fftw libpng libjpeg-turbo librsvg expat glib libgsf

# Add the vipslib we compiled from source in the builder
COPY --from=builder /vips/lib/ /usr/local/lib

# Define that we want this docker build argument
ARG GOOGLE_APPLICATION_CREDENTIALS
# Retrieve the build argument, insert it into the environment
ENV GOOGLE_APPLICATION_CREDENTIALS=$GOOGLE_APPLICATION_CREDENTIALS

RUN apk add --no-cache ca-certificates
COPY --from=builder /cryptopepe-worker/build/cryptopepe-worker /usr/local/bin/

# Copy builder files
COPY --from=builder /cryptopepe-worker/vendor/cryptopepe.io/cryptopepe-svg/builder/tmpl /app/tmpl
COPY --from=builder /cryptopepe-worker/vendor/cryptopepe.io/cryptopepe-svg/builder/builder.tmpl /app/builder.tmpl

# Copy bio builder file
COPY --from=builder /cryptopepe-worker/bio-gen/bio_config.yml /app/bio_config.yml

# Write credentials file (read from env, decode, tee into file)
RUN echo -n "$GOOGLE_APPLICATION_CREDENTIALS" | base64 -d | tee /app/secret-key-google.json

WORKDIR /app
ENTRYPOINT ["cryptopepe-worker"]
