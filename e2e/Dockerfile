# Build stage for compiling the Go binary
FROM homebrew/brew:latest AS builder

# Install Go with brew
RUN brew install go

# Copy and build the Go binary
COPY . /home/linuxbrew/.dotfiles
WORKDIR /home/linuxbrew/.dotfiles/go/cmd/checkdeps
RUN go build -buildvcs=false -o /tmp/checkdeps

# Final stage with clean environment
FROM homebrew/brew:latest

# Copy the prebuilt binary
COPY --from=builder /tmp/checkdeps /usr/local/bin/

# Copy dotfiles repository into container
COPY . /home/linuxbrew/.dotfiles
WORKDIR /home/linuxbrew/.dotfiles

ENTRYPOINT ["/home/linuxbrew/.dotfiles/e2e/entrypoint.sh"]