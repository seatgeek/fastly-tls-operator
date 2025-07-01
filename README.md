# Fastly Operator

A Go project that builds Kubernetes controllers for Fastly services.

## Project Structure

```
fastly-operator/
├── cmd/
│   └── main.go          # Application entry point
├── README.md
└── Dockerfile
```

## Building and Running

### Local Development

To build and run the application locally:

```bash
# Build the application
go build -o fastly-operator ./cmd

# Run the application
./fastly-operator
```

### Docker

To build and run with Docker:

```bash
# Build the Docker image
docker build -t fastly-operator .

# Run the container
docker run fastly-operator
```
