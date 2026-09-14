FROM golang:1.27-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
COPY docs ./docs
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/chess-server ./cmd/server

FROM alpine:3.24
RUN addgroup -S chess && adduser -S -G chess chess
COPY --from=build /out/chess-server /usr/local/bin/chess-server
USER chess
ENV ADDR=:8080
EXPOSE 8080
ENTRYPOINT ["chess-server"]
