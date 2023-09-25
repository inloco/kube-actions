/*
Copyright 2020 In Loco Tecnologia da Informação S.A.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package dot

type Runner struct {
	AgentId              int    `json:"agentId,omitempty"`
	AgentName            string `json:"agentNameTemplate,omitempty"`
	SkipSessionRecover   bool   `json:"skipSessionRecover,omitempty"`
	PoolId               int    `json:"poolId,omitempty"`
	PoolName             string `json:"poolName,omitempty"`
	DisableUpdate        bool   `json:"disableUpdate,omitempty"`
	Ephemeral            bool   `json:"ephemeral,omitempty"`
	ServerUrl            string `json:"serverUrl,omitempty"`
	GitHubUrl            string `json:"gitHubUrl,omitempty"`
	WorkFolder           string `json:"workFolder,omitempty"`
	MonitorSocketAddress string `json:"monitorSocketAddress,omitempty"`
	IsHostedServer       bool   `json:"isHostedServer,omitempty"`
}
