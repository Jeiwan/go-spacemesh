package sync

import (
	"fmt"
	"github.com/spacemeshos/go-spacemesh/log"
	"github.com/spacemeshos/go-spacemesh/mesh"
	"github.com/spacemeshos/go-spacemesh/p2p/server"
	"github.com/spacemeshos/go-spacemesh/p2p/service"
	"testing"
	"time"
)

type PeersMock struct {
	getPeers func() []Peer
}

func (pm PeersMock) GetPeers() []Peer {
	return pm.getPeers()
}

func (pm PeersMock) Close() {
	return
}
func ListenerFactory(serv server.Service, peers Peers, name string) *BlockListener {
	nbl := NewBlockListener(serv, BlockValidatorMock{}, getMesh("TestBlockListener_"+name), 1*time.Second, 2, log.New(name, "", ""))
	nbl.Peers = peers //override peers with mock
	return nbl
}

func TestBlockListener(t *testing.T) {

	fmt.Println("test sync start")
	sim := service.NewSimulator()
	n1 := sim.NewNode()
	n2 := sim.NewNode()
	bl1 := ListenerFactory(n1, PeersMock{func() []Peer { return []Peer{n2.PublicKey()} }}, "1")
	bl2 := ListenerFactory(n2, PeersMock{func() []Peer { return []Peer{n1.PublicKey()} }}, "2")
	bl2.Start()

	block1 := mesh.NewExistingBlock(mesh.BlockID(123), 0, nil)
	block2 := mesh.NewExistingBlock(mesh.BlockID(321), 1, nil)
	block3 := mesh.NewExistingBlock(mesh.BlockID(222), 2, nil)

	block1.ViewEdges[block2.ID()] = struct{}{}
	block1.ViewEdges[block3.ID()] = struct{}{}

	bl1.AddBlock(block1)
	bl1.AddBlock(block2)
	bl1.AddBlock(block3)

	bl2.FetchBlock(block1.Id)
	timeout := time.After(30 * time.Second)
loop:
	for {
		select {
		// Got a timeout! fail with a timeout error
		case <-timeout:
			t.Error("timed out ")
		default:
			if b, err := bl2.GetBlock(block1.Id); err == nil {
				fmt.Println("  ", b)
				t.Log("done!")
				break loop
			}
		}
	}
}

func TestBlockListener2(t *testing.T) {

	fmt.Println("test sync start")
	sim := service.NewSimulator()
	n1 := sim.NewNode()
	n2 := sim.NewNode()

	bl1 := ListenerFactory(n1, PeersMock{func() []Peer { return []Peer{n2.PublicKey()} }}, "3")
	bl2 := ListenerFactory(n2, PeersMock{func() []Peer { return []Peer{n1.PublicKey()} }}, "4")

	bl2.Start()

	block1 := mesh.NewBlock(true, nil, time.Now(), 0)
	block2 := mesh.NewBlock(true, nil, time.Now(), 1)
	block3 := mesh.NewBlock(true, nil, time.Now(), 2)
	block4 := mesh.NewBlock(true, nil, time.Now(), 2)
	block5 := mesh.NewBlock(true, nil, time.Now(), 3)
	block6 := mesh.NewBlock(true, nil, time.Now(), 3)
	block7 := mesh.NewBlock(true, nil, time.Now(), 4)
	block8 := mesh.NewBlock(true, nil, time.Now(), 4)
	block9 := mesh.NewBlock(true, nil, time.Now(), 4)
	block10 := mesh.NewBlock(true, nil, time.Now(), 5)

	block2.ViewEdges[block1.ID()] = struct{}{}
	block3.ViewEdges[block2.ID()] = struct{}{}
	block4.ViewEdges[block2.ID()] = struct{}{}
	block5.ViewEdges[block3.ID()] = struct{}{}
	block5.ViewEdges[block4.ID()] = struct{}{}
	block6.ViewEdges[block4.ID()] = struct{}{}
	block7.ViewEdges[block6.ID()] = struct{}{}
	block7.ViewEdges[block5.ID()] = struct{}{}
	block8.ViewEdges[block6.ID()] = struct{}{}
	block9.ViewEdges[block5.ID()] = struct{}{}
	block10.ViewEdges[block8.ID()] = struct{}{}
	block10.ViewEdges[block9.ID()] = struct{}{}

	bl1.AddBlock(block1)
	bl1.AddBlock(block2)
	bl1.AddBlock(block3)
	bl1.AddBlock(block4)
	bl1.AddBlock(block5)
	bl1.AddBlock(block6)
	bl1.AddBlock(block7)
	bl1.AddBlock(block8)
	bl1.AddBlock(block9)
	bl1.AddBlock(block10)

	bl2.FetchBlock(block10.Id)

	timeout := time.After(10 * time.Second)
	for {
		select {
		// Got a timeout! fail with a timeout error
		case <-timeout:
			t.Error("timed out ")
			return
		default:
			if b, err := bl2.GetBlock(block1.Id); err == nil {
				fmt.Println("  ", b)
				t.Log("done!")
				return
			}
		}
	}
}

//todo integration testing
