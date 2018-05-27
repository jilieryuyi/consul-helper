package consul
type KvEntity struct {
	kv IKv
	Key string
	Value []byte
}

func NewKvEntity(kv IKv, key string, value []byte) *KvEntity{
	return &KvEntity{kv, key, value}
}

func (kv *KvEntity) Set() (*KvEntity, error) {
	err := kv.kv.Set(kv.Key, kv.Value)
	return kv, err
}

func (kv *KvEntity) Get() (*KvEntity, error) {
	v, err := kv.kv.Get(kv.Key)
	kv.Value = v
	return kv, err
}

func (kv *KvEntity) Delete() (*KvEntity, error) {
	err := kv.kv.Delete(kv.Key)
	return kv, err
}