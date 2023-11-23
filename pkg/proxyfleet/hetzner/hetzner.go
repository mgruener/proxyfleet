package hetzner

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strconv"
	"time"

	netwait "github.com/antelman107/net-wait-go/wait"
	"github.com/google/uuid"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/mgruener/proxyfleet/pkg/ipify"
	"github.com/mgruener/proxyfleet/pkg/proxyfleet"
)

type hcloudProxyFleet struct {
	client    *hcloud.Client
	waiter    *netwait.Executor
	locations map[string]int
}

func New(hcloudOptions ...hcloud.ClientOption) proxyfleet.ProxyFleet {
	client := hcloud.NewClient(hcloudOptions...)
	return &hcloudProxyFleet{
		client:    client,
		waiter:    netwait.New(netwait.WithDeadline(10*time.Minute), netwait.WithWait(1*time.Second), netwait.WithBreak(1*time.Second)),
		locations: map[string]int{},
	}
}

// EnsureProxies ensures there are at least <min> and at max <max>
// proxy instances available. If not enough proxy instances are
// available, it calls SpawnProxies to the missing amount. If there
// are too many proxy instances running it calls DespawnProxies to
// remove the excess proxy instances.
//
// Returns the IPs of the activ proxy instances, at least <min> but at max <max>.
func (hcpf *hcloudProxyFleet) EnsureProxies(min uint, max uint) ([]string, error) {
	ips, err := hcpf.GetProxies(int(max))
	if err != nil {
		return ips, err
	}
	ipCount := uint(len(ips))
	if ipCount < min {
		spawnedServerIPs, err := hcpf.SpawnProxies(min - ipCount)
		if spawnedServerIPs != nil {
			ips = append(ips, spawnedServerIPs...)
		}
		if err != nil {
			return ips, err
		}
	}
	if ipCount > max {
		remainingServerIPs, err := hcpf.SpawnProxies(ipCount - max)
		ips = remainingServerIPs
		if err != nil {
			return ips, err
		}
	}
	return ips, nil
}

// GetProxies returns the ip of <count> proxy instances.
func (hcpf *hcloudProxyFleet) GetProxies(count int) ([]string, error) {
	if count == 0 {
		return []string{}, nil
	}

	servers, err := hcpf.getProxies()
	if err != nil {
		return nil, err
	}

	returnCount := count
	serverCount := len(servers)
	if (count < 0) || (count > serverCount) {
		returnCount = serverCount
	}

	ips := make([]string, returnCount)
	for i, server := range servers {
		ips[i] = server.PublicNet.IPv4.IP.String()
	}
	hcpf.waitForProxies(ips)
	return ips, nil
}

// SpawnProxies creates <count> proxy instances. Returns the IPs of the
// newly created proxies.
func (hcpf *hcloudProxyFleet) SpawnProxies(count uint) ([]string, error) {
	myip, err := ipify.MyIP()
	if err != nil {
		return []string{}, err
	}
	ctx := context.Background()
	ips := make([]string, count)
	for i := uint(0); i < count; i++ {
		id := uuid.New()
		name := "proxy-" + id.String()
		location, err := hcpf.getLocation(ctx)
		if err != nil {
			return ips, err
		}
		serverType, err := hcpf.getServerType(ctx, location)
		if err != nil {
			return ips, err
		}
		image, err := hcpf.getImage(ctx, serverType.Architecture)
		if err != nil {
			return ips, err
		}
		sshKey, err := hcpf.getSSHKey(ctx)
		if err != nil {
			return ips, err
		}
		startAfterCreate := true

		userDataTpl, ok := imageToUserData[image.Name]
		if !ok {
			return ips, fmt.Errorf("no userdata found for image '%s'", image.Name)
		}

		opts := hcloud.ServerCreateOpts{
			Image: image,
			Labels: map[string]string{
				"owner": "hcpf",
			},
			Location: location,
			Name:     name,
			PublicNet: &hcloud.ServerCreatePublicNet{
				EnableIPv4: true,
				EnableIPv6: false,
			},
			ServerType:       serverType,
			SSHKeys:          []*hcloud.SSHKey{sshKey},
			StartAfterCreate: &startAfterCreate,
			UserData:         fmt.Sprintf(userDataTpl, myip),
		}
		result, _, err := hcpf.client.Server.Create(ctx, opts)
		if err != nil {
			return ips, err
		}
		ips[i] = result.Server.PublicNet.IPv4.IP.String()
		if _, ok := hcpf.locations[location.Name]; !ok {
			hcpf.locations[location.Name] = 0
		}
		hcpf.locations[location.Name]++
	}
	hcpf.waitForProxies(ips)
	return ips, nil
}

// DespawnProxies removes <count> proxy instances. Specifying a count of -1
// removes all proxies. Returns a list of remaining proxy IPs.
func (hcpf *hcloudProxyFleet) DespawnProxies(count int) ([]string, error) {
	ctx := context.Background()
	servers, err := hcpf.getProxies()
	if err != nil {
		return nil, err
	}

	serverCount := len(servers)
	if serverCount == 0 {
		return []string{}, nil
	}

	removeCount := count
	if (count < 0) || (count > serverCount) {
		removeCount = serverCount
	}

	fmt.Println("Removing servers:", removeCount)
	remainServers := servers[removeCount:]
	ips := make([]string, len(remainServers))
	for i, server := range remainServers {
		ips[i] = server.PublicNet.IPv4.IP.String()
	}

	if removeCount == 0 {
		return ips, nil
	}

	removeServers := servers[0:removeCount]
	for _, server := range removeServers {
		_, _, err := hcpf.client.Server.DeleteWithResult(ctx, server)
		if err != nil {
			return ips, err
		}
	}

	return ips, nil
}

func (hcpf *hcloudProxyFleet) waitForProxies(ips []string) {
	ipPort := make([]string, len(ips))
	for i, ip := range ips {
		ipPort[i] = fmt.Sprintf("%s:%s", ip, "8080")
	}
	hcpf.waiter.Do(ipPort)
}

func (hcpf *hcloudProxyFleet) getProxies() ([]*hcloud.Server, error) {
	ctx := context.Background()
	opts := hcloud.ServerListOpts{
		ListOpts: hcloud.ListOpts{LabelSelector: "owner=hcpf"},
	}
	servers, _, err := hcpf.client.Server.List(ctx, opts)
	if err != nil {
		return nil, err
	}
	for _, server := range servers {
		loc := server.Datacenter.Location.Name
		if _, ok := hcpf.locations[loc]; ok {
			hcpf.locations[loc]++
			continue
		}
		hcpf.locations[loc] = 1
	}
	return servers, nil
}

func (hcpf *hcloudProxyFleet) getImage(ctx context.Context, arch hcloud.Architecture) (*hcloud.Image, error) {
	image, _, err := hcpf.client.Image.GetByNameAndArchitecture(ctx, "fedora-39", arch)
	if err != nil {
		return image, err
	}

	return image, nil
}

func (hcpf *hcloudProxyFleet) getLocation(ctx context.Context) (*hcloud.Location, error) {
	var candidate *hcloud.Location
	locations, err := hcpf.client.Location.All(ctx)
	if err != nil {
		return nil, err
	}

	// select the location with the smallest amount of proxy instances
	minUsage := 0
	for _, location := range locations {
		currentUsage, ok := hcpf.locations[location.Name]
		// if we currently know of no proxy on this location
		// it is by definiton the least used, so we use it
		// for the next proxy
		if !ok {
			return location, nil
		}
		if (minUsage == 0) || (minUsage > currentUsage) {
			minUsage = currentUsage
			candidate = location
		}
	}

	return candidate, nil
}

func (hcpf *hcloudProxyFleet) getServerType(ctx context.Context, location *hcloud.Location) (*hcloud.ServerType, error) {
	var candidate *hcloud.ServerType

	serverTypes, err := hcpf.client.ServerType.All(ctx)
	if err != nil {
		return nil, err
	}

	var minCurrency string
	var minPrice float64 = 0
	for _, serverType := range serverTypes {
		if serverType.Architecture == "arm" {
			// exclude arm for now as it is not available
			// in the US and Hetzner does not seem to provide
			// a reliable way to detect which serverType is
			// available where
			continue
		}
		var netPrice float64
		var netCurrency string
		found := false
		for _, pricing := range serverType.Pricings {
			if pricing.Location.Name == location.Name {
				netPrice, err = strconv.ParseFloat(pricing.Hourly.Net, 64)
				if err != nil {
					continue
				}
				netCurrency = pricing.Hourly.Currency
				found = true
				break
			}
		}
		// the serverType is not available at the chosen location
		if !found {
			continue
		}

		// According to https://docs.hetzner.com/cloud/general/locations/#what-about-billing
		// Hetzner bills everywhere in euro and uses the same prices regardless of location.
		// Nevertheless this might change, so we at least should emit a warning when we compare
		// prices of different currencies.
		if (minCurrency != "") && (minCurrency != netCurrency) {
			fmt.Fprintf(os.Stderr, "currency conflict for serverType '%s': %s != %s", []any{serverType.Name, minCurrency, netCurrency}...)
		}
		if (minPrice == 0) || (minPrice > netPrice) {
			minPrice = netPrice
			candidate = serverType
			minCurrency = netCurrency
		}
	}

	fmt.Printf("Using serverType '%s' at location '%s' (%s) for a price of '%f %s'\n", candidate.Name, location.Name, location.Country, minPrice, minCurrency)
	return candidate, nil
}

func (hcpf *hcloudProxyFleet) getSSHKey(ctx context.Context) (*hcloud.SSHKey, error) {
	// if a SSH key name is specified in the environment, try to use this
	if envSSHKeyName := os.Getenv("HETZNER_SSHKEY"); envSSHKeyName != "" {
		sshKey, _, err := hcpf.client.SSHKey.GetByName(ctx, envSSHKeyName)
		if err != nil {
			return nil, err
		}
		return sshKey, nil
	}

	// no SSH key name was specified in the environment, just use the newest key
	// available, assuming this is most likely one that is still valid
	sshKeys, err := hcpf.client.SSHKey.All(ctx)
	if err != nil {
		return nil, err
	}
	if len(sshKeys) > 0 {
		sort.Slice(sshKeys, func(i, j int) bool {
			return sshKeys[i].Created.After(sshKeys[j].Created)
		})
		return sshKeys[0], nil
	}

	return nil, fmt.Errorf("no SSH key found")
}
