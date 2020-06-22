package main

import (
	"sync"
)

type BufferQueueKey struct {
	appId    string
	clientId string
}

type BufferQueue struct {
	mux  sync.Mutex
	bufs map[BufferQueueKey][]byte
}

func NewBufferQueue() (*BufferQueue) {
	bq := BufferQueue{}
	bq.bufs = make(map[BufferQueueKey][]byte)
	return &bq
}

func (p *BufferQueue) add(appId, clientId string, b []byte) {
	key := BufferQueueKey{
		appId: appId,
		clientId: clientId,
	}
	p.mux.Lock()
	p.bufs[key] = append(p.bufs[key], b...)
	p.mux.Unlock()
}

func (p *BufferQueue) remove(appId, clientId string) []byte {
	key := BufferQueueKey{
		appId: appId,
		clientId: clientId,
	}
	ret := make([]byte, len(p.bufs[key]))
	p.mux.Lock()
	copy(ret, p.bufs[key])
	delete(p.bufs, key)
	p.mux.Unlock()
	return ret
}
