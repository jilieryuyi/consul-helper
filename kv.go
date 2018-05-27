package consul

import (
	"github.com/hashicorp/consul/api"
	log "github.com/sirupsen/logrus"
	"github.com/pkg/errors"
)

var KvDoesNotExists = errors.New("kv does not exists")
type Kv struct {
	kv *api.KV
}
func NewKv(kv *api.KV) *Kv {
	return &Kv{kv:kv}
}

// set key value
func (k *Kv) Set(key string, value string) error {
	log.Debugf("write %s=%s", key, value)
	kv := &api.KVPair{
		Key:key,
		Value:[]byte(value),
	}
	_, err := k.kv.Put(kv, nil)
	return err
}

// get key value
// if key does not exists, return error:KvDoesNotExists
func (k *Kv) Get(key string) (string, error) {
	kv, m, e := k.kv.Get(key, nil)
	log.Infof("kv == %+v,", kv)
	log.Infof("m == %+v,", m)
	if e != nil {
		log.Errorf("%+v", e)
		return "", e
	}
	if kv == nil {
		return "", KvDoesNotExists
	}
	return string(kv.Value), nil
}

//delete key value
func (k *Kv) Delete(key string) error {
	_, e := k.kv.Delete(key, nil)
	if e != nil {
		log.Errorf("%+v", e)
	}
	return e
}