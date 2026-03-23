FROM golang:1.25-alpine AS builder
WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o ecampus-crm ./cmd/ecampus-crm

FROM alpine:3.20
RUN apk --no-cache add ca-certificates tzdata
ENV TZ=Asia/Shanghai
WORKDIR /app

COPY --from=builder /build/ecampus-crm .
COPY --from=builder /build/configs/ecampus-crm ./configs/ecampus-crm

EXPOSE 8081
ENTRYPOINT ["./ecampus-crm"]
