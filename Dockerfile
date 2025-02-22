# Use the latest Go version (1.23)
FROM golang:1.23

# Set the working directory
WORKDIR /app

# Copy Go modules files first (if they exist) for better caching
COPY go.mod go.sum ./

# Download dependencies (only if go.mod exists)
RUN if [ -f "go.mod" ]; then go mod tidy; fi

# Copy the rest of the application code
COPY . .

# Build the Go application
RUN go build -o main .

# Expose the application port
EXPOSE 8080

# Run the Go application
CMD ["/app/main"]
