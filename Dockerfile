FROM golang:1.26.1-alpine3.23 AS build

WORKDIR /build
COPY . .
RUN go mod tidy
RUN go build -o /server /build/

FROM alpine:3.23 AS runner

WORKDIR /

RUN addgroup --system --gid 1001 gorun
RUN adduser --system --uid 1001 gorun

COPY --from=build --chown=gorun:gorun /server /server

USER gorun

CMD ["/server"]
