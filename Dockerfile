# Build
FROM golang:1.26-alpine AS build
WORKDIR /src
RUN apk add --no-cache git ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/gotype-ssh ./cmd/gotype-ssh

# Run
FROM alpine:3.21
RUN apk add --no-cache ca-certificates \
	&& adduser -D -H -u 10001 app
WORKDIR /app
COPY --from=build /out/gotype-ssh /app/gotype-ssh
USER app
ENV PORT=2222
EXPOSE 2222
CMD ["./gotype-ssh"]
