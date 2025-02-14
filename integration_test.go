//go:build all || integration
// +build all integration

package gocql

// This file groups integration tests where Cassandra has to be set up with some special integration variables
import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

// TestAuthentication verifies that gocql will work with a host configured to only accept authenticated connections
func TestAuthentication(t *testing.T) {

	if *flagProto < 2 {
		t.Skip("Authentication is not supported with protocol < 2")
	}

	if !*flagRunAuthTest {
		t.Skip("Authentication is not configured in the target cluster")
	}

	cluster := createCluster()

	cluster.Authenticator = PasswordAuthenticator{
		Username: "cassandra",
		Password: "cassandra",
	}

	session, err := cluster.CreateSession()

	if err != nil {
		t.Fatalf("Authentication error: %s", err)
	}

	session.Close()
}

func TestGetHostsFromSystem(t *testing.T) {
	clusterHosts := getClusterHosts()
	cluster := createCluster()
	session := createSessionFromCluster(cluster, t)

	hosts, partitioner, err := session.hostSource.GetHostsFromSystem()

	assertTrue(t, "err == nil", err == nil)
	assertEqual(t, "len(hosts)", len(clusterHosts), len(hosts))
	assertTrue(t, "len(partitioner) != 0", len(partitioner) != 0)
}

// TestRingDiscovery makes sure that you can autodiscover other cluster members
// when you seed a cluster config with just one node
func TestRingDiscovery(t *testing.T) {
	clusterHosts := getClusterHosts()
	cluster := createCluster()
	cluster.Hosts = clusterHosts[:1]

	session := createSessionFromCluster(cluster, t)
	defer session.Close()

	if *clusterSize > 1 {
		// wait for autodiscovery to update the pool with the list of known hosts
		time.Sleep(*flagAutoWait)
	}

	session.pool.mu.RLock()
	defer session.pool.mu.RUnlock()
	size := len(session.pool.hostConnPools)

	if *clusterSize != size {
		for p, pool := range session.pool.hostConnPools {
			t.Logf("p=%q host=%v ips=%s", p, pool.host, pool.host.ConnectAddress().String())

		}
		t.Errorf("Expected a cluster size of %d, but actual size was %d", *clusterSize, size)
	}
}

// TestHostFilterDiscovery ensures that host filtering works even when we discover hosts
func TestHostFilterDiscovery(t *testing.T) {
	clusterHosts := getClusterHosts()
	if len(clusterHosts) < 2 {
		t.Skip("skipping because we don't have 2 or more hosts")
	}
	cluster := createCluster()
	rr := RoundRobinHostPolicy().(*roundRobinHostPolicy)
	cluster.PoolConfig.HostSelectionPolicy = rr
	// we'll filter out the second host
	filtered := clusterHosts[1]
	cluster.Hosts = clusterHosts[:1]
	cluster.HostFilter = HostFilterFunc(func(host *HostInfo) bool {
		if host.ConnectAddress().String() == filtered {
			return false
		}
		return true
	})
	session := createSessionFromCluster(cluster, t)
	defer session.Close()

	assertEqual(t, "len(clusterHosts)-1 != len(rr.hosts.get())", len(clusterHosts)-1, len(rr.hosts.get()))
}

// TestHostFilterInitial ensures that host filtering works for the initial
// connection including the control connection
func TestHostFilterInitial(t *testing.T) {
	clusterHosts := getClusterHosts()
	if len(clusterHosts) < 2 {
		t.Skip("skipping because we don't have 2 or more hosts")
	}
	cluster := createCluster()
	rr := RoundRobinHostPolicy().(*roundRobinHostPolicy)
	cluster.PoolConfig.HostSelectionPolicy = rr
	// we'll filter out the second host
	filtered := clusterHosts[1]
	cluster.HostFilter = HostFilterFunc(func(host *HostInfo) bool {
		if host.ConnectAddress().String() == filtered {
			return false
		}
		return true
	})
	session := createSessionFromCluster(cluster, t)
	defer session.Close()

	assertEqual(t, "len(clusterHosts)-1 != len(rr.hosts.get())", len(clusterHosts)-1, len(rr.hosts.get()))
}

func TestWriteFailure(t *testing.T) {
	t.Skip("skipped due to unknown purpose")
	cluster := createCluster()
	createKeyspace(t, cluster, "test", false)
	cluster.Keyspace = "test"
	session, err := cluster.CreateSession()
	if err != nil {
		t.Fatal("create session:", err)
	}
	defer session.Close()
	if err := createTable(session, "CREATE TABLE test.test (id int,value int,PRIMARY KEY (id))"); err != nil {
		t.Fatalf("failed to create table with error '%v'", err)
	}
	if err := session.Query(`INSERT INTO test.test (id, value) VALUES (1, 1)`).Exec(); err != nil {
		errWrite, ok := err.(*RequestErrWriteFailure)
		if ok {
			if session.cfg.ProtoVersion >= 5 {
				// ErrorMap should be filled with some hosts that should've errored
				if len(errWrite.ErrorMap) == 0 {
					t.Fatal("errWrite.ErrorMap should have some failed hosts but it didn't have any")
				}
			} else {
				// Map doesn't get filled for V4
				if len(errWrite.ErrorMap) != 0 {
					t.Fatal("errWrite.ErrorMap should have length 0, it's: ", len(errWrite.ErrorMap))
				}
			}
		} else {
			t.Fatalf("error (%s) should be RequestErrWriteFailure, it's: %T", err, err)
		}
	} else {
		t.Fatal("a write fail error should have happened when querying test keyspace")
	}

	if err = session.Query("DROP KEYSPACE test").Exec(); err != nil {
		t.Fatal(err)
	}
}

func TestCustomPayloadMessages(t *testing.T) {
	t.Skip("SKIPPING")
	cluster := createCluster()
	session := createSessionFromCluster(cluster, t)
	defer session.Close()

	if err := createTable(session, "CREATE TABLE gocql_test.testCustomPayloadMessages (id int, value int, PRIMARY KEY (id))"); err != nil {
		t.Fatal(err)
	}

	// QueryMessage
	var customPayload = map[string][]byte{"a": []byte{10, 20}, "b": []byte{20, 30}}
	query := session.Query("SELECT id FROM testCustomPayloadMessages where id = ?", 42).Consistency(One).CustomPayload(customPayload)
	iter := query.Iter()
	rCustomPayload := iter.GetCustomPayload()
	if !reflect.DeepEqual(customPayload, rCustomPayload) {
		t.Fatal("The received custom payload should match the sent")
	}
	iter.Close()

	// Insert query
	query = session.Query("INSERT INTO testCustomPayloadMessages(id,value) VALUES(1, 1)").Consistency(One).CustomPayload(customPayload)
	iter = query.Iter()
	rCustomPayload = iter.GetCustomPayload()
	if !reflect.DeepEqual(customPayload, rCustomPayload) {
		t.Fatal("The received custom payload should match the sent")
	}
	iter.Close()

	// Batch Message
	b := session.Batch(LoggedBatch)
	b.CustomPayload = customPayload
	b.Query("INSERT INTO testCustomPayloadMessages(id,value) VALUES(1, 1)")
	if err := session.ExecuteBatch(b); err != nil {
		t.Fatalf("query failed. %v", err)
	}
}

func TestCustomPayloadValues(t *testing.T) {
	t.Skip("SKIPPING")
	cluster := createCluster()
	session := createSessionFromCluster(cluster, t)
	defer session.Close()

	if err := createTable(session, "CREATE TABLE gocql_test.testCustomPayloadValues (id int, value int, PRIMARY KEY (id))"); err != nil {
		t.Fatal(err)
	}

	values := []map[string][]byte{map[string][]byte{"a": []byte{10, 20}, "b": []byte{20, 30}}, nil, map[string][]byte{"a": []byte{10, 20}, "b": nil}}

	for _, customPayload := range values {
		query := session.Query("SELECT id FROM testCustomPayloadValues where id = ?", 42).Consistency(One).CustomPayload(customPayload)
		iter := query.Iter()
		rCustomPayload := iter.GetCustomPayload()
		if !reflect.DeepEqual(customPayload, rCustomPayload) {
			t.Fatal("The received custom payload should match the sent")
		}
	}
}

func TestSessionAwaitSchemaAgreement(t *testing.T) {
	session := createSession(t)
	defer session.Close()

	if err := session.AwaitSchemaAgreement(context.Background()); err != nil {
		t.Fatalf("expected session.AwaitSchemaAgreement to not return an error but got '%v'", err)
	}
}

func TestSessionAwaitSchemaAgreementSessionClosed(t *testing.T) {
	session := createSession(t)
	session.Close()

	if err := session.AwaitSchemaAgreement(context.Background()); !errors.Is(err, ErrConnectionClosed) {
		t.Fatalf("expected session.AwaitSchemaAgreement to return ErrConnectionClosed but got '%v'", err)
	}

}

func TestSessionAwaitSchemaAgreementContextCanceled(t *testing.T) {
	session := createSession(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := session.AwaitSchemaAgreement(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected session.AwaitSchemaAgreement to return 'context canceled' but got '%v'", err)
	}

}
