# mini-microservice/Dockerfile
FROM golang:1.26-alpine AS build
WORKDIR /app 
COPY . .
ARG SERVICE=order-service
WORKDIR /app/${SERVICE}
RUN CGO_ENABLED=0 GOOS=linux go build -o /svc .
FROM alpine:3.20
COPY --from=build /svc /svc 
CMD ["/svc"]
