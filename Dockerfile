# -------- Build stage --------
FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
go build -o api ./cmd/server

# -------- Run stage --------
FROM alpine:3.19

WORKDIR /app

COPY --from=builder /app/api .

EXPOSE 8080

CMD [ "./api" ]
