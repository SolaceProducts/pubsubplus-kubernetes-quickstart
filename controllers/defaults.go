/*
Copyright 2023 Solace Corporation

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

// One place for defaults

package controllers

const (
	DefaultBrokerImageRepoK8s   = "solace/solace-pubsub-standard"
	DefaultBrokerImageTagK8s    = "latest"
	DefaultExporterImageRepoK8s = "solace/pubsubplus-prometheus-exporter"
	DefaultExporterImageTagK8s  = "latest"

	DefaultBrokerImageRepoOpenShift   = "registry.connect.redhat.com/solace/pubsubplus-standard"
	DefaultBrokerImageTagOpenShift    = "latest"
	DefaultExporterImageRepoOpenShift = "registry.connect.redhat.com/solace/pubsubplus-prometheus-exporter" // TODO: update once published to public
	DefaultExporterImageTagOpenShift  = "latest"
)
