# frp

A Norb's new implementation of Fast Reverse Proxy, focusing on speed and multiplexing (neofrp).

## Requirements

A simple implementation of frp (Fast Reverse Proxy) that maps a port on a local machine A in the intranet to a port on a remote machine B in the internet, allowing the internet to access the service on A through B.

The implementation should support both TCP and UDP protocols, and support both TCP and QUIC in the transport layer.

Security should be enforced with mutual TLS authentication, with utilities to generate certificates and keys for both A and B based on openssl.

Bandwidth and latency should be comparable to the original frp implementation under 1Gbps network conditions.

Also, your implementation should be drastically different from the original frp implementation to avoid detection from corporate firewalls and antivirus.

## Design

In order to bypass firewall protection against the original frp implementation, it is important to avoid the patterns that are associated with the original frp.

1. Custom Protocol: The original frp uses a specific protocol which, though under TLS protection, can be detected via active probing. We shall use a custom protocol different from the original frp implementation.

2. Work Connections: Originally the frp service will spawn multiple work connections while maintaining a single control connection, and they do not share the same TLS session. This makes for easy detection without active probing. We will make sure that the handshake session persists and is used as the outbound connection, while forwarding all inbound connections.