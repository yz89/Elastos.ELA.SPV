package bloom

import (
	"crypto/rand"
	"fmt"
	"os"
	"testing"

	. "github.com/elastos/Elastos.ELA.SPV/common"
	"github.com/elastos/Elastos.ELA.SPV/core/auxpow"
	"github.com/elastos/Elastos.ELA.SPV/core"
)

func TestMerkleBlock_GetTxMerkleBranch(t *testing.T) {
	for txs := uint32(1); txs < 1<<10; txs++ {
		run(txs)
		fmt.Println("GetTxMerkleBranch() with txs:", txs, "PASSED")
	}
}

func run(txs uint32) {
	mBlock := MBlock{
		NumTx:       txs,
		AllHashes:   make([]*Uint256, 0, txs),
		MatchedBits: make([]byte, 0, txs),
	}

	matches := randMatches(txs)
	for i := uint32(0); i < txs; i++ {
		if matches[i] {
			mBlock.MatchedBits = append(mBlock.MatchedBits, 0x01)
		} else {
			mBlock.MatchedBits = append(mBlock.MatchedBits, 0x00)
		}
		mBlock.AllHashes = append(mBlock.AllHashes, randHash())
	}

	// Calculate the number of merkle branches (height) in the tree.
	height := uint32(0)
	for mBlock.CalcTreeWidth(height) > 1 {
		height++
	}

	// Build the depth-first partial merkle tree.
	mBlock.TraverseAndBuild(height, 0)

	merkleRoot := *mBlock.CalcHash(treeDepth(txs), 0)
	// Create and return the merkle block.
	merkleBlock := MerkleBlock{
		BlockHeader: core.Header{
			MerkleRoot: merkleRoot,
		},
		Transactions: mBlock.NumTx,
		Hashes:       make([]*Uint256, 0, len(mBlock.FinalHashes)),
		Flags:        make([]byte, (len(mBlock.Bits)+7)/8),
	}
	for _, hash := range mBlock.FinalHashes {
		merkleBlock.Hashes = append(merkleBlock.Hashes, hash)
	}
	for i := uint32(0); i < uint32(len(mBlock.Bits)); i++ {
		merkleBlock.Flags[i/8] |= mBlock.Bits[i] << (i % 8)
	}

	txIds, err := CheckMerkleBlock(merkleBlock)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(0)
	}

	for i := range txIds {
		mb, err := merkleBlock.GetTxMerkleBranch(txIds[i])
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(0)
		}

		calcRoot := auxpow.GetMerkleRoot(*txIds[i], mb.Branches, mb.Index)
		if merkleRoot == calcRoot {
		} else {
			fmt.Println("Merkle root not match, expect %s result %s",
				merkleRoot.String(), calcRoot.String())
			os.Exit(0)
		}
	}
}

func randHash() *Uint256 {
	var hash Uint256
	rand.Read(hash[:])
	return &hash
}

func randMatches(hashes uint32) map[uint32]bool {
	matches := make(map[uint32]bool)
	b := make([]byte, 1)
	for i := uint32(0); i < hashes; i++ {
		rand.Read(b)
		matches[i] = b[0]%2 == 1
	}
	return matches
}
