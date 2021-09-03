package geodata

import "github.com/Shadowsocks-NET/v2ray-go/v4/app/router"

//go:generate go run github.com/Shadowsocks-NET/v2ray-go/v4/common/errors/errorgen

type LoaderImplementation interface {
	LoadSite(filename, list string) ([]*router.Domain, error)
	LoadIP(filename, country string) ([]*router.CIDR, error)
}

type Loader interface {
	LoaderImplementation
	LoadGeoSite(list string) ([]*router.Domain, error)
	LoadGeoSiteWithAttr(file string, siteWithAttr string) ([]*router.Domain, error)
	LoadGeoIP(country string) ([]*router.CIDR, error)
}
