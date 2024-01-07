# syntax=docker/dockerfile:1.4

#
# docker build --no-cache --progress=plain -t api .
# docker run -it --rm --name=api -p 8080:8080 api
#

FROM cgr.dev/chainguard/go:latest as build
WORKDIR /go/src/app
COPY . .
RUN go mod download && CGO_ENABLED=0 go build -o /go/bin/app

# https://github.com/chainguard-images/images/tree/main/images/static#users
FROM cgr.dev/chainguard/static:latest
COPY --from=build --chown=65532:65532 /go/src/app/public /public
COPY --from=build --chown=65532:65532 /go/src/app/server /server
COPY --from=build --chown=65532:65532 /go/bin/app /
EXPOSE 8080
ENTRYPOINT ["/app"]