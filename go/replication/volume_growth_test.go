package replication

import (
	"code.google.com/p/weed-fs/go/sequence"
	"code.google.com/p/weed-fs/go/storage"
	"code.google.com/p/weed-fs/go/topology"
	"encoding/json"
	"fmt"
	"testing"
)

var topologyLayout = `
{
  "dc1":{
    "rack1":{
      "server111":{
        "volumes":[
          {"id":1, "size":12312},
          {"id":2, "size":12312},
          {"id":3, "size":12312}
        ],
        "limit":3
      },
      "server112":{
        "volumes":[
          {"id":4, "size":12312},
          {"id":5, "size":12312},
          {"id":6, "size":12312}
        ],
        "limit":10
      }
    },
    "rack2":{
      "server121":{
        "volumes":[
          {"id":4, "size":12312},
          {"id":5, "size":12312},
          {"id":6, "size":12312}
        ],
        "limit":4
      },
      "server122":{
        "volumes":[],
        "limit":4
      },
      "server123":{
        "volumes":[
          {"id":2, "size":12312},
          {"id":3, "size":12312},
          {"id":4, "size":12312}
        ],
        "limit":5
      }
    }
  },
  "dc2":{
  },
  "dc3":{
    "rack2":{
      "server321":{
        "volumes":[
          {"id":1, "size":12312},
          {"id":3, "size":12312},
          {"id":5, "size":12312}
        ],
        "limit":4
      }
    }
  }
}
`

func setup(topologyLayout string) *topology.Topology {
	var data interface{}
	err := json.Unmarshal([]byte(topologyLayout), &data)
	if err != nil {
		fmt.Println("error:", err)
	}
	fmt.Println("data:", data)

	//need to connect all nodes first before server adding volumes
	topo, err := topology.NewTopology("weedfs", "/etc/weedfs/weedfs.conf",
		sequence.NewMemorySequencer(), 32*1024, 5)
	if err != nil {
		panic("error: " + err.Error())
	}
	mTopology := data.(map[string]interface{})
	for dcKey, dcValue := range mTopology {
		dc := topology.NewDataCenter(dcKey)
		dcMap := dcValue.(map[string]interface{})
		topo.LinkChildNode(dc)
		for rackKey, rackValue := range dcMap {
			rack := topology.NewRack(rackKey)
			rackMap := rackValue.(map[string]interface{})
			dc.LinkChildNode(rack)
			for serverKey, serverValue := range rackMap {
				server := topology.NewDataNode(serverKey)
				serverMap := serverValue.(map[string]interface{})
				rack.LinkChildNode(server)
				for _, v := range serverMap["volumes"].([]interface{}) {
					m := v.(map[string]interface{})
					vi := storage.VolumeInfo{
						Id:      storage.VolumeId(int64(m["id"].(float64))),
						Size:    uint64(m["size"].(float64)),
						Version: storage.CurrentVersion}
					server.AddOrUpdateVolume(vi)
				}
				server.UpAdjustMaxVolumeCountDelta(int(serverMap["limit"].(float64)))
			}
		}
	}

	return topo
}

func TestFindEmptySlotsForOneVolume(t *testing.T) {
	topo := setup(topologyLayout)
	vg := NewDefaultVolumeGrowth()
	rp, _ := storage.NewReplicaPlacementFromString("002")
	servers, err := vg.findEmptySlotsForOneVolume(topo, "dc1", rp)
	if err != nil {
		fmt.Println("finding empty slots error :", err)
		t.Fail()
	}
	for _, server := range servers {
		fmt.Println("assigned node :", server.Id())
	}
}
