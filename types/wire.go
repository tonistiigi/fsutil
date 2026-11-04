package types

const (
	PACKET_STAT     = Packet_PACKET_STAT
	PACKET_REQ      = Packet_PACKET_REQ
	PACKET_DATA     = Packet_PACKET_DATA
	PACKET_FIN      = Packet_PACKET_FIN
	PACKET_ERR      = Packet_PACKET_ERR
	PACKET_HASH_REQ = Packet_PACKET_HASH_REQ
	PACKET_HASH     = Packet_PACKET_HASH
)

func (p *Packet) Marshal() ([]byte, error) {
	return p.MarshalVTStrict()
}

func (p *Packet) Unmarshal(dAtA []byte) error {
	return p.UnmarshalVT(dAtA)
}

func (p *Packet) Size() int {
	return p.SizeVT()
}
