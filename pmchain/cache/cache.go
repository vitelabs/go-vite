package chain_cache

type Cache struct {
	unconfirmedPool *UnconfirmedPool
}

func NewCache() *Cache {
	return &Cache{}
}

func (cache *Cache) UnconfirmedPool() *UnconfirmedPool {
	return cache.unconfirmedPool
}

func (cache *Cache) Destroy() {}
