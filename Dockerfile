FROM golang:1.13-alpine AS builder

WORKDIR /opt
COPY . /opt/

RUN CGO_ENABLED=0 go build -mod=vendor -a -installsuffix cgo -ldflags '-extldflags "-static"' -o iot2alexa .
# --------------------------------------
# Alpine needed to include proper certs
FROM alpine:3.7
WORKDIR /
COPY --from=builder /opt/iot2alexa /

CMD ["/iot2alexa"]
