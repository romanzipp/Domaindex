FROM node:24-alpine AS css

WORKDIR /build

COPY package.json ./

RUN npm install

COPY tailwind.src.css ./
COPY assets/templates ./assets/templates

RUN npm run build:css

# -----------------------------------
FROM golang:1.26-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /build

COPY go.mod go.sum ./

RUN go mod download

COPY . .
COPY --from=css /build/assets/static/css/tailwind.css ./assets/static/css/tailwind.css

ARG VERSION
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-w -s -X main.version=${VERSION}" -o /domaindex ./cmd/server

# -----------------------------------
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /domaindex /app/domaindex

EXPOSE 8080
VOLUME ["/app/data"]
ENTRYPOINT ["/app/domaindex"]
