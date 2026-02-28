package wework

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"sort"
	"strings"
)

// Crypto 企业微信消息加解密接口
type Crypto interface {
	// VerifySignature 验证消息签名
	// 签名算法: SHA1(sort(token, timestamp, nonce, msgEncrypt))
	VerifySignature(signature, timestamp, nonce, msgEncrypt string) bool

	// Decrypt 解密消息
	// AES-CBC 解密，密钥由 EncodingAESKey base64 解码得到
	Decrypt(encrypted string) ([]byte, error)

	// Encrypt 加密消息（用于主动回复）
	Encrypt(plaintext []byte) (string, error)
}

// cryptoImpl Crypto 接口的实现
type cryptoImpl struct {
	token  string
	aesKey []byte
	corpID string
}

// NewCrypto 创建企业微信加解密服务实例
// encodingAESKey 为 43 字符的 Base64 编码密钥，追加 "=" 后解码得到 32 字节 AES 密钥
func NewCrypto(token, encodingAESKey, corpID string) (Crypto, error) {
	aesKey, err := base64.StdEncoding.DecodeString(encodingAESKey + "=")
	if err != nil {
		return nil, fmt.Errorf("decode encoding_aes_key: %w", err)
	}
	if len(aesKey) != 32 {
		return nil, fmt.Errorf("invalid aes key length: got %d, want 32", len(aesKey))
	}
	return &cryptoImpl{
		token:  token,
		aesKey: aesKey,
		corpID: corpID,
	}, nil
}

// VerifySignature 验证消息签名
// SHA1(sort(token, timestamp, nonce, msgEncrypt)) == signature
func (c *cryptoImpl) VerifySignature(signature, timestamp, nonce, msgEncrypt string) bool {
	params := []string{c.token, timestamp, nonce, msgEncrypt}
	sort.Strings(params)
	raw := strings.Join(params, "")
	hash := sha1.Sum([]byte(raw))
	computed := fmt.Sprintf("%x", hash)
	return computed == signature
}

// Decrypt 解密企业微信加密消息
// Base64 解码 → AES-CBC 解密（IV = aesKey[:16]）→ PKCS#7 去填充 → 解析明文 → 验证 corpID
func (c *cryptoImpl) Decrypt(encrypted string) ([]byte, error) {
	// 1. Base64 解码
	ciphertext, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return nil, fmt.Errorf("base64 decode: %w", err)
	}

	// 2. 验证密文长度是 AES 块大小的整数倍
	if len(ciphertext) == 0 || len(ciphertext)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("ciphertext length %d is not a multiple of block size %d", len(ciphertext), aes.BlockSize)
	}

	// 3. AES-CBC 解密，IV = aesKey[:16]
	block, err := aes.NewCipher(c.aesKey)
	if err != nil {
		return nil, fmt.Errorf("new aes cipher: %w", err)
	}
	mode := cipher.NewCBCDecrypter(block, c.aesKey[:aes.BlockSize])
	plaintext := make([]byte, len(ciphertext))
	mode.CryptBlocks(plaintext, ciphertext)

	// 4. 去除 PKCS#7 填充
	plaintext, err = pkcs7Unpad(plaintext)
	if err != nil {
		return nil, fmt.Errorf("pkcs7 unpad: %w", err)
	}

	// 5. 解析明文: random(16) + msgLen(4, big-endian) + msg + corpID
	if len(plaintext) < 20 {
		return nil, fmt.Errorf("plaintext too short: %d bytes", len(plaintext))
	}
	msgLen := binary.BigEndian.Uint32(plaintext[16:20])
	if uint32(len(plaintext)) < 20+msgLen {
		return nil, fmt.Errorf("invalid msg length: %d, plaintext length: %d", msgLen, len(plaintext))
	}
	msg := plaintext[20 : 20+msgLen]
	corpID := string(plaintext[20+msgLen:])

	// 6. 验证 CorpID
	if corpID != c.corpID {
		return nil, fmt.Errorf("corp_id mismatch: got %s, want %s", corpID, c.corpID)
	}

	return msg, nil
}

// Encrypt 加密消息
// 构造 random(16) + msgLen(4, big-endian) + msg + corpID → PKCS#7 填充 → AES-CBC 加密 → Base64 编码
func (c *cryptoImpl) Encrypt(plaintext []byte) (string, error) {
	// 1. 构造明文: random(16) + msgLen(4, big-endian) + msg + corpID
	randomBytes := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, randomBytes); err != nil {
		return "", fmt.Errorf("generate random bytes: %w", err)
	}

	msgLen := make([]byte, 4)
	binary.BigEndian.PutUint32(msgLen, uint32(len(plaintext)))

	buf := make([]byte, 0, 20+len(plaintext)+len(c.corpID))
	buf = append(buf, randomBytes...)
	buf = append(buf, msgLen...)
	buf = append(buf, plaintext...)
	buf = append(buf, []byte(c.corpID)...)

	// 2. PKCS#7 填充
	padded := pkcs7Pad(buf, aes.BlockSize)

	// 3. AES-CBC 加密，IV = aesKey[:16]
	block, err := aes.NewCipher(c.aesKey)
	if err != nil {
		return "", fmt.Errorf("new aes cipher: %w", err)
	}
	mode := cipher.NewCBCEncrypter(block, c.aesKey[:aes.BlockSize])
	ciphertext := make([]byte, len(padded))
	mode.CryptBlocks(ciphertext, padded)

	// 4. Base64 编码
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// pkcs7Pad 对数据进行 PKCS#7 填充
func pkcs7Pad(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	padBytes := make([]byte, padding)
	for i := range padBytes {
		padBytes[i] = byte(padding)
	}
	return append(data, padBytes...)
}

// pkcs7Unpad 去除 PKCS#7 填充
func pkcs7Unpad(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty data")
	}
	padding := int(data[len(data)-1])
	if padding == 0 || padding > aes.BlockSize {
		return nil, fmt.Errorf("invalid padding value: %d", padding)
	}
	if padding > len(data) {
		return nil, fmt.Errorf("padding %d exceeds data length %d", padding, len(data))
	}
	for i := len(data) - padding; i < len(data); i++ {
		if data[i] != byte(padding) {
			return nil, fmt.Errorf("invalid padding byte at position %d", i)
		}
	}
	return data[:len(data)-padding], nil
}
