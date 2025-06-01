FROM golang:1.22-alpine AS builder
RUN apk add --no-cache gcc musl-dev sqlite-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o video-processor .

FROM alpine:latest
RUN apk add --no-cache sqlite-libs
RUN apk add --no-cache ffmpeg


WORKDIR /app

COPY --from=builder /app/video-processor .

RUN mkdir /app/data

ENTRYPOINT ["./video-processor"]

# Add a VOLUME for the SQLite database file
VOLUME ["/app/data"]
