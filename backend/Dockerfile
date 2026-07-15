FROM golang:1.26-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /server ./cmd/server

FROM alpine:3.20
RUN adduser -D -u 10001 app && mkdir -p /app/data && chown app /app/data
WORKDIR /app
COPY --from=build /server /app/server
USER app
ENV DB_PATH=/app/data/cloudcost.db
EXPOSE 8080
ENTRYPOINT ["/app/server"]
