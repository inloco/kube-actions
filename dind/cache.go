package main

type Cache struct {
	networkInfoByID            map[string]NetworkInfo
	containerInfoByID          map[string]ContainerInfo
	proxyPortPIDsByContainerID map[string][]int
}

func NewCache() *Cache {
	return &Cache{
		networkInfoByID:            map[string]NetworkInfo{},
		containerInfoByID:          map[string]ContainerInfo{},
		proxyPortPIDsByContainerID: map[string][]int{},
	}
}

func (c *Cache) AddNetworkInfo(info NetworkInfo) {
	c.networkInfoByID[info.ID] = info
}

func (c *Cache) GetNetworkInfo(id string) (NetworkInfo, bool) {
	info, ok := c.networkInfoByID[id]
	return info, ok
}

func (c *Cache) DeleteNetworkInfo(id string) (NetworkInfo, bool) {
	info, ok := c.networkInfoByID[id]
	delete(c.networkInfoByID, id)
	return info, ok
}

func (c *Cache) AddContainerInfo(info ContainerInfo) {
	c.containerInfoByID[info.ID] = info
	c.proxyPortPIDsByContainerID[info.ID] = []int{}
}

func (c *Cache) GetContainerInfo(id string) (ContainerInfo, bool) {
	info, ok := c.containerInfoByID[id]
	return info, ok
}

func (c *Cache) DeleteContainerInfo(id string) (ContainerInfo, bool) {
	info, ok := c.containerInfoByID[id]
	delete(c.containerInfoByID, id)
	return info, ok
}

func (c *Cache) AddProxyPortPID(containerID string, pid int) {
	c.proxyPortPIDsByContainerID[containerID] = append(c.proxyPortPIDsByContainerID[containerID], pid)
}

func (c *Cache) GetProxyPortPIDs(containerID string) []int {
	return c.proxyPortPIDsByContainerID[containerID]
}

func (c *Cache) DeleteProxyPortPIDs(containerID string) {
	c.proxyPortPIDsByContainerID[containerID] = []int{}
}
