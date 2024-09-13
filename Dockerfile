# Use the Lambda Go base image from public ECR
FROM public.ecr.aws/lambda/go:latest as build

# Set the desired version of Go
ARG GO_VERSION=1.23.1

# Install dependencies
RUN yum install -y tar wget git

# Download and install the specified version of Go
RUN wget https://golang.org/dl/go${GO_VERSION}.linux-amd64.tar.gz \
    && tar -C /usr/local -xzf go${GO_VERSION}.linux-amd64.tar.gz \
    && rm go${GO_VERSION}.linux-amd64.tar.gz

# Set Go paths
ENV PATH="/usr/local/go/bin:${PATH}"

# Verify installation
RUN go version

# cache dependencies
ADD src/go.mod src/go.sum ./
RUN go mod download
# build
ADD src/. .
RUN go build -o /main

FROM public.ecr.aws/lambda/go:latest
COPY --from=build /main /main
ARG COMMIT_HASH
ENV COMMIT_HASH=$COMMIT_HASH