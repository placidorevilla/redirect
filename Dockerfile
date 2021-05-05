FROM golang:1.16-alpine AS build
COPY . /go/src
WORKDIR /go/src/cmd/redirect
ENV CGO_ENABLED=0
RUN GOOS=linux GOARCH=amd64 go build -o /build/redirect

FROM scratch AS bin
COPY --from=build /build/redirect /bin/
EXPOSE 80
EXPOSE 8080
VOLUME /etc/redirect

CMD ["/bin/redirect", "-config", "/etc/redirect/config.json", "-ui-addr", "0.0.0.0:8080", "-bind", "0.0.0.0:80"]
