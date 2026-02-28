FROM quay.io/seiferma/mandos-server:latest

RUN apt-get update && \
    apt-get install -y --no-install-suggests --no-install-recommends golang socat openssh-client ca-certificates && \
    rm -rf /var/lib/{apt,dpkg,cache,log}/

RUN go install github.com/go-delve/delve/cmd/dlv@latest && \
    go install golang.org/x/tools/gopls@latest

# npx tooling
RUN apt-get update && \
    apt-get install -y --no-install-suggests --no-install-recommends npm && \
    rm -rf /var/lib/{apt,dpkg,cache,log}/ && \
    npm i -g npx

# example client for testing
RUN apt-get update && \
    apt-get install -y --no-install-suggests --no-install-recommends mandos-client procps locales && \
    sed -i -e 's/# en_US/en_US/' /etc/locale.gen && \
    dpkg-reconfigure --frontend=noninteractive locales && \
    rm -rf /var/lib/{apt,dpkg,cache,log}/
ENV DBUS_SESSION_BUS_ADDRESS=unix:path=/run/dbus/system_bus_socket
ENV LANG en_US.UTF-8
COPY clients.conf /etc/mandos/clients.conf