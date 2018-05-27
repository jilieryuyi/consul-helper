package consul

import (
	"github.com/hashicorp/consul/api"
	log "github.com/sirupsen/logrus"
	"time"
	"encoding/json"
)

type ICoder interface {
	Encode(data interface{}) ([]byte, error)
	Decode(data []byte) (interface{}, error)
}

type WatchKv struct {
	coder ICoder
	kv *api.KV
	prefix string
	notify []Notify
}

type Notify func(kv *api.KV, key string, data interface{})
type DefaultCoder struct {}

func (d *DefaultCoder) Encode(data interface{}) ([]byte, error) {
	return json.Marshal(data)
}

func (d *DefaultCoder) Decode(data []byte) (interface{}, error) {
	var res interface{}
	err := json.Unmarshal(data, &res)
	return res, err
}

type WatchKvOption func(k *WatchKv)
func SetCoder(coder ICoder) WatchKvOption {
	return func(k *WatchKv) {
		k.coder = coder
	}
}

func SetNotify(n Notify) WatchKvOption {
	return func(k *WatchKv) {
		k.notify = append(k.notify, n)
	}
}

func NewConsulWatchKv(kv *api.KV, prefix string, options ...WatchKvOption) *WatchKv {
	k := &WatchKv{
		prefix:prefix,
		kv:kv,
		notify:make([]Notify, 0),
	}
	k.coder = &DefaultCoder{}
	for _, f := range options {
		f(k)
	}
	return k
}

func (m *WatchKv) Watch() {
	go func() {
		lastIndex := uint64(0)
		for {
			_, me, err := m.kv.List(m.prefix, nil)
			if err != nil || me == nil {
				log.Errorf("%+v", err)
				time.Sleep(time.Second)
				continue
			}
			lastIndex = me.LastIndex
			break
		}
		for {
			qp := &api.QueryOptions{WaitIndex: lastIndex}
			kp, me, e :=  m.kv.List(m.prefix, qp)
			if e != nil {
				log.Errorf("%+v", e)
				time.Sleep(time.Second)
				continue
			}
			lastIndex = me.LastIndex
			for _, v := range kp {
				if len(v.Value) == 0 {
					continue
				}
				d, e := m.coder.Decode(v.Value)
				if e != nil {
					log.Errorf("%+v", v.Value)
					log.Errorf("%+v", string(v.Value))
					log.Errorf("%+v", e)
					continue
				}
				for _, n := range m.notify  {
					n(m.kv, v.Key, d)
				}
			}
			time.Sleep(time.Second)
		}
	}()
}
