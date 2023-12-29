package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"strings"
)

type DNSHeader struct {
	ID      uint16
	Flags   DNSHeaderFlags
	QDCOUNT uint16
	ANCOUNT uint16
	NSCOUNT uint16
	ARCOUNT uint16
}

func (h *DNSHeader) UnmarshalBinary(data []byte) error {
	buff := bytes.NewBuffer(data)

	if err := binary.Read(buff, binary.BigEndian, &h.ID); err != nil {
		return err
	}
	if err := h.Flags.UnmarshalBinary(buff.Next(2)); err != nil {
		return err
	}
	if err := binary.Read(buff, binary.BigEndian, &h.QDCOUNT); err != nil {
		return err
	}
	if err := binary.Read(buff, binary.BigEndian, &h.ANCOUNT); err != nil {
		return err
	}
	if err := binary.Read(buff, binary.BigEndian, &h.NSCOUNT); err != nil {
		return err
	}
	if err := binary.Read(buff, binary.BigEndian, &h.ARCOUNT); err != nil {
		return err
	}

	return nil
}

func (header DNSHeader) MarshalBinary() ([]byte, error) {
	buff := make([]byte, 12)
	var flags uint16

	binary.BigEndian.PutUint16(buff[:2], header.ID)

	if header.Flags.QR {
		flags |= 1 << 15 // set 1st bit
	}
	flags |= header.Flags.OPCODE << 11
	if header.Flags.AA {
		flags |= 1 << 10
	}
	if header.Flags.TC {
		flags |= 1 << 9
	}
	if header.Flags.RD {
		flags |= 1 << 8
	}
	if header.Flags.RA {
		flags |= 1 << 7
	}
	flags |= header.Flags.Z << 4
	flags |= header.Flags.RCODE

	binary.BigEndian.PutUint16(buff[2:4], flags)
	binary.BigEndian.PutUint16(buff[4:6], header.QDCOUNT)
	binary.BigEndian.PutUint16(buff[6:8], header.ANCOUNT)
	binary.BigEndian.PutUint16(buff[8:10], header.NSCOUNT)
	binary.BigEndian.PutUint16(buff[10:], header.ARCOUNT)

	return buff, nil
}

type DNSHeaderFlags struct {
	QR     bool
	OPCODE uint16 // 4bit
	AA     bool
	TC     bool
	RD     bool
	RA     bool
	Z      uint16 // 3bit
	RCODE  uint16 // 4bit
}

func (f *DNSHeaderFlags) UnmarshalBinary(data []byte) error {
	flags := binary.BigEndian.Uint16(data)
	qrMask := uint16(0x8000)
	opcodeMask := uint16(0x7800)
	aaMask := uint16(0x0400)
	tcMask := uint16(0x0200)
	rdMask := uint16(0x0100)
	raMask := uint16(0x0050)
	zMask := uint16(0x0070)
	rcodeMask := uint16(0x000F)

	f.QR = (flags & qrMask) != 0
	f.OPCODE = uint16((flags & opcodeMask) >> 11)
	f.AA = (flags & aaMask) != 0
	f.TC = (flags & tcMask) != 0
	f.RD = (flags & rdMask) != 0
	f.RA = (flags & raMask) != 0
	f.Z = uint16((flags & zMask) << 4)
	f.RCODE = uint16(flags & rcodeMask)

	return nil
}

type DNSQuestion struct {
	Name  string
	Type  uint16
	Class uint16
}

func (q DNSQuestion) MarshalBinary() ([]byte, error) {
	var buff bytes.Buffer

	serializedDomain := MarshalDomain(q.Name)
	if _, err := buff.Write(serializedDomain); err != nil {
		return nil, err
	}
	if err := binary.Write(&buff, binary.BigEndian, q.Type); err != nil {
		return nil, err
	}
	if err := binary.Write(&buff, binary.BigEndian, q.Class); err != nil {
		return nil, err
	}

	return buff.Bytes()[:buff.Len()], nil
}

func (q *DNSQuestion) UnMarshalBinary(buf []byte) error {
	var s strings.Builder
	r := bytes.NewReader(buf)

	for {
		labelLen, err := r.ReadByte()
		if err != nil {
			return err
		}
		if labelLen == 0x00 {
			break
		}
		labelBytes := make([]byte, labelLen)
		if _, err := r.Read(labelBytes); err != nil {
			return err
		}
		if s.Len() != 0 {
			s.WriteRune('.')
		}
		s.Write(labelBytes)
	}

	q.Name = s.String()
	if err := binary.Read(r, binary.BigEndian, &q.Type); err != nil {
		return err
	}
	if err := binary.Read(r, binary.BigEndian, &q.Class); err != nil {
		return err
	}

	return nil
}

type DNSAnswer struct {
	Name  string
	Type  uint16
	Class uint16
	TTL   uint32
	Data  []byte
}

func (a DNSAnswer) MarshalBinary() ([]byte, error) {
	var buff bytes.Buffer

	if _, err := buff.Write(MarshalDomain(a.Name)); err != nil {
		return nil, err
	}
	binary.Write(&buff, binary.BigEndian, a.Type)
	binary.Write(&buff, binary.BigEndian, a.Class)
	binary.Write(&buff, binary.BigEndian, a.TTL)
	binary.Write(&buff, binary.BigEndian, uint16(len(a.Data)))
	if _, err := buff.Write(a.Data); err != nil {
		return nil, err
	}

	return buff.Bytes()[:buff.Len()], nil
}

type DNSMessage struct {
	Header    DNSHeader
	Questions []DNSQuestion
	Answers   []DNSAnswer
}

// CreateResponse create a DNS response base on a DNS request
// It automatically set DNSHeader.ID, DNSHeaderFlags.QR, DNSHeaderFlags.OPCODE, DNSHeaderFlags.RD
func CreateResponse(req *DNSMessage) *DNSMessage {
	msg := &DNSMessage{
		Header: DNSHeader{
			ID: req.Header.ID,
			Flags: DNSHeaderFlags{
				QR:     true,
				OPCODE: req.Header.Flags.OPCODE,
				RD:     req.Header.Flags.RD,
			},
			QDCOUNT: uint16(len(req.Questions)),
		},
		Questions: req.Questions,
	}
	if req.Header.Flags.OPCODE != 0 {
		msg.Header.Flags.RCODE = 4
	}
	return msg
}

func (msg *DNSMessage) AddQuestions(questions ...DNSQuestion) {
	msg.Questions = append(msg.Questions, questions...)
	msg.Header.QDCOUNT += uint16(len(questions))
}

func (msg *DNSMessage) AddAnswers(answers ...DNSAnswer) {
	msg.Answers = append(msg.Answers, answers...)
	msg.Header.ANCOUNT += uint16(len(answers))
}

func (msg *DNSMessage) UnmarshalBinary(data []byte) error {
	byteCount := 12
	r := bytes.NewReader(data)
	if err := readHeader(r, &msg.Header); err != nil {
		return err
	}
	log.Printf("read header: %+v\n", msg.Header)
	questions, n, err := readQuestions(r, byteCount, msg.Header.QDCOUNT)
	if err != nil {
		return err
	}
	byteCount += n
	msg.Questions = questions
	answers, _, err := readAnswers(r, byteCount, msg.Header.ANCOUNT)
	if err != nil {
		return err
	}
	msg.Answers = answers

	return nil
}

func (msg DNSMessage) MarshalBinary() ([]byte, error) {
	var buff bytes.Buffer

	header, err := msg.Header.MarshalBinary()
	if err != nil {
		return nil, err
	}
	if _, err := buff.Write(header); err != nil {
		return nil, err
	}
	for _, q := range msg.Questions {
		b, err := q.MarshalBinary()
		if err != nil {
			return nil, err
		}
		if _, err := buff.Write(b); err != nil {
			return nil, err
		}
	}
	for _, a := range msg.Answers {
		b, err := a.MarshalBinary()
		if err != nil {
			return nil, err
		}
		if _, err := buff.Write(b); err != nil {
			return nil, err
		}
	}

	return buff.Bytes()[:buff.Len()], nil
}

func MarshalDomain(domain string) []byte {
	var buff bytes.Buffer

	labels := strings.Split(domain, ".")
	for _, label := range labels {
		b := []byte(label)
		buff.WriteByte(uint8(len(b)))
		buff.Write(b)
	}
	buff.WriteByte(0x0)

	return buff.Bytes()[:buff.Len()]
}

func UnMarshalDomain(buf []byte) (string, error) {
	var s strings.Builder
	var i uint
	maxLen := uint(len(buf) - 1) // Exclude the 0x00 at the end

	for i < maxLen {
		strLen := uint(buf[i])
		i++
		s.Write(buf[i : i+strLen])
		i += strLen
		if i < maxLen {
			s.WriteRune('.')
		}
	}

	return s.String(), nil
}

func readHeader(r io.Reader, header *DNSHeader) error {
	if err := binary.Read(r, binary.BigEndian, &header.ID); err != nil {
		return err
	}
	flags := make([]byte, 2)
	if _, err := r.Read(flags); err != nil {
		return err
	}
	if err := header.Flags.UnmarshalBinary(flags); err != nil {
		return err
	}
	if err := binary.Read(r, binary.BigEndian, &header.QDCOUNT); err != nil {
		return err
	}
	if err := binary.Read(r, binary.BigEndian, &header.ANCOUNT); err != nil {
		return err
	}
	if err := binary.Read(r, binary.BigEndian, &header.NSCOUNT); err != nil {
		return err
	}
	if err := binary.Read(r, binary.BigEndian, &header.ARCOUNT); err != nil {
		return err
	}

	return nil
}

// readDomain read labels and build the domain. It follows encountered pointers.
// It stops after encountering either 0x00 or a pointer.
func readDomain(r *bytes.Reader, pos int) (string, int, error) {
	byteCount := 0
	labels := []string{}
	log.Println("read domain")

	for {
		b, err := r.ReadByte()
		if err != nil {
			return "", byteCount, fmt.Errorf("failed to read domain head byte: %v", err)
		}
		byteCount++
		if b == 0 { // End of domain
			break
		}
		if b >= 192 { // this is a pointer
			// Discard the readbyte and re-read as a uint16 with the following byte
			if err := r.UnreadByte(); err != nil {
				return "", byteCount, fmt.Errorf("failed to unread ptr first byte: %v", err)
			}
			byteCount--
			var offset uint16
			if err := binary.Read(r, binary.BigEndian, &offset); err != nil {
				return "", byteCount, fmt.Errorf("failed to read pointer bytes: %v", err)
			}
			byteCount += 2
			offset &= 0x3FFF // Discard the first 2 bit indicating this is a pointer
			original := pos + byteCount
			_, err := r.Seek(int64(offset), io.SeekStart)
			if err != nil {
				return "", byteCount, fmt.Errorf("failed to seek to pointer offset: %v", err)
			}
			str, _, err := readDomain(r, int(offset))
			if err != nil {
				return "", byteCount, fmt.Errorf("failed to recursively call readDomain: %v", err)
			}
			_, err = r.Seek(int64(original), io.SeekStart)
			if err != nil {
				return "", byteCount, fmt.Errorf("failed to seek back to original position: %v", err)
			}
			labels = append(labels, str)
			break
		} else {
			size := b
			buf := make([]byte, size)
			n, err := r.Read(buf)
			if err != nil {
				return "", byteCount, fmt.Errorf("failed to read label: %v", err)
			}
			byteCount += n
			labels = append(labels, string(buf))
		}

	}

	s := strings.Join(labels, ".")

	return s, byteCount, nil
}

func readQuestion(r *bytes.Reader, pos int) (DNSQuestion, int, error) {
	byteCount := 0
	domain, n, err := readDomain(r, pos)
	if err != nil {
		return DNSQuestion{}, byteCount, err
	}
	byteCount = +n

	question := DNSQuestion{
		Name: domain,
	}
	if err := binary.Read(r, binary.BigEndian, &question.Type); err != nil {
		return DNSQuestion{}, byteCount, err
	}
	byteCount += 2
	if err := binary.Read(r, binary.BigEndian, &question.Class); err != nil {
		return DNSQuestion{}, byteCount, err
	}
	byteCount += 2

	return question, byteCount, nil
}

func readQuestions(r *bytes.Reader, pos int, qcount uint16) ([]DNSQuestion, int, error) {
	byteCount := 0
	questions := make([]DNSQuestion, 0, qcount)
	for i := 0; i < int(qcount); i++ {
		q, n, err := readQuestion(r, pos+byteCount)
		if err != nil {
			return nil, pos, err
		}
		byteCount += n
		questions = append(questions, q)
	}

	return questions, byteCount, nil
}

func readAnswer(r *bytes.Reader, pos int) (DNSAnswer, int, error) {
	var answer DNSAnswer
	byteCount := 0
	domain, n, err := readDomain(r, pos)
	if err != nil {
		return DNSAnswer{}, n, err
	}
	answer.Name = domain
	byteCount += n
	if err := binary.Read(r, binary.BigEndian, &answer.Type); err != nil {
		return DNSAnswer{}, byteCount, err
	}
	byteCount += 2
	if err := binary.Read(r, binary.BigEndian, &answer.Class); err != nil {
		return DNSAnswer{}, byteCount, err
	}
	byteCount += 2
	if err := binary.Read(r, binary.BigEndian, &answer.TTL); err != nil {
		return DNSAnswer{}, byteCount, err
	}
	byteCount += 4

	var rdLen uint16
	if err := binary.Read(r, binary.BigEndian, &rdLen); err != nil {
		return DNSAnswer{}, byteCount, err
	}
	byteCount += 2
	buf := make([]byte, rdLen)
	n, err = r.Read(buf)
	if err != nil {
		return DNSAnswer{}, byteCount, err
	}
	byteCount += n
	answer.Data = buf[:n]

	return answer, byteCount, nil
}

func readAnswers(r *bytes.Reader, pos int, ancount uint16) ([]DNSAnswer, int, error) {
	byteCount := 0
	answers := make([]DNSAnswer, ancount)
	for i := 0; i < int(ancount); i++ {
		answer, n, err := readAnswer(r, pos+byteCount)
		if err != nil {
			return nil, byteCount, err
		}
		byteCount += n
		answers[i] = answer
	}
	return answers, byteCount, nil
}
