FROM golang:1.18-alpine as build

WORKDIR /build
COPY . /build

RUN apk add --no-cache tzdata ca-certificates
RUN CGO_ENABLED=0 go build -mod=vendor -buildvcs=false -o cal

FROM scratch

COPY --from=build /build/cal /usr/bin/cal
COPY --from=build /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

ENTRYPOINT ["/usr/bin/cal"]
