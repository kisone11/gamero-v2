// Package crypto 提供 AES-GCM 对称加密工具，用于保护数据库中的敏感数据（如银行卡信息）。
// 密钥由配置文件通过 config.EncryptionKey（32 字节 hex 字符串）提供。
// 加密后的密文使用 base64url 编码，格式为 nonce(12B) || ciphertext，整体 base64url 编码存储。
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

var (
	// ErrKeyLength 密钥长度不正确
	ErrKeyLength = errors.New("crypto: AES key must be 32 bytes (AES-256)")
	// ErrCiphertextTooShort 密文过短（不含完整 nonce）
	ErrCiphertextTooShort = errors.New("crypto: ciphertext too short")
)

// Encryptor AES-GCM 加解密器
type Encryptor struct {
	key []byte // 32 字节 AES-256 密钥
}

// NewEncryptor 创建加解密器
// key 必须恰好为 32 字节（AES-256）
func NewEncryptor(key []byte) (*Encryptor, error) {
	if len(key) != 32 {
		return nil, ErrKeyLength
	}
	cp := make([]byte, 32)
	copy(cp, key)
	return &Encryptor{key: cp}, nil
}

// Encrypt 加密明文字符串，返回 base64url 编码的密文
func (e *Encryptor) Encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", fmt.Errorf("crypto: new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("crypto: new gcm: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize()) // 12 字节
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("crypto: read nonce: %w", err)
	}
	// 密文 = nonce || encrypted
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.URLEncoding.EncodeToString(ciphertext), nil
}

// Decrypt 解密 base64url 编码的密文，返回明文字符串
func (e *Encryptor) Decrypt(encoded string) (string, error) {
	data, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("crypto: base64 decode: %w", err)
	}
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", fmt.Errorf("crypto: new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("crypto: new gcm: %w", err)
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", ErrCiphertextTooShort
	}
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("crypto: decrypt: %w", err)
	}
	return string(plaintext), nil
}
