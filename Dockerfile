FROM golang:latest AS builder

WORKDIR /app

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .

ENV CGO_ENABLED 0

RUN go build -v -o /api .

FROM alpine:latest AS runner

WORKDIR /

RUN addgroup -g 1001 -S golang
RUN adduser -S golang -u 1001

COPY --from=builder /api /api

USER golang

EXPOSE 9000

ENTRYPOINT ["/api"]