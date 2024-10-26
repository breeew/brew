FROM amd64/golang:1.22.0-alpine3.18 AS builder

ARG GOPROXY_ENV

ENV GOPROXY=$GOPROXY_ENV

WORKDIR /app
COPY . .
RUN mkdir -p _build
COPY ./cmd/service/etc/config-default.toml /app/_build/etc/config-default.toml
RUN go build -a -ldflags '-extldflags "-static"' -o _build/brew-api ./cmd/


FROM amd64/alpine:3.18
LABEL MAINTAINER <w@ojbk.io>

WORKDIR /app
COPY --from=builder /app/_build/etc /app/etc
COPY --from=builder /app/_build/brew-api /app/brew-api

CMD ["./brew-api", "service", "-c", "./etc/config-default.toml"]