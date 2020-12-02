FROM golang:1.14-alpine as build
WORKDIR /work
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build

FROM alpine
COPY --from=build /work/ssogen /ssogen
ENTRYPOINT ["/ssogen"]