package bloom

import (
	"bytes"
	"errors"
	"fmt"

	. "github.com/elastos/Elastos.ELA.SPV/common"
	"github.com/elastos/Elastos.ELA.SPV/common/serialization"
	"github.com/elastos/Elastos.ELA.SPV/core"
)

type MerkleBlock struct {
	BlockHeader  core.Header
	Transactions uint32
	Hashes       []*Uint256
	Flags        []byte
}

func (msg *MerkleBlock) CMD() string {
	return "merkleblock"
}

func (msg *MerkleBlock) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)
	err := msg.BlockHeader.Serialize(buf)
	if err != nil {
		return nil, err
	}

	err = serialization.WriteElements(buf,
		msg.Transactions,
		uint32(len(msg.Hashes)),
		msg.Hashes, msg.Flags)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (msg *MerkleBlock) Deserialize(body []byte) error {
	buf := bytes.NewReader(body)
	err := msg.BlockHeader.Deserialize(buf)
	if err != nil {
		return err
	}

	msg.Transactions, err = serialization.ReadUint32(buf)
	if err != nil {
		return err
	}

	hashes, err := serialization.ReadUint32(buf)
	if err != nil {
		return err
	}

	msg.Hashes = make([]*Uint256, hashes)
	return serialization.ReadElements(buf, &msg.Hashes, &msg.Flags)
}

func MakeMerkleParent(left *Uint256, right *Uint256) (*Uint256, error) {
	// dupes can screw things up; CVE-2012-2459. check for them
	if left != nil && right != nil && left.IsEqual(right) {
		return nil, errors.New("DUP HASH CRASH")
	}
	// if left child is nil, output nil.  Need this for hard mode.
	if left == nil {
		return nil, errors.New("Left child is nil")
	}
	// if right is nil, hash left with itself
	if right == nil {
		right = left
	}

	// Concatenate the left and right nodes
	var sha [64]byte
	copy(sha[:32], left[:])
	copy(sha[32:], right[:])

	parent := Uint256(Sha256D(sha[:]))
	return &parent, nil
}

type merkleNode struct {
	p uint32   // position in the binary tree
	h *Uint256 // hash
}

func (node merkleNode) String() string {
	return fmt.Sprint("Node{pos:", node.p, ", hash:", node.h, "}")
}

// given n merkle leaves, how deep is the tree?
// iterate shifting left until greater than n
func treeDepth(n uint32) (e uint32) {
	for ; (1 << e) < n; e++ {
	}
	return
}

// smallest power of 2 that can contain n
func nextPowerOfTwo(n uint32) uint32 {
	return 1 << treeDepth(n) // 2^exponent
}

// check if a node is populated based on node position and size of tree
func inDeadZone(pos, size uint32) bool {
	msb := nextPowerOfTwo(size)
	last := size - 1 // last valid position is 1 less than size
	if pos > (msb<<1)-2 { // greater than root; not even in the tree
		return true
	}
	h := msb
	for pos >= h {
		h = h>>1 | msb
		last = last>>1 | msb
	}
	return pos > last
}

// take in a merkle block, parse through it, and return txids indicated
// If there's any problem return an error.  Checks self-consistency only.
// doing it with a stack instead of recursion.  Because...
// OK I don't know why I'm just not in to recursion OK?
func CheckMerkleBlock(m MerkleBlock) ([]*Uint256, error) {
	if m.Transactions == 0 {
		return nil, fmt.Errorf("No transactions in merkleblock")
	}
	if len(m.Flags) == 0 {
		return nil, fmt.Errorf("No flag bits")
	}
	var s []merkleNode // the stack
	var r []*Uint256   // slice to return; txids we care about

	// set initial position to root of merkle tree
	msb := nextPowerOfTwo(m.Transactions) // most significant bit possible
	pos := (msb << 1) - 2                 // current position in tree

	var i uint8 // position in the current flag byte
	var tip int
	// main loop
	for {
		tip = len(s) - 1 // slice position of stack tip
		// First check if stack operations can be performed
		// is stack one filled item?  that's complete.
		if tip == 0 && s[0].h != nil {
			if s[0].h.IsEqual(&m.BlockHeader.MerkleRoot) {
				return r, nil
			}
			return nil, fmt.Errorf("computed root %s but expect %s\n",
				s[0].h.String(), m.BlockHeader.MerkleRoot.String())
		}
		// is current position in the tree's dead zone? partial parent
		if inDeadZone(pos, m.Transactions) {
			// create merkle parent from single side (left)
			h, err := MakeMerkleParent(s[tip].h, nil)
			if err != nil {
				return r, err
			}
			s[tip-1].h = h
			s = s[:tip]          // remove 1 from stack
			pos = s[tip-1].p | 1 // move position to parent's sibling
			continue
		}
		// does stack have 3+ items? and are last 2 items filled?
		if tip > 1 && s[tip-1].h != nil && s[tip].h != nil {
			//fmt.Printf("nodes %d and %d combine into %d\n",
			//	s[tip-1].p, s[tip].p, s[tip-2].p)
			// combine two filled nodes into parent node
			h, err := MakeMerkleParent(s[tip-1].h, s[tip].h)
			if err != nil {
				return r, err
			}
			s[tip-2].h = h
			// remove children
			s = s[:tip-1]
			// move position to parent's sibling
			pos = s[tip-2].p | 1
			continue
		}

		// no stack ops to perform, so make new node from message hashes
		if len(m.Hashes) == 0 {
			return nil, fmt.Errorf("Ran out of hashes at position %d.", pos)
		}
		if len(m.Flags) == 0 {
			return nil, fmt.Errorf("Ran out of flag bits.")
		}
		var n merkleNode // make new node
		n.p = pos        // set current position for new node

		if pos&msb != 0 { // upper non-txid hash
			if m.Flags[0]&(1<<i) == 0 { // flag bit says fill node
				n.h = m.Hashes[0]       // copy hash from message
				m.Hashes = m.Hashes[1:] // pop off message
				if pos&1 != 0 { // right side; ascend
					pos = pos>>1 | msb
				} else { // left side, go to sibling
					pos |= 1
				}
			} else { // flag bit says skip; put empty on stack and descend
				pos = (pos ^ msb) << 1 // descend to left
			}
			s = append(s, n) // push new node on stack
		} else { // bottom row txid; flag bit indicates tx of interest
			if pos >= m.Transactions {
				// this can't happen because we check deadzone above...
				return nil, fmt.Errorf("got into an invalid txid node")
			}
			n.h = m.Hashes[0]       // copy hash from message
			m.Hashes = m.Hashes[1:] // pop off message
			if m.Flags[0]&(1<<i) != 0 { //txid of interest
				r = append(r, n.h)
			}
			if pos&1 == 0 { // left side, go to sibling
				pos |= 1
			}                // if on right side we don't move; stack ops will move next
			s = append(s, n) // push new node onto the stack
		}

		// done with pushing onto stack; advance flag bit
		i++
		if i == 8 { // move to next byte
			i = 0
			m.Flags = m.Flags[1:]
		}
	}
	return nil, fmt.Errorf("ran out of things to do?")
}
