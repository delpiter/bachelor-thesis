FROM golang:1.26.4-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/api-real-time-diagnosis ./cmd/api

FROM debian:bookworm-slim
COPY --from=build /out/api-real-time-diagnosis /usr/local/bin/api-real-time-diagnosis
RUN setcap cap_net_raw+ep /usr/local/bin/api-real-time-diagnosis
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/api-real-time-diagnosis"]