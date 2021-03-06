// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package app

import (
	"fmt"
	"time"

	l4g "github.com/alecthomas/log4go"
	"github.com/mattermost/mattermost-server/model"
)

const (
	DISCOVERY_SERVICE_WRITE_PING = 60 * time.Second
)

type ClusterDiscoveryService struct {
	model.ClusterDiscovery
	stop chan bool
}

func NewClusterDiscoveryService() *ClusterDiscoveryService {
	ds := &ClusterDiscoveryService{
		ClusterDiscovery: model.ClusterDiscovery{},
		stop:             make(chan bool),
	}

	return ds
}

func (me *ClusterDiscoveryService) Start() {

	<-Global().Srv.Store.ClusterDiscovery().Cleanup()

	if cresult := <-Global().Srv.Store.ClusterDiscovery().Exists(&me.ClusterDiscovery); cresult.Err != nil {
		l4g.Error(fmt.Sprintf("ClusterDiscoveryService failed to check if row exists for %v with err=%v", me.ClusterDiscovery.ToJson(), cresult.Err))
	} else {
		if cresult.Data.(bool) {
			if u := <-Global().Srv.Store.ClusterDiscovery().Delete(&me.ClusterDiscovery); u.Err != nil {
				l4g.Error(fmt.Sprintf("ClusterDiscoveryService failed to start clean for %v with err=%v", me.ClusterDiscovery.ToJson(), u.Err))
			}
		}
	}

	if result := <-Global().Srv.Store.ClusterDiscovery().Save(&me.ClusterDiscovery); result.Err != nil {
		l4g.Error(fmt.Sprintf("ClusterDiscoveryService failed to save for %v with err=%v", me.ClusterDiscovery.ToJson(), result.Err))
		return
	}

	go func() {
		l4g.Debug(fmt.Sprintf("ClusterDiscoveryService ping writer started for %v", me.ClusterDiscovery.ToJson()))
		ticker := time.NewTicker(DISCOVERY_SERVICE_WRITE_PING)
		defer func() {
			ticker.Stop()
			if u := <-Global().Srv.Store.ClusterDiscovery().Delete(&me.ClusterDiscovery); u.Err != nil {
				l4g.Error(fmt.Sprintf("ClusterDiscoveryService failed to cleanup for %v with err=%v", me.ClusterDiscovery.ToJson(), u.Err))
			}
			l4g.Debug(fmt.Sprintf("ClusterDiscoveryService ping writer stopped for %v", me.ClusterDiscovery.ToJson()))
		}()

		for {
			select {
			case <-ticker.C:
				if u := <-Global().Srv.Store.ClusterDiscovery().SetLastPingAt(&me.ClusterDiscovery); u.Err != nil {
					l4g.Error(fmt.Sprintf("ClusterDiscoveryService failed to write ping for %v with err=%v", me.ClusterDiscovery.ToJson(), u.Err))
				}
			case <-me.stop:
				return
			}
		}
	}()
}

func (me *ClusterDiscoveryService) Stop() {
	me.stop <- true
}
