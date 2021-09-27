package encoding

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"hash/fnv"
	"io"

	"golang.org/x/crypto/chacha20poly1305"

	"github.com/Shadowsocks-NET/v2ray-go/v4/common"
	"github.com/Shadowsocks-NET/v2ray-go/v4/common/bitmask"
	"github.com/Shadowsocks-NET/v2ray-go/v4/common/buf"
	"github.com/Shadowsocks-NET/v2ray-go/v4/common/crypto"
	"github.com/Shadowsocks-NET/v2ray-go/v4/common/dice"
	"github.com/Shadowsocks-NET/v2ray-go/v4/common/drain"
	"github.com/Shadowsocks-NET/v2ray-go/v4/common/protocol"
	"github.com/Shadowsocks-NET/v2ray-go/v4/proxy/vmess"
	vmessaead "github.com/Shadowsocks-NET/v2ray-go/v4/proxy/vmess/aead"
)

// ClientSession stores connection session info for VMess client.
type ClientSession struct {
	idHash          protocol.IDHash
	requestBodyKey  [16]byte
	requestBodyIV   [16]byte
	responseBodyKey [16]byte
	responseBodyIV  [16]byte
	responseReader  io.Reader
	responseHeader  byte

	readDrainer drain.Drainer
}

// NewClientSession creates a new ClientSession.
func NewClientSession(ctx context.Context, idHash protocol.IDHash, behaviorSeed int64) *ClientSession {
	session := &ClientSession{
		idHash: idHash,
	}

	randomBytes := make([]byte, 33) // 16 + 16 + 1
	common.Must2(rand.Read(randomBytes))
	copy(session.requestBodyKey[:], randomBytes[:16])
	copy(session.requestBodyIV[:], randomBytes[16:32])
	session.responseHeader = randomBytes[32]

	BodyKey := sha256.Sum256(session.requestBodyKey[:])
	copy(session.responseBodyKey[:], BodyKey[:16])
	BodyIV := sha256.Sum256(session.requestBodyIV[:])
	copy(session.responseBodyIV[:], BodyIV[:16])

	{
		var err error
		session.readDrainer, err = drain.NewBehaviorSeedLimitedDrainer(behaviorSeed, 18, 3266, 64)
		if err != nil {
			newError("unable to initialize drainer").Base(err).WriteToLog()
			session.readDrainer = drain.NewNopDrainer()
		}
	}

	return session
}

func (c *ClientSession) EncodeRequestHeader(header *protocol.RequestHeader, writer io.Writer) error {
	account := header.User.Account.(*vmess.MemoryAccount)

	buffer := buf.New()
	defer buffer.Release()

	common.Must(buffer.WriteByte(Version))
	common.Must2(buffer.Write(c.requestBodyIV[:]))
	common.Must2(buffer.Write(c.requestBodyKey[:]))
	common.Must(buffer.WriteByte(c.responseHeader))
	common.Must(buffer.WriteByte(byte(header.Option)))

	paddingLen := dice.Roll(16)
	security := byte(paddingLen<<4) | byte(header.Security)
	common.Must2(buffer.Write([]byte{security, byte(0), byte(header.Command)}))

	if header.Command != protocol.RequestCommandMux {
		if err := addrParser.WriteAddressPort(buffer, header.Address, header.Port); err != nil {
			return newError("failed to writer address and port").Base(err)
		}
	}

	if paddingLen > 0 {
		common.Must2(buffer.ReadFullFrom(rand.Reader, int32(paddingLen)))
	}

	{
		fnv1a := fnv.New32a()
		common.Must2(fnv1a.Write(buffer.Bytes()))
		hashBytes := buffer.Extend(int32(fnv1a.Size()))
		fnv1a.Sum(hashBytes[:0])
	}

	var fixedLengthCmdKey [16]byte
	copy(fixedLengthCmdKey[:], account.ID.CmdKey())
	vmessout := vmessaead.SealVMessAEADHeader(fixedLengthCmdKey, buffer.Bytes())
	common.Must2(io.Copy(writer, bytes.NewReader(vmessout)))

	return nil
}

func (c *ClientSession) EncodeRequestBody(request *protocol.RequestHeader, writer io.Writer) buf.Writer {
	var sizeParser crypto.ChunkSizeEncoder = crypto.PlainChunkSizeParser{}
	if request.Option.Has(protocol.RequestOptionChunkMasking) {
		sizeParser = NewShakeSizeParser(c.requestBodyIV[:])
	}
	var padding crypto.PaddingLengthGenerator
	if request.Option.Has(protocol.RequestOptionGlobalPadding) {
		padding = sizeParser.(crypto.PaddingLengthGenerator)
	}

	switch request.Security {
	case protocol.SecurityType_NONE:
		if request.Option.Has(protocol.RequestOptionChunkStream) {
			if request.Command.TransferType() == protocol.TransferTypeStream {
				return crypto.NewChunkStreamWriter(sizeParser, writer)
			}
			auth := &crypto.AEADAuthenticator{
				AEAD:                    new(NoOpAuthenticator),
				NonceGenerator:          crypto.GenerateEmptyBytes(),
				AdditionalDataGenerator: crypto.GenerateEmptyBytes(),
			}
			return crypto.NewAuthenticationWriter(auth, sizeParser, writer, protocol.TransferTypePacket, padding)
		}

		return buf.NewWriter(writer)
	case protocol.SecurityType_LEGACY:
		aesStream := crypto.NewAesEncryptionStream(c.requestBodyKey[:], c.requestBodyIV[:])
		cryptionWriter := crypto.NewCryptionWriter(aesStream, writer)
		if request.Option.Has(protocol.RequestOptionChunkStream) {
			auth := &crypto.AEADAuthenticator{
				AEAD:                    new(FnvAuthenticator),
				NonceGenerator:          crypto.GenerateEmptyBytes(),
				AdditionalDataGenerator: crypto.GenerateEmptyBytes(),
			}
			return crypto.NewAuthenticationWriter(auth, sizeParser, cryptionWriter, request.Command.TransferType(), padding)
		}

		return &buf.SequentialWriter{Writer: cryptionWriter}
	case protocol.SecurityType_AES128_GCM:
		aead := crypto.NewAesGcm(c.requestBodyKey[:])
		auth := &crypto.AEADAuthenticator{
			AEAD:                    aead,
			NonceGenerator:          GenerateChunkNonce(c.requestBodyIV[:], uint32(aead.NonceSize())),
			AdditionalDataGenerator: crypto.GenerateEmptyBytes(),
		}
		if request.Option.Has(protocol.RequestOptionAuthenticatedLength) {
			AuthenticatedLengthKey := vmessaead.KDF16(c.requestBodyKey[:], "auth_len")
			AuthenticatedLengthKeyAEAD := crypto.NewAesGcm(AuthenticatedLengthKey)

			lengthAuth := &crypto.AEADAuthenticator{
				AEAD:                    AuthenticatedLengthKeyAEAD,
				NonceGenerator:          GenerateChunkNonce(c.requestBodyIV[:], uint32(aead.NonceSize())),
				AdditionalDataGenerator: crypto.GenerateEmptyBytes(),
			}
			sizeParser = NewAEADSizeParser(lengthAuth)
		}
		return crypto.NewAuthenticationWriter(auth, sizeParser, writer, request.Command.TransferType(), padding)
	case protocol.SecurityType_CHACHA20_POLY1305:
		aead, err := chacha20poly1305.New(GenerateChacha20Poly1305Key(c.requestBodyKey[:]))
		common.Must(err)

		auth := &crypto.AEADAuthenticator{
			AEAD:                    aead,
			NonceGenerator:          GenerateChunkNonce(c.requestBodyIV[:], uint32(aead.NonceSize())),
			AdditionalDataGenerator: crypto.GenerateEmptyBytes(),
		}
		if request.Option.Has(protocol.RequestOptionAuthenticatedLength) {
			AuthenticatedLengthKey := vmessaead.KDF16(c.requestBodyKey[:], "auth_len")
			AuthenticatedLengthKeyAEAD, err := chacha20poly1305.New(GenerateChacha20Poly1305Key(AuthenticatedLengthKey))
			common.Must(err)

			lengthAuth := &crypto.AEADAuthenticator{
				AEAD:                    AuthenticatedLengthKeyAEAD,
				NonceGenerator:          GenerateChunkNonce(c.requestBodyIV[:], uint32(aead.NonceSize())),
				AdditionalDataGenerator: crypto.GenerateEmptyBytes(),
			}
			sizeParser = NewAEADSizeParser(lengthAuth)
		}
		return crypto.NewAuthenticationWriter(auth, sizeParser, writer, request.Command.TransferType(), padding)
	default:
		panic("Unknown security type.")
	}
}

func (c *ClientSession) DecodeResponseHeader(reader io.Reader) (*protocol.ResponseHeader, error) {
	aeadResponseHeaderLengthEncryptionKey := vmessaead.KDF16(c.responseBodyKey[:], vmessaead.KDFSaltConstAEADRespHeaderLenKey)
	aeadResponseHeaderLengthEncryptionIV := vmessaead.KDF(c.responseBodyIV[:], vmessaead.KDFSaltConstAEADRespHeaderLenIV)[:12]

	aeadResponseHeaderLengthEncryptionKeyAESBlock := common.Must2(aes.NewCipher(aeadResponseHeaderLengthEncryptionKey)).(cipher.Block)
	aeadResponseHeaderLengthEncryptionAEAD := common.Must2(cipher.NewGCM(aeadResponseHeaderLengthEncryptionKeyAESBlock)).(cipher.AEAD)

	var aeadEncryptedResponseHeaderLength [18]byte
	var decryptedResponseHeaderLength int
	var decryptedResponseHeaderLengthBinaryDeserializeBuffer uint16

	if n, err := io.ReadFull(reader, aeadEncryptedResponseHeaderLength[:]); err != nil {
		c.readDrainer.AcknowledgeReceive(n)
		return nil, drain.WithError(c.readDrainer, reader, newError("Unable to Read Header Len").Base(err))
	} else { // nolint: golint
		c.readDrainer.AcknowledgeReceive(n)
	}
	if decryptedResponseHeaderLengthBinaryBuffer, err := aeadResponseHeaderLengthEncryptionAEAD.Open(nil, aeadResponseHeaderLengthEncryptionIV, aeadEncryptedResponseHeaderLength[:], nil); err != nil {
		return nil, drain.WithError(c.readDrainer, reader, newError("Failed To Decrypt Length").Base(err))
	} else { // nolint: golint
		common.Must(binary.Read(bytes.NewReader(decryptedResponseHeaderLengthBinaryBuffer), binary.BigEndian, &decryptedResponseHeaderLengthBinaryDeserializeBuffer))
		decryptedResponseHeaderLength = int(decryptedResponseHeaderLengthBinaryDeserializeBuffer)
	}

	aeadResponseHeaderPayloadEncryptionKey := vmessaead.KDF16(c.responseBodyKey[:], vmessaead.KDFSaltConstAEADRespHeaderPayloadKey)
	aeadResponseHeaderPayloadEncryptionIV := vmessaead.KDF(c.responseBodyIV[:], vmessaead.KDFSaltConstAEADRespHeaderPayloadIV)[:12]

	aeadResponseHeaderPayloadEncryptionKeyAESBlock := common.Must2(aes.NewCipher(aeadResponseHeaderPayloadEncryptionKey)).(cipher.Block)
	aeadResponseHeaderPayloadEncryptionAEAD := common.Must2(cipher.NewGCM(aeadResponseHeaderPayloadEncryptionKeyAESBlock)).(cipher.AEAD)

	encryptedResponseHeaderBuffer := make([]byte, decryptedResponseHeaderLength+16)

	if n, err := io.ReadFull(reader, encryptedResponseHeaderBuffer); err != nil {
		c.readDrainer.AcknowledgeReceive(n)
		return nil, drain.WithError(c.readDrainer, reader, newError("Unable to Read Header Data").Base(err))
	} else { // nolint: golint
		c.readDrainer.AcknowledgeReceive(n)
	}

	if decryptedResponseHeaderBuffer, err := aeadResponseHeaderPayloadEncryptionAEAD.Open(nil, aeadResponseHeaderPayloadEncryptionIV, encryptedResponseHeaderBuffer, nil); err != nil {
		return nil, drain.WithError(c.readDrainer, reader, newError("Failed To Decrypt Payload").Base(err))
	} else { // nolint: golint
		c.responseReader = bytes.NewReader(decryptedResponseHeaderBuffer)
	}

	buffer := buf.StackNew()
	defer buffer.Release()

	if _, err := buffer.ReadFullFrom(c.responseReader, 4); err != nil {
		return nil, newError("failed to read response header").Base(err).AtWarning()
	}

	if buffer.Byte(0) != c.responseHeader {
		return nil, newError("unexpected response header. Expecting ", int(c.responseHeader), " but actually ", int(buffer.Byte(0)))
	}

	header := &protocol.ResponseHeader{
		Option: bitmask.Byte(buffer.Byte(1)),
	}

	if buffer.Byte(2) != 0 {
		cmdID := buffer.Byte(2)
		dataLen := int32(buffer.Byte(3))

		buffer.Clear()
		if _, err := buffer.ReadFullFrom(c.responseReader, dataLen); err != nil {
			return nil, newError("failed to read response command").Base(err)
		}
		command, err := UnmarshalCommand(cmdID, buffer.Bytes())
		if err == nil {
			header.Command = command
		}
	}
	aesStream := crypto.NewAesDecryptionStream(c.responseBodyKey[:], c.responseBodyIV[:])
	c.responseReader = crypto.NewCryptionReader(aesStream, reader)

	return header, nil
}

func (c *ClientSession) DecodeResponseBody(request *protocol.RequestHeader, reader io.Reader) buf.Reader {
	var sizeParser crypto.ChunkSizeDecoder = crypto.PlainChunkSizeParser{}
	if request.Option.Has(protocol.RequestOptionChunkMasking) {
		sizeParser = NewShakeSizeParser(c.responseBodyIV[:])
	}
	var padding crypto.PaddingLengthGenerator
	if request.Option.Has(protocol.RequestOptionGlobalPadding) {
		padding = sizeParser.(crypto.PaddingLengthGenerator)
	}

	switch request.Security {
	case protocol.SecurityType_NONE:
		if request.Option.Has(protocol.RequestOptionChunkStream) {
			if request.Command.TransferType() == protocol.TransferTypeStream {
				return crypto.NewChunkStreamReader(sizeParser, reader)
			}

			auth := &crypto.AEADAuthenticator{
				AEAD:                    new(NoOpAuthenticator),
				NonceGenerator:          crypto.GenerateEmptyBytes(),
				AdditionalDataGenerator: crypto.GenerateEmptyBytes(),
			}

			return crypto.NewAuthenticationReader(auth, sizeParser, reader, protocol.TransferTypePacket, padding)
		}

		return buf.NewReader(reader)
	case protocol.SecurityType_LEGACY:
		if request.Option.Has(protocol.RequestOptionChunkStream) {
			auth := &crypto.AEADAuthenticator{
				AEAD:                    new(FnvAuthenticator),
				NonceGenerator:          crypto.GenerateEmptyBytes(),
				AdditionalDataGenerator: crypto.GenerateEmptyBytes(),
			}
			return crypto.NewAuthenticationReader(auth, sizeParser, c.responseReader, request.Command.TransferType(), padding)
		}

		return buf.NewReader(c.responseReader)
	case protocol.SecurityType_AES128_GCM:
		aead := crypto.NewAesGcm(c.responseBodyKey[:])

		auth := &crypto.AEADAuthenticator{
			AEAD:                    aead,
			NonceGenerator:          GenerateChunkNonce(c.responseBodyIV[:], uint32(aead.NonceSize())),
			AdditionalDataGenerator: crypto.GenerateEmptyBytes(),
		}
		if request.Option.Has(protocol.RequestOptionAuthenticatedLength) {
			AuthenticatedLengthKey := vmessaead.KDF16(c.requestBodyKey[:], "auth_len")
			AuthenticatedLengthKeyAEAD := crypto.NewAesGcm(AuthenticatedLengthKey)

			lengthAuth := &crypto.AEADAuthenticator{
				AEAD:                    AuthenticatedLengthKeyAEAD,
				NonceGenerator:          GenerateChunkNonce(c.requestBodyIV[:], uint32(aead.NonceSize())),
				AdditionalDataGenerator: crypto.GenerateEmptyBytes(),
			}
			sizeParser = NewAEADSizeParser(lengthAuth)
		}
		return crypto.NewAuthenticationReader(auth, sizeParser, reader, request.Command.TransferType(), padding)
	case protocol.SecurityType_CHACHA20_POLY1305:
		aead, _ := chacha20poly1305.New(GenerateChacha20Poly1305Key(c.responseBodyKey[:]))

		auth := &crypto.AEADAuthenticator{
			AEAD:                    aead,
			NonceGenerator:          GenerateChunkNonce(c.responseBodyIV[:], uint32(aead.NonceSize())),
			AdditionalDataGenerator: crypto.GenerateEmptyBytes(),
		}
		if request.Option.Has(protocol.RequestOptionAuthenticatedLength) {
			AuthenticatedLengthKey := vmessaead.KDF16(c.requestBodyKey[:], "auth_len")
			AuthenticatedLengthKeyAEAD, err := chacha20poly1305.New(GenerateChacha20Poly1305Key(AuthenticatedLengthKey))
			common.Must(err)

			lengthAuth := &crypto.AEADAuthenticator{
				AEAD:                    AuthenticatedLengthKeyAEAD,
				NonceGenerator:          GenerateChunkNonce(c.requestBodyIV[:], uint32(aead.NonceSize())),
				AdditionalDataGenerator: crypto.GenerateEmptyBytes(),
			}
			sizeParser = NewAEADSizeParser(lengthAuth)
		}
		return crypto.NewAuthenticationReader(auth, sizeParser, reader, request.Command.TransferType(), padding)
	default:
		panic("Unknown security type.")
	}
}

func GenerateChunkNonce(nonce []byte, size uint32) crypto.BytesGenerator {
	c := append([]byte(nil), nonce...)
	count := uint16(0)
	return func() []byte {
		binary.BigEndian.PutUint16(c, count)
		count++
		return c[:size]
	}
}
