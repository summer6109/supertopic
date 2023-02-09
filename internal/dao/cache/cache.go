package cache

import (
	"time"

	"github.com/allegro/bigcache/v3"
	"github.com/rocboss/paopao-ce/internal/conf"
	"github.com/rocboss/paopao-ce/internal/core"
	"github.com/sirupsen/logrus"
)

func NewBigCacheIndexService(indexPosts core.IndexPostsService) (core.CacheIndexService, core.VersionInfo) {
	s := conf.BigCacheIndexSetting

	config := bigcache.DefaultConfig(s.ExpireInSecond)
	config.Shards = s.MaxIndexPage
	config.Verbose = s.Verbose
	config.MaxEntrySize = 10000
	config.Logger = logrus.StandardLogger()
	cache, err := bigcache.NewBigCache(config)
	if err != nil {
		logrus.Fatalf("initial bigCahceIndex failure by err: %v", err)
	}

	cacheIndex := &bigCacheIndexServant{
		ips:             indexPosts,
		cache:           cache,
		preventDuration: 10 * time.Second,
	}

	// indexActionCh capacity custom configure by conf.yaml need in [10, 10000]
	// or re-compile source to adjust min/max capacity
	capacity := conf.CacheIndexSetting.MaxUpdateQPS
	if capacity < 10 {
		capacity = 10
	} else if capacity > 10000 {
		capacity = 10000
	}
	cacheIndex.indexActionCh = make(chan *core.IndexAction, capacity)
	cacheIndex.cachePostsCh = make(chan *postsEntry, capacity)

	// 启动索引更新器
	go cacheIndex.startIndexPosts()

	return cacheIndex, cacheIndex
}

func NewSimpleCacheIndexService(indexPosts core.IndexPostsService) (core.CacheIndexService, core.VersionInfo) {
	s := conf.SimpleCacheIndexSetting
	cacheIndex := &simpleCacheIndexServant{
		ips:             indexPosts,
		maxIndexSize:    s.MaxIndexSize,
		indexPosts:      nil,
		checkTick:       time.NewTicker(s.CheckTickDuration), // check whether need update index every 1 minute
		expireIndexTick: time.NewTicker(time.Second),
	}

	// force expire index every ExpireTickDuration second
	if s.ExpireTickDuration != 0 {
		cacheIndex.expireIndexTick.Reset(s.CheckTickDuration)
	} else {
		cacheIndex.expireIndexTick.Stop()
	}

	// indexActionCh capacity custom configure by conf.yaml need in [10, 10000]
	// or re-compile source to adjust min/max capacity
	capacity := conf.CacheIndexSetting.MaxUpdateQPS
	if capacity < 10 {
		capacity = 10
	} else if capacity > 10000 {
		capacity = 10000
	}
	cacheIndex.indexActionCh = make(chan core.IdxAct, capacity)

	// start index posts
	cacheIndex.atomicIndex.Store(cacheIndex.indexPosts)
	go cacheIndex.startIndexPosts()

	return cacheIndex, cacheIndex
}

func NewNoneCacheIndexService(indexPosts core.IndexPostsService) (core.CacheIndexService, core.VersionInfo) {
	obj := &noneCacheIndexServant{
		ips: indexPosts,
	}
	return obj, obj
}
