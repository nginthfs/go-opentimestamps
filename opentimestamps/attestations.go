package opentimestamps

import (
	"bytes"
	"fmt"
)

const (
	attestationTagSize             = 8
	attestationMaxPayloadSize      = 8192
	pendingAttestationMaxUriLength = 1000
)

var (
	bitcoinAttestationTag = mustDecodeHex("0588960d73d71901")
	pendingAttestationTag = mustDecodeHex("83dfe30d2ef90c8e")
)

type attestation interface {
	match(tag []byte) bool
	decode(*deserializationContext) (attestation, error)
}

type baseAttestation struct {
	tag []byte
}

func (b *baseAttestation) match(tag []byte) bool {
	return bytes.Equal(b.tag, tag)
}

type pendingAttestation struct {
	baseAttestation
	uri string
}

func newPendingAttestation() *pendingAttestation {
	return &pendingAttestation{
		baseAttestation: baseAttestation{tag: pendingAttestationTag},
	}
}

func (p *pendingAttestation) match(tag []byte) bool {
	return p.baseAttestation.match(tag)
}

func (p *pendingAttestation) decode(
	ctx *deserializationContext,
) (attestation, error) {
	uri, err := ctx.readVarBytes(0, pendingAttestationMaxUriLength)
	if err != nil {
		return nil, err
	}
	// TODO utf8 checks
	ret := *p
	ret.uri = string(uri)
	return &ret, nil
}

func (p *pendingAttestation) String() string {
	return fmt.Sprintf("VERIFY PendingAttestation(url=%s)", p.uri)
}

type bitcoinAttestation struct {
	baseAttestation
	height uint64
}

func newBitcoinAttestation() *bitcoinAttestation {
	return &bitcoinAttestation{
		baseAttestation: baseAttestation{bitcoinAttestationTag},
	}
}

func (b *bitcoinAttestation) String() string {
	return fmt.Sprintf("VERIFY BitcoinAttestation(height=%d)", b.height)
}

func (b *bitcoinAttestation) match(tag []byte) bool {
	return b.baseAttestation.match(tag)
}

func (b *bitcoinAttestation) decode(
	ctx *deserializationContext,
) (attestation, error) {
	height, err := ctx.readVarUint()
	if err != nil {
		return nil, err
	}
	ret := *b
	ret.height = height
	return &ret, nil
}

// This is a catch-all for when we don't know how to parse it
type unknownAttestation struct {
	tag   []byte
	bytes []byte
}

func (u unknownAttestation) match(tag []byte) bool {
	panic("not implemented")
}

func (u unknownAttestation) decode(*deserializationContext) (attestation, error) {
	panic("not implemented")
}

func (u unknownAttestation) String() string {
	return fmt.Sprintf("UnknownAttestation(bytes=%q)", u.bytes)
}

var attestations []attestation = []attestation{
	newPendingAttestation(),
	newBitcoinAttestation(),
}

func ParseAttestation(ctx *deserializationContext) (attestation, error) {
	tag, err := ctx.readBytes(attestationTagSize)
	if err != nil {
		return nil, err
	}

	attBytes, err := ctx.readVarBytes(
		0, attestationMaxPayloadSize,
	)
	if err != nil {
		return nil, err
	}
	attCtx := newDeserializationContext(
		bytes.NewBuffer(attBytes),
	)

	for _, a := range attestations {
		if a.match(tag) {
			att, err := a.decode(attCtx)
			if err != nil {
				return nil, err
			}
			if !attCtx.assertEOF() {
				return nil, fmt.Errorf("expected EOF in attCtx")
			}
			return att, nil
		}
	}
	return unknownAttestation{tag, attBytes}, nil
}
