# syntax=docker/dockerfile:1

# https://docs.docker.com/guides/golang/build-images/#multi-stage-builds
FROM golang:1.24 AS build-stage

# https://docs.docker.com/guides/zscaler/#building-with-the-certificate
# You'll need to create trusted_certs.crt to build locally with zscaler (HP)
#COPY trusted_certs.crt /usr/local/share/ca-certificates/zscaler-root-ca.crt
#RUN apt-get update && \
#    apt-get install -y ca-certificates && \
#    update-ca-certificates

# To make things easier when running the rest of your commands, create a directory inside the image that you're building.
# This also instructs Docker to use this directory as the default destination for all subsequent commands.
# This way you don't have to type out full file paths in the Dockerfile, the relative paths will be based on this directory.
WORKDIR /app

# before you can run go mod download inside your image, you need to get your go.mod and go.sum files copied into it.
COPY go.mod go.sum ./

RUN go mod download

# copy your source code into the image
COPY . .

# Now, to compile your application
RUN CGO_ENABLED=0 GOOS=linux go build -o /docker

# Deploy the application binary into a lean image
FROM gcr.io/distroless/base-debian11:latest AS build-release-stage

WORKDIR /

COPY --from=build-stage /docker /docker

# Optional:
# To bind to a TCP port, runtime parameters must be supplied to the docker command.
# But we can document in the Dockerfile what ports the application is going to listen on by default.
# https://docs.docker.com/reference/dockerfile/#expose
EXPOSE 8080

USER nonroot:nonroot

ENTRYPOINT ["/docker"]

# Since you have two Dockerfiles now, you have to tell Docker what Dockerfile you'd like to use to build the image.
# docker build -t docker-gs-ping:multistage -f Dockerfile.multistage .