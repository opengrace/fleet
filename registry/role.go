package registry

import (
	"path"
	"time"

	"github.com/coreos/fleet/etcd"
)

const (
	leasePrefix = "lease"
)

// AcquireLease acquires a named lease only if the lease is not
// currently held. If a Lease cannot be acquired, a nil Lease
// object is returned. An error is returned only if there is a
// failure communicating with the Registry.
func (r *EtcdRegistry) AcquireLease(name, machID string, period time.Duration) (Lease, error) {
	key := path.Join(r.keyPrefix, leasePrefix, name)
	req := etcd.Create{
		Key:   key,
		Value: machID,
		TTL:   period,
	}

	var lease Lease
	resp, err := r.etcd.Do(&req)
	if err == nil {
		lease = &etcdLease{
			key:   key,
			value: machID,
			idx:   resp.Node.ModifiedIndex,
			etcd:  r.etcd,
		}
	} else if isNodeExist(err) {
		err = nil
	}

	return lease, err
}

// etcdLease is a proxy to an auto-expiring lease stored in the Registry.
// The creator of a Lease must repeatedly call Renew to keep their lease from
// expiring. etcdLease implements the Lease interface.
type etcdLease struct {
	key   string
	value string
	idx   uint64
	etcd  etcd.Client
}

// Release explicitly releases the ownership of a lease back to the Registry.
// After calling Release, the etcdLease object should be discarded. An
// error is returned if the etcdLease has already expired, or if
// communication with the Registry fails.
func (l *etcdLease) Release() error {
	req := etcd.Delete{
		Key:           l.key,
		PreviousIndex: l.idx,
	}
	_, err := l.etcd.Do(&req)
	return err
}

// Renew attempts to update the remaining lease time to the provided time
// period. It will only succeed if the lease has not been changed in the
// Registry since it was last renewed or first acquired.
// An error is returned if the lease has already expired, or if communication
// with the Registry fails.
func (l *etcdLease) Renew(period time.Duration) error {
	req := etcd.Set{
		Key:           l.key,
		Value:         l.value,
		PreviousIndex: l.idx,
		TTL:           period,
	}

	resp, err := l.etcd.Do(&req)
	if err == nil {
		l.idx = resp.Node.ModifiedIndex
	}

	return err
}
