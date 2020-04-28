FROM golang:1.13 as build
WORKDIR /go/src/github.com/codedropau/kube-scheduler-ratelimit
ADD . /go/src/github.com/codedropau/kube-scheduler-ratelimit
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o bin/kube-scheduler-ratelimit github.com/codedropau/kube-scheduler-ratelimit

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=build /go/src/github.com/codedropau/kube-scheduler-ratelimit/bin/kube-scheduler-ratelimit /usr/local/bin/kube-scheduler-ratelimit
CMD ["kube-scheduler-ratelimit"]
