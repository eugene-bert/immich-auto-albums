FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /immich-auto-albums .

FROM alpine:3.24
RUN apk add --no-cache ca-certificates \
    && adduser -D -u 1000 -H appuser \
    && mkdir -p /data \
    && chown -R appuser:appuser /data
COPY --from=build /immich-auto-albums /immich-auto-albums
ENV DB_PATH=/data/rules.db
USER appuser
ENTRYPOINT ["/immich-auto-albums"]
