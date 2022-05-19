FROM docker.io/library/golang:1.16-alpine as build

WORKDIR /app

COPY *.go ./

RUN go build -o /buildah-example

FROM alpine

COPY --from=build /buildah-example /buildah-example

EXPOSE 8080

CMD [ "/buildah-example" ]