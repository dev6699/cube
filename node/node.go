package node

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/dev6699/cube/stats"
)

type Node struct {
	Name            string
	Ip              string
	Api             string
	Cores           int
	Memory          int64
	MemoryAllocated int64
	Disk            int64
	DiskAllocated   int64
	Stats           stats.Stats
	Role            string
	TaskCount       int
}

func New(name string, api string, role string) *Node {
	return &Node{
		Name: name,
		Api:  api,
		Role: role,
	}
}

func (n *Node) GetStats() (*stats.Stats, error) {
	url := fmt.Sprintf("%s/stats", n.Api)
	resp, err := httpWithRetry(http.Get, url, 10)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("[node] invalid sttaus code: %v", resp.StatusCode)
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var stats stats.Stats
	err = json.Unmarshal(body, &stats)
	if err != nil {
		return nil, err
	}

	if stats.MemStats == nil || stats.DiskStats == nil {
		return nil, fmt.Errorf("[node] error getting stats from node %s", n.Name)
	}

	n.Memory = int64(stats.MemTotalKb())
	n.Disk = int64(stats.DiskTotal())
	n.Stats = stats

	return &n.Stats, nil
}

func httpWithRetry(f func(string) (*http.Response, error), url string, count int) (*http.Response, error) {
	var resp *http.Response
	var err error
	for i := 0; i < count; i++ {
		resp, err = f(url)
		if err != nil {
			fmt.Printf("Error calling url %v\n", url)
			time.Sleep(5 * time.Second)
		} else {
			break
		}
	}
	return resp, err
}
