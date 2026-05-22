package runtime

import (
	"mihomoTui/internal/cache"
	"mihomoTui/internal/event"
	"mihomoTui/internal/hotreload"
	"mihomoTui/internal/merge"
	"mihomoTui/internal/proxygroup"
	"mihomoTui/internal/subscription"
)

type State struct {
	EventBus      *event.Bus
	Cache         *cache.Manager
	SubManager    *subscription.Manager
	MergeEngine   *merge.Engine
	ProxyGroupMgr *proxygroup.Manager
	HotReloadMgr  *hotreload.Manager
}

func NewState(reloadClient *hotreload.Manager) (*State, error) {
	bus := event.NewBus()
	cacheMgr := cache.NewManager()
	subMgr := subscription.NewManager()
	mergeEngine := merge.NewEngine()
	pgMgr := proxygroup.NewManager()

	return &State{
		EventBus:      bus,
		Cache:         cacheMgr,
		SubManager:    subMgr,
		MergeEngine:   mergeEngine,
		ProxyGroupMgr: pgMgr,
		HotReloadMgr:  reloadClient,
	}, nil
}
