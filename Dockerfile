# Stage 1: The "builder" stage, where we compile the Go app
FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o /main .

# Stage 2: The final, lightweight production stage
FROM alpine:latest

COPY --from=builder /main .

CMD ["./main"]