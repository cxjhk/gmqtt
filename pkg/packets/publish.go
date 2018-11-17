package packets

import (
	"encoding/binary"
	"io"

	"fmt"
)

type Publish struct {
	FixHeader *FixHeader
	Dup bool //是否重发 [MQTT-3.3.1.-1]
	Qos uint8 //qos等级
	Retain bool //是否保留消息
	TopicName []byte //主题名
	PacketId //报文标识符
	Payload []byte
}

func (c *Publish) String() string {
	return fmt.Sprintf("Publish, Pid: %v, Dup: %v, Qos: %v, Retain: %v, TopicName: %s, Payload: %s",
		c.PacketId, c.Dup, c.Qos, c.Retain, c.TopicName, c.Payload)
}


//copy一份分发
func (p *Publish) CopyPublish() *Publish{
	pub := &Publish{
		Dup:p.Dup,
		Qos:p.Qos,
		Retain:p.Retain,
		PacketId:p.PacketId,
	}
	pub.Payload = make([]byte,len(p.Payload))
	pub.TopicName = make([]byte,len(p.TopicName))
	copy(pub.TopicName, p.TopicName)
	copy(pub.Payload, p.Payload)
	return pub
}



func NewPublishPacket(fh *FixHeader,r io.Reader) (*Publish,error) {
	p := &Publish{FixHeader:fh}
	p.Dup = (1 & (fh.Flags >> 3)) > 0
	p.Qos = (fh.Flags >> 1) & 3
	if p.Qos == 0 && p.Dup { //[MQTT-3.3.1-2]、 [MQTT-4.3.1-1]
		return  nil ,ErrInvalFlags
	}
	if p.Qos > QOS_2 {
		return  nil ,ErrInvalQos
	}
	if fh.Flags & 1 == 1 { //保留标志
		p.Retain = true
	}
	err := p.Unpack(r)
	if err != nil {
		return nil,err
	}
	return p,nil
}




//pack
func (p *Publish) Pack(w io.Writer) error {
	p.FixHeader = &FixHeader{PacketType:PUBLISH}
	var dup,retain byte
	dup = 0
	retain = 0
	if p.Dup {
		dup = 8
	}
	if p.Retain {
		retain = 1
	}
	p.FixHeader.Flags = dup | retain | (p.Qos << 1)
	topicLength := uint16(len(p.TopicName))
	lenByte := make([]byte,2)
	binary.BigEndian.PutUint16(lenByte,topicLength)
	remainLength := int(topicLength) + 2 + len(p.Payload) //4 : 2个字节packetid 2个字节的topiclen
	if p.Qos == QOS_1 || p.Qos == QOS_2{
		remainLength += 2
	}
	p.FixHeader.RemainLength = remainLength
	p.FixHeader.Pack(w)
	w.Write(lenByte)
	w.Write(p.TopicName)
	if p.Qos == QOS_1 || p.Qos == QOS_2 {
		pid := make([]byte,2)
		binary.BigEndian.PutUint16(pid,p.PacketId)
		w.Write(pid)
	}
	_, err := w.Write(p.Payload)
	return err

}


func (p *Publish) Unpack(r io.Reader) error {
	var size int
	var err error
	restBuffer := make([]byte,p.FixHeader.RemainLength)
	_,err = io.ReadFull(r,restBuffer)
	if err != nil {
		return err
	}
	p.TopicName, size, err = DecodeUTF8String(restBuffer)
	if err != nil {
		return err
	}
	restBuffer = restBuffer[size:]
	if !ValidTopicName(p.TopicName) {
		return ErrInvalTopicName
	}
	if p.Qos > QOS_0 {
		p.PacketId = binary.BigEndian.Uint16(restBuffer[0:2])
		restBuffer = restBuffer[2:]
	}
	p.Payload = restBuffer
	return nil
}

//qos = 1
func (p *Publish) NewPuback() *Puback{
	pub := &Puback{FixHeader:&FixHeader{PacketType:PUBACK,Flags:RESERVED,RemainLength:2}}
	pub.PacketId = p.PacketId
	return pub
}

//qos2
func (p *Publish) NewPubrec() *Pubrec{
	pub := &Pubrec{FixHeader:&FixHeader{PacketType:PUBREC,Flags:RESERVED,RemainLength:2}}
	pub.PacketId = p.PacketId
	return pub
}
