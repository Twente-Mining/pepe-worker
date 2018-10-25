# Build Geth in a stock Go builder container
FROM golang:1.9-alpine as builder

RUN apk add --no-cache make gcc musl-dev linux-headers

ADD . /cryptopepe-worker
RUN cd /cryptopepe-worker && build/env.sh go build -v -o build/cryptopepe-worker .

# Pull Geth into a second stage deploy alpine container
FROM alpine:latest

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
