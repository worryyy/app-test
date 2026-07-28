FROM golang:1.25-alpine AS builder
WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o ecampus ./cmd/ecampus \
    && CGO_ENABLED=0 GOOS=linux go build -o ecampus-migrate ./cmd/ecampus-migrate

FROM alpine:3.20
RUN apk --no-cache add ca-certificates tzdata
ENV TZ=Asia/Shanghai
WORKDIR /app

COPY --from=builder /build/ecampus .
COPY --from=builder /build/ecampus-migrate .
COPY --from=builder /build/configs/ecampus ./configs/ecampus
COPY --from=builder /build/db/migrations ./db/migrations

EXPOSE 8080
ENTRYPOINT ["./ecampus"]
