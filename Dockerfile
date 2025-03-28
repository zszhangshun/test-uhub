FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY . .
RUN GOOS=linux GOARCH=arm64 go build -o app .

FROM arm64v8/alpine:latest
WORKDIR /app

COPY --from=builder /app/app .
COPY ./config/* /app/config/
COPY ./static/ /static/
# 确保/app/static目录存在
RUN ln -sfT /static /app/static


CMD ["./app", "-c", "/app/config/config.json"]