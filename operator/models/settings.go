package models

type RunnerSettings struct {
	AgentId              int    `json:"agentId,omitempty"`
	AgentName            string `json:"agentNameTemplate,omitempty"`
	SkipSessionRecover   bool   `json:"skipSessionRecover,omitempty"`
	PoolId               int    `json:"poolId,omitempty"`
	PoolName             string `json:"poolName,omitempty"`
	ServerUrl            string `json:"serverUrl,omitempty"`
	GitHubUrl            string `json:"gitHubUrl,omitempty"`
	WorkFolder           string `json:"workFolder,omitempty"`
	MonitorSocketAddress string `json:"monitorSocketAddress,omitempty"`
	IsHostedServer       bool   `json:"isHostedServer,omitempty"`
}
