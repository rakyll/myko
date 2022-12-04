FROM golang:1.19-buster as builder

WORKDIR /build

COPY . .
RUN GOOS=linux CGO_ENABLED=0 go build -o /myko ./cmd/myko

FROM ubuntu:latest

RUN set -x && apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y \
    protobuf-compiler && \
    rm -rf /var/lib/apt/lists/*

COPY --from=builder /myko /

RUN chmod +x /myko

ENTRYPOINT [ "/myko" ]
