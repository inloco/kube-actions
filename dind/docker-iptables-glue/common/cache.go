package common

type Cache interface {
	AddNetworkInfo(info NetworkInfo)
	GetNetworkInfo(id string) (NetworkInfo, bool)
	DeleteNetworkInfo(id string) (NetworkInfo, bool)

	AddContainerInfo(info ContainerInfo)
	GetContainerInfo(id string) (ContainerInfo, bool)
	DeleteContainerInfo(id string) (ContainerInfo, bool)

	AddProxyPortPID(containerID string, pid int)
	GetProxyPortPIDs(containerID string) []int
	DeleteProxyPortPIDs(containerID string)
}

type cache struct {
	networkInfoByID            map[string]NetworkInfo
	containerInfoByID          map[string]ContainerInfo
	proxyPortPIDsByContainerID map[string][]int
}

func NewCache() Cache {
	return &cache{
		networkInfoByID: map[string]NetworkInfo{},
		containerInfoByID: map[string]ContainerInfo{},
		proxyPortPIDsByContainerID: map[string][]int{},
	}
}

func (c *cache) AddNetworkInfo(info NetworkInfo) {
	c.networkInfoByID[info.ID] = info
}

func (c *cache) GetNetworkInfo(id string) (NetworkInfo, bool) {
	info, ok := c.networkInfoByID[id]
	return info, ok
}

func (c *cache) DeleteNetworkInfo(id string) (NetworkInfo, bool) {
	info, ok := c.networkInfoByID[id]
	delete(c.networkInfoByID, id)
	return info, ok
}

func (c *cache) AddContainerInfo(info ContainerInfo) {
	c.containerInfoByID[info.ID] = info
	c.proxyPortPIDsByContainerID[info.ID] = []int{}
}

func (c *cache) GetContainerInfo(id string) (ContainerInfo, bool) {
	info, ok := c.containerInfoByID[id]
	return info, ok
}

func (c *cache) DeleteContainerInfo(id string) (ContainerInfo, bool) {
	info, ok := c.containerInfoByID[id]
	delete(c.containerInfoByID, id)
	return info, ok
}

func (c *cache) AddProxyPortPID(containerID string, pid int) {
	c.proxyPortPIDsByContainerID[containerID] = append(c.proxyPortPIDsByContainerID[containerID], pid)
}

func (c *cache) GetProxyPortPIDs(containerID string) []int {
	return c.proxyPortPIDsByContainerID[containerID]
}

func (c *cache) DeleteProxyPortPIDs(containerID string) {
	c.proxyPortPIDsByContainerID[containerID] = []int{}
}
