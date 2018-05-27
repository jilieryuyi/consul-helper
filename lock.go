package consul

import (
	log "github.com/sirupsen/logrus"
	"github.com/hashicorp/consul/api"
)

type Lock struct {
	Key string
	session *Session
	Kv *api.KV
}

func NewLock(session *Session, Kv *api.KV, key string) *Lock {
	con := &Lock{
		Key: key,
	}
	con.session = session
	con.Kv      = Kv
	return con
}

// timeOut seconds
func (con *Lock) Lock() (bool, error) {
	p := &api.KVPair{Key: con.Key, Value: nil, Session: con.session.ID}
	success, _, err := con.Kv.Acquire(p, nil)
	if err != nil {
		log.Errorf("lock error: %+v", err)
		return false, err
	}
	return success, nil

}

// unlock
func (con *Lock) Unlock() (bool, error) {
	p := &api.KVPair{Key: con.Key, Value: nil, Session: con.session.ID}
	success, _, err := con.Kv.Release(p, nil)
	if err != nil {
		log.Errorf("unlock error: %+v", err)
		return false, err
	}
	return success, err

}

// force unlock
func (con *Lock) Delete() {
	_, err := con.Kv.Delete(con.Key, nil)
	if err != nil {
		log.Errorf("delete errpor: %v", err)
		return
	}
}