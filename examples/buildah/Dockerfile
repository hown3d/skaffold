FROM golang:1.16-alpine as build

WORKDIR /app

COPY go.mod app/

COPY *.go /app/

RUN go build -o /buildah-example main.go

FROM alpine

COPY --from=build /buildah-example /buildah-example

EXPOSE 8080

CMD [ "/buildah-example" ]