# Build stage for compiling the Go binary
FROM homebrew/ubuntu24.04:latest AS builder

# Install Go with brew
RUN brew install go

# Copy and build the Go binary
COPY . /home/linuxbrew/.dotfiles
WORKDIR /home/linuxbrew/.dotfiles/go/cmd/reconfig
RUN go build -buildvcs=false -o /tmp/reconfig

# Final stage with clean environment. Ubuntu 24.04 has glibc 2.39, which
# current bottles need; on older images brew installs its own glibc and gcc.
FROM homebrew/ubuntu24.04:latest

# Copy the prebuilt binary
COPY --from=builder /tmp/reconfig /usr/local/bin/

# Copy dotfiles repository into container
COPY . /home/linuxbrew/.dotfiles
WORKDIR /home/linuxbrew/.dotfiles

ENTRYPOINT ["/home/linuxbrew/.dotfiles/e2e/entrypoint.sh"]