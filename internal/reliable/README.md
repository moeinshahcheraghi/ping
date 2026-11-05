# Reliable frame manager

Derived from github.com/esrrhs/gohome/network at 909c7d106b1f (MIT; see LICENSE).
The protobuf types remain upstream aliases. Local changes allow idle-only,
jittered control intervals without modifying the data/ACK/retransmission wire
protocol or the dependency cache. Configure before starting the connection pump.
The upstream reliability and API tests accompany the adapted implementation.
