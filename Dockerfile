# Go build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o server ./cmd/server/

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add libvirt-daemon qemu-img

WORKDIR /app

COPY --from=builder /app/server .
COPY --from=builder /app/config.yaml .
COPY --from=builder /app/migrations ./migrations
COPY --from=builder /app/templates ./templates
COPY --from=builder /app/uploads ./uploads

RUN mkdir /app/logs && \
    chown -R nobody:nogroup /app

USER nobody

EXPOSE 8080 8081

CMD ["./server"]
