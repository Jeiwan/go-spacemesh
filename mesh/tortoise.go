package mesh

import (
	"fmt"
	"github.com/golang-collections/go-datastructures/bitarray"
	"github.com/spacemeshos/go-spacemesh/log"
)

type LayerQueue chan *Layer
type NewIdQueue chan uint32

type BlockPosition struct {
	visibility bitarray.BitArray
	layer      LayerID
}

type Algorithm struct {
	block2Id          map[BlockID]uint32
	allBlocks         map[BlockID]*TortoiseBlock
	layerQueue        LayerQueue
	idQueue           NewIdQueue
	posVotes          []bitarray.BitArray
	visibilityMap     [20000]BlockPosition
	layers            map[LayerID]*Layer
	layerSize         uint32
	cachedLayers      uint32
	remainingBlockIds uint32
	totalBlocks       uint32
}

func NewAlgorithm(layerSize uint32, cachedLayers uint32) Algorithm {
	totBlocks := layerSize * cachedLayers
	trtl := Algorithm{
		block2Id:          make(map[BlockID]uint32),
		allBlocks:         make(map[BlockID]*TortoiseBlock),
		layerQueue:        make(LayerQueue, cachedLayers+1),
		idQueue:           make(NewIdQueue, layerSize),
		remainingBlockIds: totBlocks,
		totalBlocks:       totBlocks,
		posVotes:          make([]bitarray.BitArray, totBlocks),
		//visibilityMap:     make([20000]BlockPosition),
		layers:       make(map[LayerID]*Layer),
		layerSize:    layerSize,
		cachedLayers: cachedLayers,
	}
	return trtl
}

func (alg *Algorithm) GlobalVotingAvg() uint64 {
	return 100
}

func (alg *Algorithm) LayerVotingAvg() uint64 {
	return 30
}

func (alg *Algorithm) IsTortoiseValid(originBlock *TortoiseBlock, targetBlock BlockID, targetBlockIdx uint64, visibleBlocks bitarray.BitArray) bool {
	voteFor, voteAgainst := alg.countTotalVotesForBlock(targetBlockIdx, visibleBlocks)

	if voteFor > alg.GlobalVotingAvg() {
		return true
	}
	if voteAgainst > alg.GlobalVotingAvg() {
		return false
	}

	voteFor, voteAgainst = alg.CountVotesInLastLayer(alg.allBlocks[targetBlock]) //??

	if voteFor > alg.LayerVotingAvg() {
		return true
	}
	if voteAgainst > alg.LayerVotingAvg() {
		return false
	}

	return originBlock.Coin
}

func (alg *Algorithm) getLayerById(layerId LayerID) (*Layer, error) {
	if _, ok := alg.layers[layerId]; !ok {
		return nil, fmt.Errorf("layer Id not found %v", layerId)
	}
	return alg.layers[layerId], nil
}

func (alg *Algorithm) CountVotesInLastLayer(block *TortoiseBlock) (uint64, uint64) {
	return block.ConVotes, block.ProVotes
}

func (alg *Algorithm) createBlockVotingMap(origin *TortoiseBlock) (*bitarray.BitArray, *bitarray.BitArray) {
	blockMap := bitarray.NewBitArray(uint64(alg.totalBlocks))
	visibilityMap := bitarray.NewBitArray(uint64(alg.totalBlocks))
	// Count direct voters
	for blockId, vote := range origin.BlockVotes { //todo: check for double votes
		//todo: assert that block exists
		targetBlockId := uint64(alg.block2Id[blockId])
		block := alg.allBlocks[blockId]
		visibilityMap.SetBit(targetBlockId)
		targetPosition := alg.visibilityMap[targetBlockId]
		visibilityMap = visibilityMap.Or(targetPosition.visibility)
		if vote {
			blockMap.SetBit(targetBlockId)
			block.ProVotes++
		} else {
			block.ConVotes++
		}
	}
	count := 0
	ln := len(origin.BlockVotes)
	// Go over all other blocks that exist and calculate the origin blocks votes for them
	for blockId, targetBlockIdx := range alg.block2Id {
		if count < ln {
			if _, ok := origin.BlockVotes[blockId]; ok {
				count++
				continue
			}
		}
		val, err := visibilityMap.GetBit(uint64(targetBlockIdx))
		if err != nil {
			return &blockMap, &visibilityMap //todo: put error
		}
		if val {
			if alg.IsTortoiseValid(origin, blockId, uint64(targetBlockIdx), visibilityMap) {
				blockMap.SetBit(uint64(targetBlockIdx))
			}
		}
	}
	return &blockMap, &visibilityMap
}

func (alg *Algorithm) countTotalVotesForBlock(targetIdx uint64, visibleBlocks bitarray.BitArray) (uint64, uint64) {
	var posVotes, conVotes uint64 = 0, 0
	targetLayer := alg.visibilityMap[targetIdx].layer
	ln := len(alg.block2Id)
	for blockIdx := 0; blockIdx < ln; blockIdx++ { // possible bug what if there is an BlockId > len(alg.block2id)
		//if this block sees our
		//if alg.allBlocks[targetID].index
		blockPosition := &alg.visibilityMap[blockIdx]
		if blockPosition.layer <= targetLayer {
			continue
		}
		if val, err := visibleBlocks.GetBit(uint64(blockIdx)); val { //if this block is visible from our target
			if val, err = blockPosition.visibility.GetBit(targetIdx); err == nil && val {
				if set, _ := alg.posVotes[blockIdx].GetBit(targetIdx); set {
					posVotes++
				} else {
					conVotes++
				}
			}
		}

	}
	return posVotes, conVotes
}

func (alg *Algorithm) zeroBitColumn(idx uint64) {
	for row, bitvec := range alg.posVotes {
		bitvec.ClearBit(idx)
		alg.visibilityMap[row].visibility.ClearBit(idx)
	}
}

func (alg *Algorithm) recycleLayer(l *Layer) {
	for _, block := range l.blocks {
		id := alg.block2Id[block.Id]
		alg.idQueue <- alg.block2Id[block.Id]
		delete(alg.block2Id, block.Id)
		delete(alg.allBlocks, block.Id)
		alg.zeroBitColumn(uint64(id))
	}
	delete(alg.layers, l.index)
}

func (alg *Algorithm) assignIdForBlock(blk *TortoiseBlock) uint32 {
	//todo: should this section be protected by a mutex?
	alg.allBlocks[blk.Id] = blk
	if len(alg.idQueue) > 0 {
		id := <-alg.idQueue
		alg.block2Id[blk.Id] = id
		return id
	}
	if alg.remainingBlockIds > 0 {
		newId := alg.totalBlocks - alg.remainingBlockIds
		alg.block2Id[blk.Id] = newId
		alg.remainingBlockIds--

		return newId
	} else {
		log.Error("Cannot find Id for block, something went wrong")
		panic("Cannot find Id for block, something went wrong")
		return 0
	}

}

func (alg *Algorithm) HandleIncomingLayer(l *Layer) {
	alg.layers[l.index] = l
	alg.layerQueue <- l
	if len(alg.layerQueue) >= int(alg.cachedLayers) {
		layer := <-alg.layerQueue
		alg.recycleLayer(layer)
	}
	for _, originBlock := range l.blocks {
		//todo: what to do if block is invalid?
		votesBM, visibleBM := alg.createBlockVotingMap(originBlock)
		blockId := alg.assignIdForBlock(originBlock)
		alg.posVotes[blockId] = *votesBM
		alg.visibilityMap[blockId] = BlockPosition{*visibleBM, originBlock.Layer()}
	}
}

func (alg *Algorithm) HandleLateBlock(b *TortoiseBlock) {
	log.Info("received block with layer Id %v block id: %v ", b.Layer(), b.ID())
}
