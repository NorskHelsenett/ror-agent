package resourceupdate

import (
	"fmt"
	"runtime"
	"time"

	"github.com/NorskHelsenett/ror-agent/internal/services/authservice"
	"github.com/NorskHelsenett/ror-agent/pkg/clients/clusteragentclient"

	"github.com/NorskHelsenett/ror/pkg/apicontracts/apiresourcecontracts"

	"github.com/NorskHelsenett/ror/pkg/helpers/resourcecache/resourcecachehashlist"
	"github.com/NorskHelsenett/ror/pkg/rlog"

	"github.com/NorskHelsenett/ror/pkg/helpers/stringhelper"

	"github.com/go-co-op/gocron"
)

var ResourceCache resourcecache

type resourcecache struct {
	client                  clusteragentclient.RorAgentClientInterface
	HashList                resourcecachehashlist.HashList
	Workqueue               ResourceCacheWorkqueue
	cleanupRunning          bool
	scheduler               *gocron.Scheduler
	memLogLastEstimateBytes uint64
}

func (rc *resourcecache) MustInit(client clusteragentclient.RorAgentClientInterface) {
	var err error
	if client == nil {
		rc.client, err = clusteragentclient.NewRorAgentClient(clusteragentclient.GetDefaultRorAgentClientConfig())
		if err != nil {
			rlog.Fatal("failed to initialize cluster agent client for resource cache", err)
		}
	} else {
		rc.client = client
	}
	rc.HashList, err = rc.client.GetRorClient().V1().Resources().GetHashList(rc.client.GetRorClient().GetOwnerref())
	if err != nil {
		rlog.Fatal("could not get hashlist for clusterid", err)
	}
	rlog.Info("got hashList from ror-api", rlog.Int("length", len(rc.HashList.Items)))

	rc.scheduler = gocron.NewScheduler(time.Local)
	rc.scheduler.StartAsync()
	rc.addWorkqueScheduler(10)
	rc.startCleanup()
}

func (rc resourcecache) CleanupRunning() bool {
	return rc.cleanupRunning
}
func (rc *resourcecache) MarkActive(uid string) {
	rc.HashList.MarkActive(uid)
}

func (rc resourcecache) addWorkqueScheduler(seconds int) {
	_, _ = rc.scheduler.Every(seconds).Second().Tag("workquerunner").Do(rc.runWorkqueScheduler)
}
func (rc resourcecache) runWorkqueScheduler() {
	if rc.Workqueue.NeedToRun() {
		rlog.Warn("resourceQue has non zero length", rlog.Int("resource que length", rc.Workqueue.ItemCount()))
		rc.RunWorkQue()
	}
}

func (rc *resourcecache) startCleanup() {
	rc.cleanupRunning = true
	runAt := time.Now().Add(1 * time.Minute)
	_, err := rc.scheduler.Every(1).Day().At(runAt.Format("15:04:05")).LimitRunsTo(1).Tag("resourcescleanup").Do(rc.finnishCleanup)
	if err != nil {
		rlog.Error("failed scheduling resource cleanup", err, rlog.Any("tag", "resourcescleanup"), rlog.Any("run_at", runAt))
		return
	}
	rlog.Info("scheduled resource cleanup", rlog.Any("tag", "resourcescleanup"), rlog.Any("run_at", runAt))
}

func (rc *resourcecache) finnishCleanup() {
	if !rc.cleanupRunning {
		return
	}
	rc.cleanupRunning = false
	_ = rc.scheduler.RemoveByTag("resourcescleanup")
	inactive := rc.HashList.GetInactiveUid()
	rlog.Info("resource cleanup running", rlog.Int("inactive_count", len(inactive)))
	if len(inactive) == 0 {
		runtime.GC()
		return
	}
	for _, uid := range inactive {
		rlog.Debug(fmt.Sprintf("Removing resource %s", uid))
		resource := apiresourcecontracts.ResourceUpdateModel{
			Owner:  authservice.CreateOwnerref(),
			Uid:    uid,
			Action: apiresourcecontracts.K8sActionDelete,
		}
		_ = rc.sendResourceUpdateToRor(&resource)
	}
	rlog.Info(fmt.Sprintf("resource cleanup done, %d resources removed", len(inactive)))
	runtime.GC()
}

func (rc resourcecache) PrettyPrintHashes() {
	stringhelper.PrettyprintStruct(rc.HashList)
}

// RunWorkQue Will run from the scheduler if the resource-que is non zero length.
// Resources in the que wil be requed using the sendResourceUpdateToRor function.
func (rc *resourcecache) RunWorkQue() {
	for _, resourceReturn := range rc.Workqueue {
		err := rc.sendResourceUpdateToRor(resourceReturn.ResourceUpdate)
		if err != nil {
			rlog.Error("error re-sending resource update to ror, added to retryque", err)
			rc.Workqueue.Add(resourceReturn.ResourceUpdate)
			return
		}
		rc.Workqueue.DeleteByUid(resourceReturn.ResourceUpdate.Uid)
		rc.HashList.UpdateHash(resourceReturn.ResourceUpdate.Uid, resourceReturn.ResourceUpdate.Hash)
	}
}

// the function sends the resource to the ror api. If receiving a non 2xx statuscode it will retun an error.
func (rc *resourcecache) sendResourceUpdateToRor(resourceUpdate *apiresourcecontracts.ResourceUpdateModel) error {
	rorClient := rc.client.GetRorClient()
	var err error

	switch resourceUpdate.Action {
	case apiresourcecontracts.K8sActionUpdate:
		err = rorClient.V1().Resources().Update(resourceUpdate)
	case apiresourcecontracts.K8sActionAdd:
		err = rorClient.V1().Resources().Create(resourceUpdate)
	case apiresourcecontracts.K8sActionDelete:
		err = rorClient.V1().Resources().Delete(resourceUpdate.Uid)
	default:
		rlog.Error("Not implemented", nil)

	}
	if err != nil {
		return err
	}
	return nil
}
