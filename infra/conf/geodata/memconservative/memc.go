package memconservative

import (
	"runtime"

	"github.com/Shadowsocks-NET/v2ray-go/v4/app/router"
	"github.com/Shadowsocks-NET/v2ray-go/v4/infra/conf/geodata"
)

type memConservativeLoader struct {
	geoipcache   GeoIPCache
	geositecache GeoSiteCache
}

func (m *memConservativeLoader) LoadIP(filename, country string) ([]*router.CIDR, error) {
	defer runtime.GC()
	geoip, err := m.geoipcache.Unmarshal(filename, country)
	if err != nil {
		return nil, newError("failed to decode geodata file: ", filename).Base(err)
	}
	return geoip.Cidr, nil
}

func (m *memConservativeLoader) LoadSite(filename, list string) ([]*router.Domain, error) {
	defer runtime.GC()
	geosite, err := m.geositecache.Unmarshal(filename, list)
	if err != nil {
		return nil, newError("failed to decode geodata file: ", filename).Base(err)
	}
	return geosite.Domain, nil
}

func newMemConservativeLoader() geodata.LoaderImplementation {
	return &memConservativeLoader{make(map[string]*router.GeoIP), make(map[string]*router.GeoSite)}
}

func init() {
	geodata.RegisterGeoDataLoaderImplementationCreator("memconservative", newMemConservativeLoader)
}
