# Keyrafted

> **Keyrafted** is the server component of the Keyraft project — a lightweight, self-hosted, distributed configuration and secrets management system.

Keyrafted is responsible for running the cluster nodes, managing replication, storing configuration and secrets securely, and serving requests from clients, CLI, and SDKs.

---

## Features

* Distributed key-value store with Raft-based replication
* Encrypted storage (AES-256-GCM) for secrets
* Namespaces for isolation
* Token-based authentication with scoped API keys
* Versioned configuration values
* Watch API for live updates
* HTTP/JSON protocol
* CLI-first management via `keyraft` CLI

Planned Features:

* WebSocket/SSE streaming for live watch updates
* Role-based access control (RBAC)
* Audit logging
* Multi-region high availability
* gRPC API support

---

## Architecture

* Multi-node cluster using Raft for consensus
* Leader node handles write; followers replicate state and serve reads
* Client requests (CLI/SDK) are load-balanced across nodes
* Watch API ensures consistent real-time updates across all nodes

Diagram:

```
       +------------------------+
       |   Client / CLI / SDK   |
       +-----------+------------+
                   |
                   v
          +------------------+
          |   Keyrafted Node  |
          |  Leader / Follower|
          +--------+---------+
                   |
           Raft Replication
                   |
          +--------+---------+
          |   Follower Node   |
          +------------------+
```

---

## Installation

### Binary

```bash
curl -L https://keyraft.io/install.sh | sh
keyrafted --data-dir /data --listen :7331
```

### Docker

```bash
docker run -d -p 7331:7331 -v keyrafted-data:/data keyraft/keyrafted
```

---

## Configuration

* Data directory for storage: `--data-dir /path/to/data`
* Listen address: `--listen :PORT`
* Optional TLS configuration for production

---

## Running Keyrafted

```bash
# Start single node
keyrafted --data-dir /data --listen :7331

# For multi-node cluster, configure each node with unique ID and peer addresses
keyrafted --data-dir /data1 --listen :7331 --node-id node1 --peers node2:7332,node3:7333
```

---

## Logs & Monitoring

* Logs are written to stdout by default
* Can integrate with external logging/monitoring systems
* Metrics endpoint available for Prometheus or other monitoring tools

---

## Security

* Secrets encrypted at rest (AES-256-GCM)
* API keys with scoped access control
* TLS recommended in production
* Audit logs planned for admin actions

---

## Contributing

* Contributions welcome
* Submit PRs for review
* Follow coding guidelines and maintain protocol backward compatibility

---

## License

Apache License 2.0

---

## References / Inspiration

* etcd
* HashiCorp Vault
* Consul
* Apache ZooKeeper
