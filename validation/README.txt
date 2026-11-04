Validation scope
===============
Local Linux cloud host, private user+network namespace, loopback raw ICMP.
No production Iran/foreign servers or OpenVPN user sessions were exercised.
No DPI appliance, remote filtering behavior, 100 Mbps WAN guarantee, or
zero-latency-regression claim was tested.

icmp-unit.txt: go test -p 4 ./... -skip '^TestE2E' -count=1 -timeout=5m
icmp-race.txt: go test -race -p 4 ./... -run 'TestPadding|TestHeartbeat|TestEchoMode|TestConfigureTCP|TestServerRejects|TestRunPadding' -count=1 -timeout=3m
icmp-functional.txt: required real ICMP, GOMAXPROCS=8, 2000 streams;
  Echo TCP/SOCKS5/UDP, shared-server clients, padded saturation, narrow pipe,
  half-close and slow reader. Some tests ran alongside race checks; timings
  from that run are not clean performance comparisons.
icmp-comparison.txt: sequential padded Echo and unpadded Echo saturation,
  same 256 KiB buffer / 512-frame profile, GOMAXPROCS=4, after builds and
  other tests finished. One run per case; no statistical performance claim.
icmp-padding-bench.txt: EncryptInto benchmark for 950-byte plaintext with
  preallocated buffers, padding 0/32/64, three runs each. Not an end-to-end
  sendICMP allocation benchmark.
resource-bench.txt: prior initial FrameMgr allocation benchmark for default
  and capacity profile. Not a total process RSS comparison.

Kernel sysctl was changed only inside private network namespaces. No host
firewall, global sysctl, remote deployment, or git push was performed.
Only amd64 binary was executed here; ARM binaries were cross-compiled.

CLI logging redirects stderr during some tests to cmd/pingtunnel.stderr;
package exit status and the captured end-of-suite PASS were inspected.
