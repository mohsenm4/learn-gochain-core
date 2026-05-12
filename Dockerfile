FROM golang:1.24-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/node ./cmd/node

FROM alpine:3.20
WORKDIR /app
COPY --from=builder /out/node /app/node
ENTRYPOINT ["/app/node"]
