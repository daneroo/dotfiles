# Build stage for compiling the Go binary
FROM ghcr.io/homebrew/ubuntu26.04:latest AS builder

# Install Go with brew
RUN brew install go

# Copy and build the Go binary
COPY . /home/linuxbrew/.dotfiles
WORKDIR /home/linuxbrew/.dotfiles/go/cmd/reconfig
RUN go build -buildvcs=false -o /tmp/reconfig

# Final stage with clean environment. Current bottles need the glibc of
# Ubuntu 24.04 or newer; on older images brew installs its own glibc and gcc.
FROM ghcr.io/homebrew/ubuntu26.04:latest

# Copy the prebuilt binary
COPY --from=builder /tmp/reconfig /usr/local/bin/

# Copy dotfiles repository into container, owned by the user reconfig runs
# as, as on a real machine
COPY --chown=linuxbrew:linuxbrew . /home/linuxbrew/.dotfiles
WORKDIR /home/linuxbrew/.dotfiles

ENTRYPOINT ["/home/linuxbrew/.dotfiles/e2e/entrypoint.sh"]