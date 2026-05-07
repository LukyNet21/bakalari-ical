FROM golang:1.26.1-alpine3.23 AS build

WORKDIR /build
COPY . .
RUN go mod tidy
RUN go build -o /server /build/cmd/web
RUN go build -o /newcal /build/cmd/newcal
RUN go build -o /newkey /build/cmd/newkey

FROM alpine:3.23 AS runner

WORKDIR /

RUN addgroup --system --gid 1001 gorun
RUN adduser --system --uid 1001 gorun

COPY --from=build --chown=gorun:gorun /server /server
COPY --from=build --chown=gorun:gorun /newcal /newcal
COPY --from=build --chown=gorun:gorun /newkey /newkey

USER gorun

CMD ["/server"]
