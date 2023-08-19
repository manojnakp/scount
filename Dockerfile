# syntax=docker/dockerfile:1

FROM golang:1.21-alpine3.18 AS build-stage
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
RUN go build -o /scount-api

FROM build-stage AS test-stage
RUN go test -v ./...

FROM alpine:3.18
WORKDIR /
COPY --from=build-stage /scount-api /bin/scount-api
EXPOSE 8080
USER 1001
CMD ["/bin/scount-api"]
