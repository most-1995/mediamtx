FROM --platform=linux/amd64 golang:1.22-alpine as builder

WORKDIR /build

COPY . .

RUN go build -o ./mediamtx .

# FROM --platform=linux/amd64 alpine:3.17

# COPY --from=builder /build/mediamtx .
# COPY --from=builder /build/mediamtx.yml ./mediamtx.yml

EXPOSE 8000
EXPOSE 9000
EXPOSE 8554
EXPOSE 1934
EXPOSE 8888
EXPOSE 8889
EXPOSE 9997
EXPOSE 9996
EXPOSE 8890/udp
EXPOSE 8189/udp


CMD ["./mediamtx"]

