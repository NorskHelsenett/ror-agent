package resourceupdate

import (
	"os"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"unsafe"

	"github.com/NorskHelsenett/ror/pkg/helpers/resourcecache/resourcecachehashlist"
	"github.com/NorskHelsenett/ror/pkg/rlog"
)

const resourceCacheMemLogEnv = "ROR_RESOURCECACHE_MEMLOG"

func (rc *resourcecache) startMemoryLogging(intervalSeconds int) {
	if rc == nil || rc.scheduler == nil {
		return
	}
	if !resourceCacheMemLogEnabled() {
		return
	}
	_, _ = rc.scheduler.Every(intervalSeconds).Second().Tag("resourcecachememlog").Do(rc.logMemoryEstimate)
	rlog.Info("resource cache memory logging enabled",
		rlog.Any("interval_seconds", intervalSeconds),
		rlog.Any("env", resourceCacheMemLogEnv),
	)
}

func resourceCacheMemLogEnabled() bool {
	val, ok := os.LookupEnv(resourceCacheMemLogEnv)
	if !ok {
		return false
	}
	val = strings.TrimSpace(strings.ToLower(val))
	if val == "" {
		return false
	}
	if b, err := strconv.ParseBool(val); err == nil {
		return b
	}
	return val == "1" || val == "yes" || val == "y" || val == "on" || val == "enable" || val == "enabled"
}

func (rc *resourcecache) logMemoryEstimate() {
	if rc == nil {
		return
	}

	hashItems := len(rc.HashList.Items)
	workItems := rc.Workqueue.ItemCount()

	hashEstimate := estimateHashListBytes(rc.HashList)
	workEstimate := estimateWorkqueueBytes(rc.Workqueue)
	estimated := hashEstimate + workEstimate

	delta := int64(estimated) - int64(rc.memLogLastEstimateBytes)
	rc.memLogLastEstimateBytes = estimated

	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	rlog.Info("resource cache memory estimate (approx)",
		rlog.Any("hash_items", hashItems),
		rlog.Any("workqueue_items", workItems),
		rlog.Any("estimated_bytes", estimated),
		rlog.Any("estimated_delta_bytes", delta),
		rlog.Any("heap_alloc_bytes", ms.HeapAlloc),
		rlog.Any("total_alloc_bytes", ms.TotalAlloc),
		rlog.Any("sys_bytes", ms.Sys),
		rlog.Any("next_gc_bytes", ms.NextGC),
		rlog.Any("num_gc", ms.NumGC),
		rlog.Any("pause_total_ns", ms.PauseTotalNs),
		rlog.Any("last_gc_unix_ns", ms.LastGC),
		rlog.Any("heap_objects", ms.HeapObjects),
	)
}

// estimateHashListBytes returns an approximate deep-ish size of the hash list.
// It counts slice backing array + string contents for Uid/Hash.
func estimateHashListBytes(hl resourcecachehashlist.HashList) uint64 {
	items := hl.Items
	var total uintptr

	// slice backing array
	total += uintptr(cap(items)) * unsafe.Sizeof(resourcecachehashlist.HashItem{})

	// string contents
	for _, it := range items {
		total += uintptr(len(it.Uid))
		total += uintptr(len(it.Hash))
	}

	return uint64(total)
}

// estimateWorkqueueBytes returns an approximate size of the workqueue.
// It counts slice backing array + selected string fields in ResourceUpdateModel.
// It intentionally does NOT attempt to size the Resource payload deeply (can be expensive).
func estimateWorkqueueBytes(wq ResourceCacheWorkqueue) uint64 {
	var total uintptr

	// slice backing array
	total += uintptr(cap(wq)) * unsafe.Sizeof(ResourceCacheWorkqueueObject{})

	for _, obj := range wq {
		// Uid is used by workqueue logic; account for it even if ResourceUpdate is nil.
		if obj.ResourceUpdate == nil {
			continue
		}

		// Count string fields on the top-level struct (ApiVersion, Kind, Uid, Hash, etc)
		total += estimateTopLevelStringFieldsBytes(obj.ResourceUpdate)
	}

	return uint64(total)
}

func estimateTopLevelStringFieldsBytes(ptr any) uintptr {
	if ptr == nil {
		return 0
	}
	val := reflect.ValueOf(ptr)
	if val.Kind() == reflect.Pointer {
		if val.IsNil() {
			return 0
		}
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return 0
	}

	var total uintptr
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		if field.Kind() == reflect.String {
			total += uintptr(len(field.String()))
		}
	}
	return total
}
