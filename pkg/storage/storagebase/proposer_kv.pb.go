// Code generated by protoc-gen-gogo.
// source: cockroach/pkg/storage/storagebase/proposer_kv.proto
// DO NOT EDIT!

/*
	Package storagebase is a generated protocol buffer package.

	It is generated from these files:
		cockroach/pkg/storage/storagebase/proposer_kv.proto
		cockroach/pkg/storage/storagebase/state.proto

	It has these top-level messages:
		RaftCommand
		Split
		Merge
		ReplicatedProposalData
		ReplicaState
		RangeInfo
*/
package storagebase

import proto "github.com/gogo/protobuf/proto"
import fmt "fmt"
import math "math"
import cockroach_roachpb3 "github.com/cockroachdb/cockroach/pkg/roachpb"
import cockroach_roachpb1 "github.com/cockroachdb/cockroach/pkg/roachpb"
import cockroach_roachpb "github.com/cockroachdb/cockroach/pkg/roachpb"
import cockroach_storage_engine_enginepb "github.com/cockroachdb/cockroach/pkg/storage/engine/enginepb"

import github_com_cockroachdb_cockroach_pkg_roachpb "github.com/cockroachdb/cockroach/pkg/roachpb"

import io "io"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion2 // please upgrade the proto package

// A RaftCommand is a command which can be serialized and sent via
// raft.
type RaftCommand struct {
	RangeID       github_com_cockroachdb_cockroach_pkg_roachpb.RangeID `protobuf:"varint,1,opt,name=range_id,json=rangeId,casttype=github.com/cockroachdb/cockroach/pkg/roachpb.RangeID" json:"range_id"`
	OriginReplica cockroach_roachpb.ReplicaDescriptor                  `protobuf:"bytes,2,opt,name=origin_replica,json=originReplica" json:"origin_replica"`
	Cmd           cockroach_roachpb3.BatchRequest                      `protobuf:"bytes,3,opt,name=cmd" json:"cmd"`
	// When the command is applied, its result is an error if the lease log
	// counter has already reached (or exceeded) max_lease_index.
	//
	// The lease index is a replay protection mechanism. Similar to the Raft
	// applied index, it is strictly increasing, but may have gaps. A command
	// will only apply successfully if its max_lease_index has not been surpassed
	// by the Range's applied lease index (in which case the command may need to
	// be 'refurbished', that is, regenerated with a higher max_lease_index).
	// When the command applies, the new lease index will increase to
	// max_lease_index (so a potential later replay will fail).
	//
	// Refurbishment is conditional on whether there is a difference between the
	// local pending and the applying version of the command - if the local copy
	// has a different max_lease_index, an earlier incarnation of the command has
	// already been refurbished, and no repeated refurbishment takes place.
	//
	// This mechanism was introduced as a simpler alternative to using the Raft
	// applied index, which is fraught with complexity due to the need to predict
	// exactly the log position at which a command will apply, even when the Raft
	// leader is not colocated with the lease holder (which usually proposes all
	// commands).
	//
	// Pinning the lease-index to the assigned slot (as opposed to allowing gaps
	// as we do now) is an interesting venue to explore from the standpoint of
	// parallelization: One could hope to enforce command ordering in that way
	// (without recourse to a higher-level locking primitive such as the command
	// queue). This is a hard problem: First of all, managing the pending
	// commands gets more involved; a command must not be removed if others have
	// been added after it, and on removal, the assignment counters must be
	// updated accordingly. Even worse though, refurbishments must be avoided at
	// all costs (since a refurbished command is likely to order after one that
	// it originally preceded (and which may well commit successfully without
	// a refurbishment).
	MaxLeaseIndex uint64 `protobuf:"varint,4,opt,name=max_lease_index,json=maxLeaseIndex" json:"max_lease_index"`
}

func (m *RaftCommand) Reset()                    { *m = RaftCommand{} }
func (m *RaftCommand) String() string            { return proto.CompactTextString(m) }
func (*RaftCommand) ProtoMessage()               {}
func (*RaftCommand) Descriptor() ([]byte, []int) { return fileDescriptorProposerKv, []int{0} }

// Split is emitted when a Replica commits a split trigger. It signals that the
// Replica has prepared the on-disk state for both the left and right hand
// sides of the split, and that the left hand side Replica should be updated as
// well as the right hand side created.
type Split struct {
	cockroach_roachpb1.SplitTrigger `protobuf:"bytes,1,opt,name=trigger,embedded=trigger" json:"trigger"`
	// RHSDelta holds the statistics for what was written to what is now the
	// right-hand side of the split during the batch which executed it.
	// The on-disk state of the right-hand side is already correct, but the
	// Store must learn about this delta to update its counters appropriately.
	RHSDelta cockroach_storage_engine_enginepb.MVCCStats `protobuf:"bytes,2,opt,name=rhs_delta,json=rhsDelta" json:"rhs_delta"`
}

func (m *Split) Reset()                    { *m = Split{} }
func (m *Split) String() string            { return proto.CompactTextString(m) }
func (*Split) ProtoMessage()               {}
func (*Split) Descriptor() ([]byte, []int) { return fileDescriptorProposerKv, []int{1} }

// Merge is emitted by a Replica which commits a transaction with
// a MergeTrigger (i.e. absorbs its right neighbor).
type Merge struct {
	cockroach_roachpb1.MergeTrigger `protobuf:"bytes,1,opt,name=trigger,embedded=trigger" json:"trigger"`
}

func (m *Merge) Reset()                    { *m = Merge{} }
func (m *Merge) String() string            { return proto.CompactTextString(m) }
func (*Merge) ProtoMessage()               {}
func (*Merge) Descriptor() ([]byte, []int) { return fileDescriptorProposerKv, []int{2} }

// ReplicaProposalData is the structured information which together with
// a RocksDB WriteBatch constitutes the proposal payload in proposer-evaluated
// KV. For the majority of proposals, we expect ReplicatedProposalData to be
// trivial; only changes to the metadata state (splits, merges, rebalances,
// leases, log truncation, ...) of the Replica or certain special commands must
// sideline information here based on which all Replicas must take action.
//
// TODO(tschottdorf): We may need to add a lease identifier to allow the
// followers to reliably produce errors for proposals which apply after a
// lease change.
type ReplicatedProposalData struct {
	// Whether to block concurrent readers while processing the proposal data.
	BlockReads bool `protobuf:"varint,1,opt,name=block_reads,json=blockReads" json:"block_reads"`
	// Updates to the Replica's ReplicaState. By convention and as outlined on
	// the comment on the ReplicaState message, this field is sparsely populated
	// and any field set overwrites the corresponding field in the state, perhaps
	// which additional side effects (for instance on a descriptor update).
	State ReplicaState `protobuf:"bytes,2,opt,name=state" json:"state"`
	Split *Split       `protobuf:"bytes,3,opt,name=split" json:"split,omitempty"`
	Merge *Merge       `protobuf:"bytes,4,opt,name=merge" json:"merge,omitempty"`
	// TODO(tschottdorf): trim this down; we shouldn't need the whole request.
	ComputeChecksum *cockroach_roachpb3.ComputeChecksumRequest `protobuf:"bytes,5,opt,name=compute_checksum,json=computeChecksum" json:"compute_checksum,omitempty"`
}

func (m *ReplicatedProposalData) Reset()                    { *m = ReplicatedProposalData{} }
func (m *ReplicatedProposalData) String() string            { return proto.CompactTextString(m) }
func (*ReplicatedProposalData) ProtoMessage()               {}
func (*ReplicatedProposalData) Descriptor() ([]byte, []int) { return fileDescriptorProposerKv, []int{3} }

func init() {
	proto.RegisterType((*RaftCommand)(nil), "cockroach.storage.storagebase.RaftCommand")
	proto.RegisterType((*Split)(nil), "cockroach.storage.storagebase.Split")
	proto.RegisterType((*Merge)(nil), "cockroach.storage.storagebase.Merge")
	proto.RegisterType((*ReplicatedProposalData)(nil), "cockroach.storage.storagebase.ReplicatedProposalData")
}
func (m *RaftCommand) Marshal() (data []byte, err error) {
	size := m.Size()
	data = make([]byte, size)
	n, err := m.MarshalTo(data)
	if err != nil {
		return nil, err
	}
	return data[:n], nil
}

func (m *RaftCommand) MarshalTo(data []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	data[i] = 0x8
	i++
	i = encodeVarintProposerKv(data, i, uint64(m.RangeID))
	data[i] = 0x12
	i++
	i = encodeVarintProposerKv(data, i, uint64(m.OriginReplica.Size()))
	n1, err := m.OriginReplica.MarshalTo(data[i:])
	if err != nil {
		return 0, err
	}
	i += n1
	data[i] = 0x1a
	i++
	i = encodeVarintProposerKv(data, i, uint64(m.Cmd.Size()))
	n2, err := m.Cmd.MarshalTo(data[i:])
	if err != nil {
		return 0, err
	}
	i += n2
	data[i] = 0x20
	i++
	i = encodeVarintProposerKv(data, i, uint64(m.MaxLeaseIndex))
	return i, nil
}

func (m *Split) Marshal() (data []byte, err error) {
	size := m.Size()
	data = make([]byte, size)
	n, err := m.MarshalTo(data)
	if err != nil {
		return nil, err
	}
	return data[:n], nil
}

func (m *Split) MarshalTo(data []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	data[i] = 0xa
	i++
	i = encodeVarintProposerKv(data, i, uint64(m.SplitTrigger.Size()))
	n3, err := m.SplitTrigger.MarshalTo(data[i:])
	if err != nil {
		return 0, err
	}
	i += n3
	data[i] = 0x12
	i++
	i = encodeVarintProposerKv(data, i, uint64(m.RHSDelta.Size()))
	n4, err := m.RHSDelta.MarshalTo(data[i:])
	if err != nil {
		return 0, err
	}
	i += n4
	return i, nil
}

func (m *Merge) Marshal() (data []byte, err error) {
	size := m.Size()
	data = make([]byte, size)
	n, err := m.MarshalTo(data)
	if err != nil {
		return nil, err
	}
	return data[:n], nil
}

func (m *Merge) MarshalTo(data []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	data[i] = 0xa
	i++
	i = encodeVarintProposerKv(data, i, uint64(m.MergeTrigger.Size()))
	n5, err := m.MergeTrigger.MarshalTo(data[i:])
	if err != nil {
		return 0, err
	}
	i += n5
	return i, nil
}

func (m *ReplicatedProposalData) Marshal() (data []byte, err error) {
	size := m.Size()
	data = make([]byte, size)
	n, err := m.MarshalTo(data)
	if err != nil {
		return nil, err
	}
	return data[:n], nil
}

func (m *ReplicatedProposalData) MarshalTo(data []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	data[i] = 0x8
	i++
	if m.BlockReads {
		data[i] = 1
	} else {
		data[i] = 0
	}
	i++
	data[i] = 0x12
	i++
	i = encodeVarintProposerKv(data, i, uint64(m.State.Size()))
	n6, err := m.State.MarshalTo(data[i:])
	if err != nil {
		return 0, err
	}
	i += n6
	if m.Split != nil {
		data[i] = 0x1a
		i++
		i = encodeVarintProposerKv(data, i, uint64(m.Split.Size()))
		n7, err := m.Split.MarshalTo(data[i:])
		if err != nil {
			return 0, err
		}
		i += n7
	}
	if m.Merge != nil {
		data[i] = 0x22
		i++
		i = encodeVarintProposerKv(data, i, uint64(m.Merge.Size()))
		n8, err := m.Merge.MarshalTo(data[i:])
		if err != nil {
			return 0, err
		}
		i += n8
	}
	if m.ComputeChecksum != nil {
		data[i] = 0x2a
		i++
		i = encodeVarintProposerKv(data, i, uint64(m.ComputeChecksum.Size()))
		n9, err := m.ComputeChecksum.MarshalTo(data[i:])
		if err != nil {
			return 0, err
		}
		i += n9
	}
	return i, nil
}

func encodeFixed64ProposerKv(data []byte, offset int, v uint64) int {
	data[offset] = uint8(v)
	data[offset+1] = uint8(v >> 8)
	data[offset+2] = uint8(v >> 16)
	data[offset+3] = uint8(v >> 24)
	data[offset+4] = uint8(v >> 32)
	data[offset+5] = uint8(v >> 40)
	data[offset+6] = uint8(v >> 48)
	data[offset+7] = uint8(v >> 56)
	return offset + 8
}
func encodeFixed32ProposerKv(data []byte, offset int, v uint32) int {
	data[offset] = uint8(v)
	data[offset+1] = uint8(v >> 8)
	data[offset+2] = uint8(v >> 16)
	data[offset+3] = uint8(v >> 24)
	return offset + 4
}
func encodeVarintProposerKv(data []byte, offset int, v uint64) int {
	for v >= 1<<7 {
		data[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	data[offset] = uint8(v)
	return offset + 1
}
func (m *RaftCommand) Size() (n int) {
	var l int
	_ = l
	n += 1 + sovProposerKv(uint64(m.RangeID))
	l = m.OriginReplica.Size()
	n += 1 + l + sovProposerKv(uint64(l))
	l = m.Cmd.Size()
	n += 1 + l + sovProposerKv(uint64(l))
	n += 1 + sovProposerKv(uint64(m.MaxLeaseIndex))
	return n
}

func (m *Split) Size() (n int) {
	var l int
	_ = l
	l = m.SplitTrigger.Size()
	n += 1 + l + sovProposerKv(uint64(l))
	l = m.RHSDelta.Size()
	n += 1 + l + sovProposerKv(uint64(l))
	return n
}

func (m *Merge) Size() (n int) {
	var l int
	_ = l
	l = m.MergeTrigger.Size()
	n += 1 + l + sovProposerKv(uint64(l))
	return n
}

func (m *ReplicatedProposalData) Size() (n int) {
	var l int
	_ = l
	n += 2
	l = m.State.Size()
	n += 1 + l + sovProposerKv(uint64(l))
	if m.Split != nil {
		l = m.Split.Size()
		n += 1 + l + sovProposerKv(uint64(l))
	}
	if m.Merge != nil {
		l = m.Merge.Size()
		n += 1 + l + sovProposerKv(uint64(l))
	}
	if m.ComputeChecksum != nil {
		l = m.ComputeChecksum.Size()
		n += 1 + l + sovProposerKv(uint64(l))
	}
	return n
}

func sovProposerKv(x uint64) (n int) {
	for {
		n++
		x >>= 7
		if x == 0 {
			break
		}
	}
	return n
}
func sozProposerKv(x uint64) (n int) {
	return sovProposerKv(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *RaftCommand) Unmarshal(data []byte) error {
	l := len(data)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowProposerKv
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := data[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: RaftCommand: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: RaftCommand: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field RangeID", wireType)
			}
			m.RangeID = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowProposerKv
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				m.RangeID |= (github_com_cockroachdb_cockroach_pkg_roachpb.RangeID(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field OriginReplica", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowProposerKv
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				msglen |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthProposerKv
			}
			postIndex := iNdEx + msglen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.OriginReplica.Unmarshal(data[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Cmd", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowProposerKv
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				msglen |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthProposerKv
			}
			postIndex := iNdEx + msglen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.Cmd.Unmarshal(data[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 4:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field MaxLeaseIndex", wireType)
			}
			m.MaxLeaseIndex = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowProposerKv
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				m.MaxLeaseIndex |= (uint64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipProposerKv(data[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthProposerKv
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *Split) Unmarshal(data []byte) error {
	l := len(data)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowProposerKv
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := data[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: Split: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Split: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field SplitTrigger", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowProposerKv
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				msglen |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthProposerKv
			}
			postIndex := iNdEx + msglen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.SplitTrigger.Unmarshal(data[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field RHSDelta", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowProposerKv
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				msglen |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthProposerKv
			}
			postIndex := iNdEx + msglen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.RHSDelta.Unmarshal(data[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipProposerKv(data[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthProposerKv
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *Merge) Unmarshal(data []byte) error {
	l := len(data)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowProposerKv
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := data[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: Merge: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Merge: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field MergeTrigger", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowProposerKv
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				msglen |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthProposerKv
			}
			postIndex := iNdEx + msglen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.MergeTrigger.Unmarshal(data[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipProposerKv(data[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthProposerKv
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *ReplicatedProposalData) Unmarshal(data []byte) error {
	l := len(data)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowProposerKv
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := data[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: ReplicatedProposalData: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: ReplicatedProposalData: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field BlockReads", wireType)
			}
			var v int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowProposerKv
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				v |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			m.BlockReads = bool(v != 0)
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field State", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowProposerKv
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				msglen |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthProposerKv
			}
			postIndex := iNdEx + msglen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.State.Unmarshal(data[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Split", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowProposerKv
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				msglen |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthProposerKv
			}
			postIndex := iNdEx + msglen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.Split == nil {
				m.Split = &Split{}
			}
			if err := m.Split.Unmarshal(data[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Merge", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowProposerKv
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				msglen |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthProposerKv
			}
			postIndex := iNdEx + msglen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.Merge == nil {
				m.Merge = &Merge{}
			}
			if err := m.Merge.Unmarshal(data[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 5:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ComputeChecksum", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowProposerKv
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				msglen |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthProposerKv
			}
			postIndex := iNdEx + msglen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.ComputeChecksum == nil {
				m.ComputeChecksum = &cockroach_roachpb3.ComputeChecksumRequest{}
			}
			if err := m.ComputeChecksum.Unmarshal(data[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipProposerKv(data[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthProposerKv
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func skipProposerKv(data []byte) (n int, err error) {
	l := len(data)
	iNdEx := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowProposerKv
			}
			if iNdEx >= l {
				return 0, io.ErrUnexpectedEOF
			}
			b := data[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		wireType := int(wire & 0x7)
		switch wireType {
		case 0:
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowProposerKv
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if data[iNdEx-1] < 0x80 {
					break
				}
			}
			return iNdEx, nil
		case 1:
			iNdEx += 8
			return iNdEx, nil
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowProposerKv
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				length |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			iNdEx += length
			if length < 0 {
				return 0, ErrInvalidLengthProposerKv
			}
			return iNdEx, nil
		case 3:
			for {
				var innerWire uint64
				var start int = iNdEx
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return 0, ErrIntOverflowProposerKv
					}
					if iNdEx >= l {
						return 0, io.ErrUnexpectedEOF
					}
					b := data[iNdEx]
					iNdEx++
					innerWire |= (uint64(b) & 0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				innerWireType := int(innerWire & 0x7)
				if innerWireType == 4 {
					break
				}
				next, err := skipProposerKv(data[start:])
				if err != nil {
					return 0, err
				}
				iNdEx = start + next
			}
			return iNdEx, nil
		case 4:
			return iNdEx, nil
		case 5:
			iNdEx += 4
			return iNdEx, nil
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
	}
	panic("unreachable")
}

var (
	ErrInvalidLengthProposerKv = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowProposerKv   = fmt.Errorf("proto: integer overflow")
)

func init() {
	proto.RegisterFile("cockroach/pkg/storage/storagebase/proposer_kv.proto", fileDescriptorProposerKv)
}

var fileDescriptorProposerKv = []byte{
	// 593 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0x8c, 0x53, 0x41, 0x8b, 0xd3, 0x40,
	0x14, 0xde, 0xec, 0x6e, 0xd9, 0x3a, 0x45, 0x77, 0x09, 0x22, 0x65, 0xc1, 0xa4, 0x94, 0x2e, 0x54,
	0x5c, 0x13, 0x51, 0x41, 0xf0, 0x98, 0x14, 0xd6, 0xc2, 0x2e, 0xe8, 0x74, 0xf1, 0xa0, 0x87, 0x30,
	0x99, 0x8c, 0x49, 0x68, 0x92, 0x89, 0x33, 0xd3, 0xa5, 0x3f, 0xc3, 0xbf, 0x21, 0xfe, 0x91, 0x1e,
	0xf7, 0x24, 0x9e, 0x8a, 0xd6, 0x7f, 0xe1, 0x49, 0x66, 0x32, 0xa9, 0x5d, 0x8c, 0xd6, 0x53, 0x1e,
	0xef, 0x7d, 0xdf, 0x37, 0xef, 0xbd, 0xef, 0x05, 0x3c, 0xc5, 0x14, 0x4f, 0x19, 0x45, 0x38, 0x71,
	0xcb, 0x69, 0xec, 0x72, 0x41, 0x19, 0x8a, 0x49, 0xfd, 0x0d, 0x11, 0x27, 0x6e, 0xc9, 0x68, 0x49,
	0x39, 0x61, 0xc1, 0xf4, 0xca, 0x29, 0x19, 0x15, 0xd4, 0xbc, 0xbf, 0x26, 0x39, 0x1a, 0xe8, 0x6c,
	0x10, 0x8e, 0xed, 0x9b, 0x9a, 0x2a, 0x2a, 0x43, 0x17, 0x95, 0x69, 0xc5, 0x3f, 0xee, 0x35, 0x03,
	0x22, 0x24, 0x90, 0x46, 0x0c, 0x9a, 0x11, 0x39, 0x11, 0x68, 0x03, 0xf5, 0xb8, 0xb9, 0x79, 0x52,
	0xc4, 0x69, 0x51, 0x7f, 0x24, 0xeb, 0x0a, 0x63, 0xcd, 0x78, 0xb4, 0x7d, 0x5c, 0x2e, 0x90, 0x20,
	0x1a, 0x7e, 0x37, 0xa6, 0x31, 0x55, 0xa1, 0x2b, 0xa3, 0x2a, 0xdb, 0xff, 0xbc, 0x0b, 0x3a, 0x10,
	0xbd, 0x17, 0x3e, 0xcd, 0x73, 0x54, 0x44, 0x66, 0x08, 0xda, 0x0c, 0x15, 0x31, 0x09, 0xd2, 0xa8,
	0x6b, 0xf4, 0x8c, 0xe1, 0x9e, 0x77, 0xb6, 0x58, 0xda, 0x3b, 0xab, 0xa5, 0x7d, 0x00, 0x65, 0x7e,
	0x3c, 0xfa, 0xb9, 0xb4, 0x9f, 0xc5, 0xa9, 0x48, 0x66, 0xa1, 0x83, 0x69, 0xee, 0xae, 0x9b, 0x88,
	0x42, 0xb7, 0x71, 0x50, 0x47, 0xf3, 0xe0, 0x81, 0x12, 0x1e, 0x47, 0xe6, 0x6b, 0x70, 0x87, 0xb2,
	0x34, 0x4e, 0x8b, 0x80, 0x91, 0x32, 0x4b, 0x31, 0xea, 0xee, 0xf6, 0x8c, 0x61, 0xe7, 0xc9, 0xc0,
	0xf9, 0xed, 0xc5, 0x9a, 0x5c, 0x21, 0x46, 0x84, 0x63, 0x96, 0x96, 0x82, 0x32, 0x6f, 0x5f, 0xf6,
	0x03, 0x6f, 0x57, 0x0a, 0xba, 0x6c, 0x3e, 0x07, 0x7b, 0x38, 0x8f, 0xba, 0x7b, 0x4a, 0xc7, 0x6e,
	0xd0, 0xf1, 0x90, 0xc0, 0x09, 0x24, 0x1f, 0x66, 0x84, 0x0b, 0x2d, 0x21, 0x19, 0xe6, 0x29, 0x38,
	0xcc, 0xd1, 0x3c, 0xc8, 0x08, 0xe2, 0x24, 0x48, 0x8b, 0x88, 0xcc, 0xbb, 0xfb, 0x3d, 0x63, 0xb8,
	0x5f, 0x3f, 0x93, 0xa3, 0xf9, 0xb9, 0xac, 0x8d, 0x65, 0xa9, 0xff, 0xc9, 0x00, 0xad, 0x49, 0x99,
	0xa5, 0xc2, 0xf4, 0xc1, 0x81, 0x60, 0x69, 0x1c, 0x13, 0xa6, 0xd6, 0xd4, 0xfc, 0xa8, 0x82, 0x5e,
	0x56, 0x30, 0xaf, 0x2d, 0x05, 0xaf, 0x97, 0xb6, 0x01, 0x6b, 0xa6, 0xf9, 0x0e, 0xdc, 0x62, 0x09,
	0x0f, 0x22, 0x92, 0x89, 0x7a, 0x07, 0xa7, 0xce, 0x9f, 0xf7, 0x58, 0x99, 0xef, 0xd4, 0x37, 0xe0,
	0x5c, 0xbc, 0xf1, 0xfd, 0x89, 0x40, 0x82, 0x7b, 0x47, 0xda, 0x9b, 0x36, 0x7c, 0x39, 0x19, 0x49,
	0x15, 0xd8, 0x66, 0x09, 0x57, 0x51, 0xff, 0x1c, 0xb4, 0x2e, 0x08, 0x8b, 0xc9, 0xff, 0xb5, 0xaa,
	0xa0, 0x7f, 0x6f, 0xb5, 0xff, 0x65, 0x17, 0xdc, 0xd3, 0xcb, 0x16, 0x24, 0x7a, 0xa5, 0x7e, 0x23,
	0x94, 0x8d, 0x90, 0x40, 0xe6, 0x09, 0xe8, 0x84, 0x19, 0xc5, 0xd3, 0x80, 0x11, 0x14, 0x71, 0xf5,
	0x46, 0x5b, 0xaf, 0x0f, 0xa8, 0x02, 0x94, 0x79, 0xf3, 0x0c, 0xb4, 0xd4, 0x39, 0xea, 0x41, 0x1f,
	0x3a, 0xff, 0xfc, 0xf1, 0x6a, 0xe3, 0xe5, 0x9c, 0x44, 0xab, 0x55, 0x7c, 0xf3, 0x05, 0x68, 0x71,
	0xb9, 0x58, 0xed, 0xf6, 0x60, 0x8b, 0x90, 0x32, 0x01, 0x56, 0x14, 0xc9, 0xcd, 0xe5, 0xa4, 0xca,
	0xe4, 0xed, 0x5c, 0xb5, 0x15, 0x58, 0x51, 0xcc, 0x4b, 0x70, 0x84, 0x69, 0x5e, 0xce, 0x04, 0x09,
	0x70, 0x42, 0xf0, 0x94, 0xcf, 0xf2, 0x6e, 0x4b, 0xc9, 0x3c, 0x68, 0x58, 0xa8, 0x5f, 0x41, 0x7d,
	0x8d, 0xd4, 0xa7, 0x07, 0x0f, 0xf1, 0xcd, 0xbc, 0x77, 0xb2, 0xf8, 0x6e, 0xed, 0x2c, 0x56, 0x96,
	0x71, 0xbd, 0xb2, 0x8c, 0xaf, 0x2b, 0xcb, 0xf8, 0xb6, 0xb2, 0x8c, 0x8f, 0x3f, 0xac, 0x9d, 0xb7,
	0x9d, 0x8d, 0x4e, 0x7e, 0x05, 0x00, 0x00, 0xff, 0xff, 0x87, 0xb0, 0xc7, 0x45, 0xdc, 0x04, 0x00,
	0x00,
}
