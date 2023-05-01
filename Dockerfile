# Use the official Golang image as the base image
FROM golang:1.20

LABEL authors="fpaupier"

# Set the working directory for your application
WORKDIR /app

# Copy Go modules and dependencies to the Docker container
COPY src/go.mod src/go.sum ./
RUN go mod download

# Copy your application source code to the Docker container
COPY src/*.go ./

# Build your application
RUN go build -o tms-downloader

# Set an entrypoint for your application
ENTRYPOINT [ "./tms-downloader" ]