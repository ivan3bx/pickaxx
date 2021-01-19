FROM golang:1.15

WORKDIR /root/build

RUN GO111MODULE=off go get -u github.com/gobuffalo/packr/v2/packr2
COPY go.* /root/build/
RUN go mod download

COPY . /root/build/
RUN go test -race ./...
RUN packr2
RUN CGO_ENABLED=0 GOOS=linux go build -o app cmd/*.go

# Following does not work on rpi
#FROM alpine:latest  
#RUN apk --no-cache add ca-certificates

#WORKDIR /root/
#COPY --from=0 /root/build/app .
#CMD ["./app"]  
