package types

import (
	"os"

	"google.golang.org/protobuf/proto"
)

func (s *Stat) IsDir() bool {
	return os.FileMode(s.Mode).IsDir()
}

func (s *Stat) Marshal() ([]byte, error) {
	return proto.MarshalOptions{Deterministic: true}.Marshal(s)
}

func (s *Stat) Unmarshal(dAtA []byte) error {
	return proto.UnmarshalOptions{Merge: true}.Unmarshal(dAtA, s)
}

func (s *Stat) Clone() *Stat {
	return proto.Clone(s).(*Stat)
}
