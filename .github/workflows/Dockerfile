FROM golang:1.23-alpine

WORKDIR /backend

# Install build dependencies
## musl-dev included to support glibc dependencies
## (https://stackoverflow.com/questions/52056387/how-to-install-go-in-alpine-linux)
RUN apk add --no-cache gcc musl-dev

# Create separate caching layer for Go dependencies. If these files have
# not changed, Docker will use the cache for dependency resolution and 
# avoid re-downloading unchanged modules even if other source code has changed
COPY go.mod .
COPY go.sum .

RUN go mod download

# Copy other source code in new cache layer separate from dependencies
COPY . .

# Download dependencies and run tests
CMD ["go", "test", "--tags", "fts5", "-v", "./..."]