package task

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"time"

	dockerTypes "github.com/docker/docker/api/types"
	units "github.com/docker/go-units"
	lru "github.com/hashicorp/golang-lru"
	"github.com/ringo380/lessoncraft/docker"
	"github.com/ringo380/lessoncraft/event"
	"github.com/ringo380/lessoncraft/internal/circuitbreaker"
	"github.com/ringo380/lessoncraft/pwd/types"
	"github.com/ringo380/lessoncraft/router"
	"github.com/ringo380/lessoncraft/storage"
)

type InstanceStats struct {
	Instance string `json:"instance"`
	Mem      string `json:"mem"`
	Cpu      string `json:"cpu"`
}

type collectStats struct {
	event   event.EventApi
	factory docker.FactoryApi
	cli     *http.Client
	cache   *lru.Cache
	storage storage.StorageApi
}

var CollectStatsEvent event.EventType

func init() {
	CollectStatsEvent = event.EventType("instance stats")
}

func (t *collectStats) Name() string {
	return "CollectStats"
}

func (t *collectStats) Run(ctx context.Context, instance *types.Instance) error {
	if instance.Type == "windows" {
		host := router.EncodeHost(instance.SessionId, instance.IP, router.HostOpts{EncodedPort: 222})
		req, err := http.NewRequest("GET", fmt.Sprintf("http://%s/stats", host), nil)
		if err != nil {
			log.Printf("Could not create request to get stats of windows instance with IP %s. Got: %v\n", instance.IP, err)
			// Return a degraded response with default stats
			stats := InstanceStats{
				Instance: instance.Name,
				Mem:      "N/A (stats collection failed)",
				Cpu:      "N/A (stats collection failed)",
			}
			t.event.Emit(CollectStatsEvent, instance.SessionId, stats)
			return fmt.Errorf("Could not create request to get stats of windows instance with IP %s, using default stats: %v", instance.IP, err)
		}
		req.Header.Set("X-Proxy-Host", instance.SessionHost)
		resp, err := t.cli.Do(req)
		if err != nil {
			log.Printf("Could not get stats of windows instance with IP %s. Got: %v\n", instance.IP, err)
			// Check if this is a circuit breaker error
			if err.Error() == "circuit breaker is open: circuit breaker is open" {
				log.Printf("Stats collector circuit breaker is open for instance %s, using default stats", instance.Name)
			}
			// Return a degraded response with default stats
			stats := InstanceStats{
				Instance: instance.Name,
				Mem:      "N/A (stats collection failed)",
				Cpu:      "N/A (stats collection failed)",
			}
			t.event.Emit(CollectStatsEvent, instance.SessionId, stats)
			return fmt.Errorf("Could not get stats of windows instance with IP %s, using default stats: %v", instance.IP, err)
		}
		if resp.StatusCode != 200 {
			log.Printf("Could not get stats of windows instance with IP %s. Got status code: %d\n", instance.IP, resp.StatusCode)
			// Return a degraded response with default stats
			stats := InstanceStats{
				Instance: instance.Name,
				Mem:      "N/A (stats collection failed)",
				Cpu:      "N/A (stats collection failed)",
			}
			t.event.Emit(CollectStatsEvent, instance.SessionId, stats)
			return fmt.Errorf("Could not get stats of windows instance with IP %s, using default stats: status code %d", instance.IP, resp.StatusCode)
		}
		var info map[string]float64
		err = json.NewDecoder(resp.Body).Decode(&info)
		if err != nil {
			log.Printf("Could not get stats of windows instance with IP %s. Got: %v\n", instance.IP, err)
			// Return a degraded response with default stats
			stats := InstanceStats{
				Instance: instance.Name,
				Mem:      "N/A (stats collection failed)",
				Cpu:      "N/A (stats collection failed)",
			}
			t.event.Emit(CollectStatsEvent, instance.SessionId, stats)
			return fmt.Errorf("Could not decode stats of windows instance with IP %s, using default stats: %v", instance.IP, err)
		}
		stats := InstanceStats{Instance: instance.Name}

		stats.Mem = fmt.Sprintf("%.2f%% (%s / %s)", ((info["mem_used"] / info["mem_total"]) * 100), units.BytesSize(info["mem_used"]), units.BytesSize(info["mem_total"]))
		stats.Cpu = fmt.Sprintf("%.2f%%", info["cpu"]*100)
		t.event.Emit(CollectStatsEvent, instance.SessionId, stats)
		return nil
	}
	var session *types.Session
	if sess, found := t.cache.Get(instance.SessionId); !found {
		s, err := t.storage.SessionGet(instance.SessionId)
		if err != nil {
			log.Printf("Failed to get session %s: %v", instance.SessionId, err)
			// Return a degraded response with default stats
			stats := InstanceStats{
				Instance: instance.Name,
				Mem:      "N/A (stats collection failed)",
				Cpu:      "N/A (stats collection failed)",
			}
			t.event.Emit(CollectStatsEvent, instance.SessionId, stats)
			return fmt.Errorf("Failed to get session for stats collection, using default stats: %v", err)
		}
		t.cache.Add(s.Id, s)
		session = s
	} else {
		session = sess.(*types.Session)
	}
	dockerClient, err := t.factory.GetForSession(session)
	if err != nil {
		log.Printf("Failed to get Docker client for session %s: %v", session.Id, err)
		// Check if this is a circuit breaker error
		if err.Error() == "Docker daemon circuit breaker is open, too many failures detected" {
			log.Printf("Docker daemon circuit breaker is open for session %s, using default stats", session.Id)
		}
		// Return a degraded response with default stats
		stats := InstanceStats{
			Instance: instance.Name,
			Mem:      "N/A (stats collection failed)",
			Cpu:      "N/A (stats collection failed)",
		}
		t.event.Emit(CollectStatsEvent, instance.SessionId, stats)
		return fmt.Errorf("Failed to get Docker client for stats collection, using default stats: %v", err)
	}
	reader, err := dockerClient.ContainerStats(instance.Name)
	if err != nil {
		log.Printf("Error while trying to collect instance stats for %s: %v", instance.Name, err)
		// Return a degraded response with default stats
		stats := InstanceStats{
			Instance: instance.Name,
			Mem:      "N/A (stats collection failed)",
			Cpu:      "N/A (stats collection failed)",
		}
		t.event.Emit(CollectStatsEvent, instance.SessionId, stats)
		return fmt.Errorf("Failed to collect container stats, using default stats: %v", err)
	}
	dec := json.NewDecoder(reader)
	var v *dockerTypes.StatsJSON
	e := dec.Decode(&v)
	if e != nil {
		log.Printf("Error while trying to decode instance stats for %s: %v", instance.Name, e)
		// Return a degraded response with default stats
		stats := InstanceStats{
			Instance: instance.Name,
			Mem:      "N/A (stats collection failed)",
			Cpu:      "N/A (stats collection failed)",
		}
		t.event.Emit(CollectStatsEvent, instance.SessionId, stats)
		return fmt.Errorf("Failed to decode container stats, using default stats: %v", e)
	}
	stats := InstanceStats{Instance: instance.Name}
	// Memory
	var memPercent float64 = 0
	if v.MemoryStats.Limit != 0 {
		memPercent = float64(v.MemoryStats.Usage) / float64(v.MemoryStats.Limit) * 100.0
	}
	mem := float64(v.MemoryStats.Usage)
	memLimit := float64(v.MemoryStats.Limit)

	stats.Mem = fmt.Sprintf("%.2f%% (%s / %s)", memPercent, units.BytesSize(mem), units.BytesSize(memLimit))

	// cpu
	previousCPU := v.PreCPUStats.CPUUsage.TotalUsage
	previousSystem := v.PreCPUStats.SystemUsage
	cpuPercent := calculateCPUPercentUnix(previousCPU, previousSystem, v)
	stats.Cpu = fmt.Sprintf("%.2f%%", cpuPercent)

	t.event.Emit(CollectStatsEvent, instance.SessionId, stats)
	return nil
}

func proxyHost(r *http.Request) (*url.URL, error) {
	if r.Header.Get("X-Proxy-Host") == "" {
		return nil, nil
	}
	u := new(url.URL)
	*u = *r.URL
	u.Host = fmt.Sprintf("%s:8443", r.Header.Get("X-Proxy-Host"))
	return u, nil
}

func NewCollectStats(e event.EventApi, f docker.FactoryApi, s storage.StorageApi) *collectStats {
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   1 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConnsPerHost: 5,
		Proxy:               proxyHost,
	}

	// Wrap the transport with a circuit breaker
	cbTransport := circuitbreaker.WrapTransport(transport, circuitbreaker.Options{
		Name:                     "stats-collector",
		FailureThreshold:         3,
		ResetTimeout:             10 * time.Second,
		HalfOpenSuccessThreshold: 1,
		OnStateChange: func(name string, from, to circuitbreaker.State) {
			log.Printf("Stats collector circuit breaker state changed from %v to %v", from, to)
		},
	})

	cli := &http.Client{
		Transport: cbTransport,
	}
	c, _ := lru.New(5000)
	return &collectStats{event: e, factory: f, cli: cli, cache: c, storage: s}
}

func calculateCPUPercentUnix(previousCPU, previousSystem uint64, v *dockerTypes.StatsJSON) float64 {
	var (
		cpuPercent = 0.0
		// calculate the change for the cpu usage of the container in between readings
		cpuDelta = float64(v.CPUStats.CPUUsage.TotalUsage) - float64(previousCPU)
		// calculate the change for the entire system between readings
		systemDelta = float64(v.CPUStats.SystemUsage) - float64(previousSystem)
	)

	if systemDelta > 0.0 && cpuDelta > 0.0 {
		cpuPercent = (cpuDelta / systemDelta) * float64(len(v.CPUStats.CPUUsage.PercpuUsage)) * 100.0
	}
	return cpuPercent
}
