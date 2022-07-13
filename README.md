> **Warning** This project is no longer being maintained. Use https://github.com/database64128/shadowsocks-go instead.

# v2ray-go

An opinionated fork of [v2fly/v2ray-core](https://github.com/v2fly/v2ray-core).

## Additional Features

| No. | Description | Commit | Author |
| --- | ----------- | ------ | ------ |
|  1. | 📅 Add flag `-suppressTimestamps` to suppress timestamps in logs. | 7c9f1f86727541e0e996733841773e54bce8b296 | @database64128 |
|  2. | 🔌 Refine systemd unit files | 09c9cd81347730557de50df100611c81043a449d | @database64128 |
|  3. | 🍥 DNS query strategy: Add UseIPFailFast: This strategy enables both IPv4 and IPv6. When either A or AAAA query fails, the lookup operation is deemed a failure. Fixes v2fly/v2ray-core#1209. | ddfb88eea5534730417c7c62c27ed932c3645306 | @database64128 |
|  4. | 📦 Release packages: Use proper GOOS-GOARCH naming instead of the confusing "friendly" names. | 989cc06ebeb79c172836cadc325bd037b909b165 | @database64128 |
|  5. | 🚮 Removed browser forwarder's `securedload`. | 8d9d576a2ca5fc152c453d9ceaa2f7dc0e559f83 | @moodyhunter |
|  6. | 🚮 Removed legacy MTProto v1. | 7894e4b46c10785bf29ff5ae47991769bdd32410 | @moodyhunter |
|  7. | 🚮 Removed deprecated `inboundDetour` and `outboundDetour` | 25ac8395e697ffad50ff735ca49487a3198bc096 | @moodyhunter |
|  8. | 🚮 Removed useless VLESS | aba0348410ebe7c3165b2f2f24377a2fe08cca32 | @moodyhunter |
|  9. | ⚡️ Performance: Enable ReadV by default | 0fcfd1ed94fe194fd5acb5553090a1f97de966f4 | @AkinoKaede |
| 10. | 🚮 VMess: Drop `vnext` and VMess MD5 (with `alterId`). To migrate your JSON config, just move what's previously in `vnext` up one level. | d99fa66e116612b3f834073671178b4b4e390a65, e7bc3878aed6ad861dea5f05b19053af88e23f2b | @AkinoKaede, @database64128 |
| 11. | 🌐 DNS: Respect TTL in RRs | 95eaaa93b80b39edb86b8b69d03e1fe3e3932aea | @rurirei |
| 12. | 👀 Happy Eyeballs v1 with built-in DNS and configurable fallback delay and domain strategy | #12 | @database64128 |
| 13. | 🔧 Fix: `net.ParseAddress` can't handle CIDR | #12 | @database64128 |
| 14. | 🪢 Replace `sendThrough` with `bindInterface` (interface name), `bind4` (interface name or IPv4), `bind6` (interface name or IPv6) | #12 | @database64128 |
| 15. | ➕ Performance: Increase buffer size from 2K to 20K | a87b0d2fc0f6b7b9051b98403f1a89e621c2e4a1 | @database64128 |
| 16. | ⏱️ Remove TCP idle timeout | 73e50fe7441263681cc104c0831cf9fe9a299b36 | @database64128 |
| 17. | 👃 Add routeOnly sniffing option | 2786ece5f3e96eaa177e62d7fb26be5a37ebb484 | @nekohasekai |
| 18. | 🦘 Sniffer: Add `SkipDomainDestinations` and `SkippedDomains` | 54893b89443136a49b46fba56bdf03f45f918f83 | @database64128 |
| 19. | 🦩 Refine linux TFO setsockopt | 497167dfc60af447a68630cc2e25f4638ee172d2 | @database64128 |

## License

This project is licensed under [AGPLv3](LICENSE)

The upstream [v2fly/v2ray-core](https://github.com/v2fly/v2ray-core) is licensed under the [MIT License](https://github.com/v2fly/v2ray-core/blob/master/LICENSE).
