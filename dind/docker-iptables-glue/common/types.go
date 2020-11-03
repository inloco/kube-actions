package common

type NetworkInfo struct {
	ID      string
	Action  int
	Subnets []Subnet
}

type Subnet struct {
	DockerSubnet    string
	HostProxySubnet string
	SecondOctet     int
}

type ContainerInfo struct {
	ID     string
	Action int
	Pid    int
	Ports  []Port
	IPs    []string
}

type Port struct {
	Proto string
	Port  int
}

type ContainerInspectInfo struct {
	Pid      int
	Ports    []Port
	Networks []struct {
		ID string
		IP string
	}
}
