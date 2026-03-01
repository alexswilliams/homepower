FROM docker.io/golang:1.26.0-trixie AS build
WORKDIR /usr/src/app
COPY Makefile go.mod go.sum ./
RUN make deps
COPY . ./
RUN make bin/main

FROM scratch
WORKDIR /app
COPY --from=build /usr/src/app/bin/main /app/main
ENTRYPOINT ["/app/main"]
