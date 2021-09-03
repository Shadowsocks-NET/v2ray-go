# v2ray-go

An opinionated fork of [v2fly/v2ray-core](https://github.com/v2fly/v2ray-core).

## Additional Features

1. 📅 Add flag `-suppressTimestamps` to suppress timestamps in logs
2. 🔌 Refine systemd unit files
3. 🍥 DNS query strategy: Add UseIPFailFast: This strategy enables both IPv4 and IPv6. When either A or AAAA query fails, the lookup operation is deemed a failure. Fixes v2fly/v2ray-core#1209.

## License

This project is licensed under [AGPLv3](LICENSE)

The upstream [v2fly/v2ray-core](https://github.com/v2fly/v2ray-core) is licensed under the [MIT License](https://github.com/v2fly/v2ray-core/blob/master/LICENSE).
