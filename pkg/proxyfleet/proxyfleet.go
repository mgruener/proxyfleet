package proxyfleet

type ProxyFleet interface {
	EnsureProxies(min uint, max uint) ([]string, error)
	SpawnProxies(count uint) ([]string, error)
	GetProxies(count int) ([]string, error)
	DespawnProxies(count int) ([]string, error)
}
