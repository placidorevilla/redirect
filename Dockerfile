FROM golang:alpine as build
RUN apk add --no-cache git
WORKDIR /go/src/github.com/reddec/redirect
COPY . .
RUN go get -d -v ./...
RUN go install -v ./...

FROM alpine:latest as production
COPY --from=build /go/bin/redirect /bin/
EXPOSE 10100
EXPOSE 10101
VOLUME /etc/redirect

CMD ["/bin/redirect", "-config", "/etc/redirect/config.json", "-ui-addr", "0.0.0.0:10101", "-bind", "0.0.0.0:10100"]
