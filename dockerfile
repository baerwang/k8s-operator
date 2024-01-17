FROM golang:1.21 as build

WORKDIR /app

COPY . .

ENV GOPROXY="https://goproxy.cn,direct"

RUN CGO_ENABLED=0 go build -ldflags "-s -w" -trimpath -o ingress-manage

FROM alpine

WORKDIR /app

COPY --from=build /app/ingress-manage .

CMD ["./ingress-manage"]