FROM go:1.24-alpine AS go
RUN apk update && apk add --no-cache -y git curl
WORKDIR /down.lsp
COPY . .
RUN go build -o down.lsp ./main.go

FROM alpine:latest as monolithic
WORKDIR /down
RUN apk add --no-cache -y tzdata git curl
ENV TZ="UTC"
COPY --from=go /down.lsp/down.lsp /down/down.lsp

EXPOSE 1880
RUN mkdir -p /var/down
VOLUME /var/down

ENV DOWN_HOST=127.0.0.1
ENV DOWN_PORT=1880
ENV DOWN_MODE=production

ENTRYPOINT ["/down/down.lsp", "lsp"]



