//go:build integration
// +build integration

package gocql

import "testing"

func TestShardAwarePortIntegrationNoReconnections(t *testing.T) {
	testShardAwarePortNoReconnections(t, func() *ClusterConfig {
		return createCluster()
	})
}

func TestShardAwarePortIntegrationMaliciousNAT(t *testing.T) {
	testShardAwarePortMaliciousNAT(t, func() *ClusterConfig {
		return createCluster()
	})
}

func TestShardAwarePortIntegrationUnreachable(t *testing.T) {
	testShardAwarePortUnreachable(t, func() *ClusterConfig {
		return createCluster()
	})
}

func TestShardAwarePortIntegrationUnusedIfNotEnabled(t *testing.T) {
	testShardAwarePortUnusedIfNotEnabled(t, func() *ClusterConfig {
		return createCluster()
	})
}
