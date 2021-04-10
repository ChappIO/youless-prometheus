FROM golang:1.16 as build

WORKDIR /go/src
ADD main.go .
ENV CGO_ENABLED=0
RUN go build -o /go/bin/youless-prometheus main.go

FROM scratch
WORKDIR /
COPY --from=build /go/bin/youless-prometheus /
ENTRYPOINT ["/youless-prometheus"]
