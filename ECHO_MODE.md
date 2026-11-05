# Authenticated bidirectional Echo mode

Enable `echo_mode: true` in the JSON config, or `-echo-mode`, on both upgraded
peers. The default is false and retains the existing v3 protocol. A one-sided
upgrade cannot communicate: the encrypted envelopes use a separate authenticated
domain, with no automatic fallback. All peers still need the same strong password.

## Behavior

* Client and server carry encrypted application frames only in type-8, code-0
  Echo Requests. Each endpoint advances its own sequence counter.
* After validating the ICMP checksum, AEAD, message magic, auth code and direction,
  the application (or the native Linux kernel responder) returns a type-0,
  code-0 Echo Reply with the exact request payload,
  identifier and sequence. Replies are acknowledgements, never new application data.
* The client session identifier is authenticated with the encrypted message and
  preserved for independent clients sharing a server. Reflected opposite-direction
  messages, wrong identifiers and unauthenticated input receive no application reply.
* A bounded window suppresses repeat delivery of the most recent 4096 encrypted
  messages, while still acknowledging duplicates. This is a bounded duplicate
  filter, not permanent replay protection. FrameMgr still handles loss/retransmission.
* Legacy framing, forwarding, congestion control and default configuration remain
  available. Echo mode requires authenticated encryption and raw IPv4 ICMP sockets.
  Unprivileged Android datagram ICMP can rewrite identifiers and is rejected.

## Deployment requirements

Both hosts must accept incoming Echo Requests from their peer. This mode cannot
work through a path that permits only client-originated Echo Requests and their
matching replies. NAT and firewalls may treat server-originated requests
differently; verify the actual path before switching an existing service.

On Linux, native raw sockets now inspect the namespace's
`net.ipv4.icmp_echo_ignore_all` setting when they open. With the normal value
`0`, the kernel sends exact Echo Replies and the application avoids a second
reply. Ordinary ping remains available. With value `1`, the application replies
only to authenticated tunnel requests and logs that ordinary ping is disabled.
Other platforms retain application replies. Restart the tunnel after changing
this host setting; it is intentionally not monitored on every packet.

The application does not modify sysctls or firewall rules. Do not disable normal
kernel Echo replies merely to run this version. If an older setup changed that
setting, restore the intended host policy yourself before restarting the tunnel.
Kernel replies to arbitrary ping payloads are normal host behavior and do not
mean that those payloads passed tunnel authentication.

## OpenVPN TCP/2083 example

These example addresses must be replaced. OpenVPN must be reachable on the
foreign host at `127.0.0.1:2083`; the Iran-side port must be free. Preserve the
existing OpenVPN certificates, keys and server verification settings.

Foreign host (`server.json`):

```json
{
  "type": "server",
  "icmp_listen": "0.0.0.0",
  "password": "REPLACE_WITH_A_STRONG_SHARED_PASSWORD",
  "encrypt": "chacha20",
  "echo_mode": true
}
```

Iran host (`client.json`):

```json
{
  "type": "client",
  "listen": "0.0.0.0:2083",
  "server": "2.2.2.2",
  "target": "127.0.0.1:2083",
  "password": "REPLACE_WITH_A_STRONG_SHARED_PASSWORD",
  "encrypt": "chacha20",
  "echo_mode": true
}
```

Protect each file with `chmod 600`. Run the corresponding config with
`sudo ./pingtunnel-echo -c server.json` or `sudo ./pingtunnel-echo -c client.json`.
The OpenVPN client connects to the Iran host's real IP on TCP/2083.

## What this change establishes

The change fixes the observable mismatch between an Echo Request's payload and
the application's Echo Reply. It retains AEAD encryption of internal messages.
It does not reproduce the timing, size distribution or traffic rate of ordinary
ping, conceal that the protocol is ICMP, or prevent an operator from blocking
or limiting ICMP. It has not been validated against a particular operator's DPI.
Exact replies also add carrier traffic; measure bandwidth and throughput on the
actual link before deploying broadly.

Tests cover exact replies, invalid input, mode separation, session routing,
duplicate delivery, sequence wrapping, bounded replay state, and end-to-end
TCP/UDP/SOCKS forwarding. Real-ICMP tests must report their underlay explicitly;
the unprivileged harness can otherwise use its in-process UDP transport.

Build with the Go version pinned in `go.mod`. Build the
current source with:

```sh
go build -mod=readonly -trimpath -o pingtunnel-echo ./cmd
go test ./... -skip '^TestE2E' -count=1
go test -race ./... -run 'TestEchoMode|TestLoadConfigEchoMode|TestWireCompatWithOldVersion' -count=1
```

On Linux, the following runs selected end-to-end tests in a private loopback-only
user/network namespace. The sysctl affects only that temporary namespace:

```sh
go test -c -o /tmp/pingtunnel-echo.test .
unshare --user --map-root-user --net sh -c '
  set -e
  ip link set lo up
  sysctl -w net.ipv4.icmp_echo_ignore_all=1
  PINGTUNNEL_E2E_REQUIRE_REAL_ICMP=1 /tmp/pingtunnel-echo.test \
    -test.run "^TestE2E_(EchoMode|EchoModeTwoClients|TCP_ForwardEcho|UDP_ForwardProxy|SOCKS5_ForwardProxy)$" \
    -test.v -test.timeout=3m
'
```

## Idle connections and resource use

This module supports the capacity and padding settings.
For many connections use the capacity sample configs; the historical default
1 MiB / 20000-frame profile is retained for compatibility and can exceed an
8 GiB server's budget at thousands of connections.

Idle FrameMgr PING/HB generation uses a five-second interval varied by ±10%
after five seconds without payload. Active flows keep one-second control probes.
Payload, ACK, retransmission and close processing are not held until a keepalive.
The adapted frame manager lives in `internal/reliable/` with its upstream tests
and MIT attribution; protobuf types and framing remain compatible.

Linux requests 4 MiB read/write socket buffers once per shared ICMP socket.
Kernel limits can clamp those requests. No global network setting is changed.
Receiver plaintext scratch storage and send-frame slices are reused while exact
Echo replies retain the original authenticated ciphertext.

These changes reduce avoidable CPU work, control traffic and duplicate replies.
They do not make high-rate bidirectional ICMP indistinguishable from ordinary ping.
