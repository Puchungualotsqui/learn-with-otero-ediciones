# ---- Stage 1: build Go binary ----
FROM golang:1.25.1-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o main .


# ---- Stage 2: final image ----
FROM alpine:latest
WORKDIR /app
RUN apk add --no-cache tzdata
COPY --from=builder /app/main .
COPY static ./static

# copy environment file too
COPY .venv ./.venv

# only copy final compiled tailwind file
COPY static/css/output.css ./static/css/output.css

EXPOSE 3000
CMD ["./main"]
