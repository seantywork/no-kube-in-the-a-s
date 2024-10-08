package modules

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha512"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"math/big"
	mathrand "math/rand"
	"os"
	"os/exec"
	"strings"
	"time"

	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"

	goya "github.com/goccy/go-yaml"
)

type CertSet struct {
	RootCertPEM   []byte
	RootKeyPEM    []byte
	RootPubPEM    []byte
	ClientCertPEM []byte
	ClientKeyPEM  []byte
	ClientPubPEM  []byte
	ServCertPEM   []byte
	ServKeyPEM    []byte
	ServPubPEM    []byte
}

func RandomHex(n int) (string, error) {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func CheckIfSliceContains[T comparable](slice []T, ele T) bool {

	hit := false

	for i := 0; i < len(slice); i++ {

		if slice[i] == ele {

			hit = true

			return hit
		}

	}

	return hit

}

func GetKubeConfigPath() (string, error) {

	var kube_config_path string

	cmd := exec.Command(".npia/get_kubeconfig_path")

	out, err := cmd.Output()

	if err != nil {

		return "", fmt.Errorf("failed to get kube config path: %s", err.Error())

	}

	strout := string(out)

	ret_strout := strings.ReplaceAll(strout, "\n", "")

	ret_strout = strings.ReplaceAll(ret_strout, " ", "")

	kube_config_path = ret_strout

	return kube_config_path, nil
}

func GetContextUserPrivateKeyBytes(context_nm string) ([]byte, error) {

	var kube_config map[interface{}]interface{}

	var ret_byte []byte

	kube_config_path, err := GetKubeConfigPath()

	if err != nil {
		return ret_byte, fmt.Errorf("failed to get context user private key: %s", err.Error())
	}

	kube_config_file_byte, err := os.ReadFile(kube_config_path)

	err = goya.Unmarshal(kube_config_file_byte, &kube_config)

	if err != nil {
		return ret_byte, fmt.Errorf("failed to get context user private key: %s", err.Error())
	}

	contexts_len := len(kube_config["contexts"].([]interface{}))

	context_user_nm := ""

	for i := 0; i < contexts_len; i++ {

		if kube_config["contexts"].([]interface{})[i].(map[string]interface{})["name"].(string) == context_nm {

			context_user_nm = kube_config["contexts"].([]interface{})[i].(map[string]interface{})["context"].(map[string]interface{})["user"].(string)

			break
		}

	}

	if context_user_nm == "" {
		return ret_byte, fmt.Errorf("failed to get context user private key: %s", "matching user not found")
	}

	user_len := len(kube_config["users"].([]interface{}))

	var user_priv_key_data []byte

	for i := 0; i < user_len; i++ {

		if kube_config["users"].([]interface{})[i].(map[string]interface{})["name"].(string) == context_user_nm {

			tmp_base64, okay := kube_config["users"].([]interface{})[i].(map[string]interface{})["user"].(map[string]interface{})["client-key-data"].(string)

			if !okay {
				return ret_byte, fmt.Errorf("failed to get context user private key: %s", "no key data")
			}

			dec_base64, err := base64.StdEncoding.DecodeString(tmp_base64)

			user_priv_key_data = dec_base64

			if err != nil {
				return ret_byte, fmt.Errorf("failed to get context user private key: %s", err.Error())
			}

			break
		}

	}

	ret_byte = user_priv_key_data

	return ret_byte, nil
}

func GetContextUserPublicKeyBytes(context_nm string) ([]byte, error) {

	var kube_config map[interface{}]interface{}

	var ret_byte []byte

	kube_config_path, err := GetKubeConfigPath()

	if err != nil {
		return ret_byte, fmt.Errorf("failed to get context user public key: %s", err.Error())
	}

	kube_config_file_byte, err := os.ReadFile(kube_config_path)

	err = goya.Unmarshal(kube_config_file_byte, &kube_config)

	if err != nil {
		return ret_byte, fmt.Errorf("failed to get context user public key: %s", err.Error())
	}

	contexts_len := len(kube_config["contexts"].([]interface{}))

	context_user_nm := ""

	for i := 0; i < contexts_len; i++ {

		if kube_config["contexts"].([]interface{})[i].(map[string]interface{})["name"].(string) == context_nm {

			context_user_nm = kube_config["contexts"].([]interface{})[i].(map[string]interface{})["context"].(map[string]interface{})["user"].(string)

			break
		}

	}

	if context_user_nm == "" {
		return ret_byte, fmt.Errorf("failed to get context user public key: %s", "matching user not found")
	}

	user_len := len(kube_config["users"].([]interface{}))

	var user_pub_key_data []byte

	user_certificate_data := ""

	for i := 0; i < user_len; i++ {

		if kube_config["users"].([]interface{})[i].(map[string]interface{})["name"].(string) == context_user_nm {

			tmp_base64, okay := kube_config["users"].([]interface{})[i].(map[string]interface{})["user"].(map[string]interface{})["client-certificate-data"].(string)

			if !okay {
				return ret_byte, fmt.Errorf("failed to get context user public key: %s", "no key data")
			}

			dec_base64, err := base64.StdEncoding.DecodeString(tmp_base64)

			user_certificate_data = string(dec_base64)

			if err != nil {
				return ret_byte, fmt.Errorf("failed to get context user public key: %s", err.Error())
			}

			break
		}

	}

	if user_certificate_data == "" {
		return ret_byte, fmt.Errorf("failed to get context user public key: %s", "no matching user key")
	}

	block, _ := pem.Decode([]byte(user_certificate_data))
	var cert *x509.Certificate
	cert, err = x509.ParseCertificate(block.Bytes)

	if err != nil {
		return ret_byte, fmt.Errorf("failed to get context user public key: %s", err.Error())
	}

	rsaPublicKey := cert.PublicKey.(*rsa.PublicKey)

	user_pub_key_data, err = PublicKeyToBytes(rsaPublicKey)

	if err != nil {
		return ret_byte, fmt.Errorf("failed to get context user public key: %s", err.Error())
	}

	ret_byte = user_pub_key_data

	return ret_byte, nil

}

func GetContextUserPublicKeyBytes_Detached(config_b []byte, context_nm string) ([]byte, error) {

	var kube_config map[interface{}]interface{}

	var ret_byte []byte

	err := goya.Unmarshal(config_b, &kube_config)

	if err != nil {
		return ret_byte, fmt.Errorf("failed to get context user public key: %s", err.Error())
	}

	contexts_len := len(kube_config["contexts"].([]interface{}))

	context_user_nm := ""

	for i := 0; i < contexts_len; i++ {

		if kube_config["contexts"].([]interface{})[i].(map[string]interface{})["name"].(string) == context_nm {

			context_user_nm = kube_config["contexts"].([]interface{})[i].(map[string]interface{})["context"].(map[string]interface{})["user"].(string)

			break
		}

	}

	if context_user_nm == "" {
		return ret_byte, fmt.Errorf("failed to get context user public key: %s", "matching user not found")
	}

	user_len := len(kube_config["users"].([]interface{}))

	var user_pub_key_data []byte

	user_certificate_data := ""

	for i := 0; i < user_len; i++ {

		if kube_config["users"].([]interface{})[i].(map[string]interface{})["name"].(string) == context_user_nm {

			tmp_base64, okay := kube_config["users"].([]interface{})[i].(map[string]interface{})["user"].(map[string]interface{})["client-certificate-data"].(string)

			if !okay {
				return ret_byte, fmt.Errorf("failed to get context user public key: %s", "no key data")
			}

			dec_base64, err := base64.StdEncoding.DecodeString(tmp_base64)

			user_certificate_data = string(dec_base64)

			if err != nil {
				return ret_byte, fmt.Errorf("failed to get context user public key: %s", err.Error())
			}

			break
		}

	}

	if user_certificate_data == "" {
		return ret_byte, fmt.Errorf("failed to get context user public key: %s", "no matching user key")
	}

	block, _ := pem.Decode([]byte(user_certificate_data))
	var cert *x509.Certificate
	cert, err = x509.ParseCertificate(block.Bytes)

	if err != nil {
		return ret_byte, fmt.Errorf("failed to get context user public key: %s", err.Error())
	}

	rsaPublicKey := cert.PublicKey.(*rsa.PublicKey)

	user_pub_key_data, err = PublicKeyToBytes(rsaPublicKey)

	if err != nil {
		return ret_byte, fmt.Errorf("failed to get context user public key: %s", err.Error())
	}

	ret_byte = user_pub_key_data

	return ret_byte, nil

}

func GetContextClusterPublicKeyBytes(context_nm string) ([]byte, error) {

	var kube_config map[interface{}]interface{}

	var ret_byte []byte

	kube_config_path, err := GetKubeConfigPath()

	if err != nil {
		return ret_byte, fmt.Errorf("failed to get context cluster public key: %s", err.Error())
	}

	kube_config_file_byte, err := os.ReadFile(kube_config_path)

	err = goya.Unmarshal(kube_config_file_byte, &kube_config)

	if err != nil {
		return ret_byte, fmt.Errorf("failed to get context cluster public key: %s", err.Error())
	}

	contexts_len := len(kube_config["contexts"].([]interface{}))

	context_cluster_nm := ""

	for i := 0; i < contexts_len; i++ {

		if kube_config["contexts"].([]interface{})[i].(map[string]interface{})["name"].(string) == context_nm {

			context_cluster_nm = kube_config["contexts"].([]interface{})[i].(map[string]interface{})["context"].(map[string]interface{})["cluster"].(string)

			break
		}

	}

	if context_cluster_nm == "" {
		return ret_byte, fmt.Errorf("failed to get context user public key: %s", "matching cluster not found")
	}

	clusters_len := len(kube_config["clusters"].([]interface{}))

	var cluster_pub_key_data []byte

	cluster_certificate_data := ""

	for i := 0; i < clusters_len; i++ {

		if kube_config["clusters"].([]interface{})[i].(map[string]interface{})["name"].(string) == context_cluster_nm {

			tmp_base64, okay := kube_config["clusters"].([]interface{})[i].(map[string]interface{})["cluster"].(map[string]interface{})["certificate-authority-data"].(string)

			if !okay {
				return ret_byte, fmt.Errorf("failed to get context cluster public key: %s", "no key data")
			}

			dec_base64, err := base64.StdEncoding.DecodeString(tmp_base64)

			cluster_certificate_data = string(dec_base64)

			if err != nil {
				return ret_byte, fmt.Errorf("failed to get context cluster public key: %s", err.Error())
			}

			break
		}

	}

	if cluster_certificate_data == "" {
		return ret_byte, fmt.Errorf("failed to get context user public key: %s", "no matching user key")
	}

	block, _ := pem.Decode([]byte(cluster_certificate_data))
	var cert *x509.Certificate
	cert, err = x509.ParseCertificate(block.Bytes)

	if err != nil {
		return ret_byte, fmt.Errorf("failed to get context cluster public key: %s", err.Error())
	}

	rsaPublicKey := cert.PublicKey.(*rsa.PublicKey)

	cluster_pub_key_data, err = PublicKeyToBytes(rsaPublicKey)

	if err != nil {
		return ret_byte, fmt.Errorf("failed to get context cluster public key: %s", err.Error())
	}

	ret_byte = cluster_pub_key_data

	return ret_byte, nil

}

func GetContextUserCertificateBytes(context_nm string) ([]byte, error) {

	var kube_config map[interface{}]interface{}

	var ret_byte []byte

	kube_config_path, err := GetKubeConfigPath()

	if err != nil {
		return ret_byte, fmt.Errorf("failed to get context user certificate: %s", err.Error())
	}

	kube_config_file_byte, err := os.ReadFile(kube_config_path)

	err = goya.Unmarshal(kube_config_file_byte, &kube_config)

	if err != nil {
		return ret_byte, fmt.Errorf("failed to get context user certificate: %s", err.Error())
	}

	contexts_len := len(kube_config["contexts"].([]interface{}))

	context_user_nm := ""

	for i := 0; i < contexts_len; i++ {

		if kube_config["contexts"].([]interface{})[i].(map[string]interface{})["name"].(string) == context_nm {

			context_user_nm = kube_config["contexts"].([]interface{})[i].(map[string]interface{})["context"].(map[string]interface{})["user"].(string)

			break
		}

	}

	if context_user_nm == "" {
		return ret_byte, fmt.Errorf("failed to get context user certificate: %s", "matching user not found")
	}

	user_len := len(kube_config["users"].([]interface{}))

	user_certificate_data := ""

	for i := 0; i < user_len; i++ {

		if kube_config["users"].([]interface{})[i].(map[string]interface{})["name"].(string) == context_user_nm {

			tmp_base64, okay := kube_config["users"].([]interface{})[i].(map[string]interface{})["user"].(map[string]interface{})["client-certificate-data"].(string)

			if !okay {
				return ret_byte, fmt.Errorf("failed to get context user certificate: %s", "no key data")
			}

			dec_base64, err := base64.StdEncoding.DecodeString(tmp_base64)

			user_certificate_data = string(dec_base64)

			if err != nil {
				return ret_byte, fmt.Errorf("failed to get context user certificate: %s", err.Error())
			}

			break
		}

	}

	if user_certificate_data == "" {
		return ret_byte, fmt.Errorf("failed to get context user certificate: %s", "no matching user key")
	}

	ret_byte = []byte(user_certificate_data)

	return ret_byte, nil

}

func GetContextUserCertificateBytes_Detached(config_b []byte, context_nm string) ([]byte, error) {

	var kube_config map[interface{}]interface{}

	var ret_byte []byte

	err := goya.Unmarshal(config_b, &kube_config)

	if err != nil {
		return ret_byte, fmt.Errorf("failed to get context user certificate: %s", err.Error())
	}

	contexts_len := len(kube_config["contexts"].([]interface{}))

	context_user_nm := ""

	for i := 0; i < contexts_len; i++ {

		if kube_config["contexts"].([]interface{})[i].(map[string]interface{})["name"].(string) == context_nm {

			context_user_nm = kube_config["contexts"].([]interface{})[i].(map[string]interface{})["context"].(map[string]interface{})["user"].(string)

			break
		}

	}

	if context_user_nm == "" {
		return ret_byte, fmt.Errorf("failed to get context user certificate: %s", "matching user not found")
	}

	user_len := len(kube_config["users"].([]interface{}))

	user_certificate_data := ""

	for i := 0; i < user_len; i++ {

		if kube_config["users"].([]interface{})[i].(map[string]interface{})["name"].(string) == context_user_nm {

			tmp_base64, okay := kube_config["users"].([]interface{})[i].(map[string]interface{})["user"].(map[string]interface{})["client-certificate-data"].(string)

			if !okay {
				return ret_byte, fmt.Errorf("failed to get context user certificate: %s", "no key data")
			}

			dec_base64, err := base64.StdEncoding.DecodeString(tmp_base64)

			user_certificate_data = string(dec_base64)

			if err != nil {
				return ret_byte, fmt.Errorf("failed to get context user certificate: %s", err.Error())
			}

			break
		}

	}

	if user_certificate_data == "" {
		return ret_byte, fmt.Errorf("failed to get context user certificate: %s", "no matching user key")
	}

	ret_byte = []byte(user_certificate_data)

	return ret_byte, nil

}

func GenerateKeyPair(bits int) (*rsa.PrivateKey, *rsa.PublicKey, error) {

	var pubkey *rsa.PublicKey

	privkey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return privkey, pubkey, fmt.Errorf("failed to gen key pair: %s", err.Error())
	}

	pubkey = &privkey.PublicKey

	return privkey, pubkey, nil
}

func PrivateKeyToBytes(priv *rsa.PrivateKey) ([]byte, error) {
	var ret_byte []byte
	privBytes := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(priv),
		},
	)

	if privBytes == nil {
		return ret_byte, fmt.Errorf("failed to encode priv key to bytes: %s", "invalid")
	}

	return privBytes, nil
}

func PublicKeyToBytes(pub *rsa.PublicKey) ([]byte, error) {

	var ret_byte []byte

	pubASN1, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return pubASN1, fmt.Errorf("failed to encode pub key to bytes: %s", err.Error())
	}

	pubBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: pubASN1,
	})

	if pubBytes == nil {
		return ret_byte, fmt.Errorf("failed to encode priv key to bytes: %s", "invalid")
	}

	return pubBytes, nil
}

func CertToBytes(crt *x509.Certificate) ([]byte, error) {

	cert_b := crt.Raw

	certBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert_b,
	})

	if certBytes == nil {

		return nil, fmt.Errorf("failed to encode cert to bytes: %s", "invalid")
	}

	return certBytes, nil

}

func BytesToPrivateKey(priv []byte) (*rsa.PrivateKey, error) {

	var privkey *rsa.PrivateKey

	block, _ := pem.Decode(priv)
	enc := x509.IsEncryptedPEMBlock(block)
	b := block.Bytes
	var err error
	if enc {
		fmt.Println("is encrypted pem block")
		b, err = x509.DecryptPEMBlock(block, nil)
		if err != nil {
			return privkey, fmt.Errorf("failed to decode bytes to priv key: %s", err.Error())
		}
	}
	key, err := x509.ParsePKCS1PrivateKey(b)
	if err != nil {
		return privkey, fmt.Errorf("failed to decode bytes to priv key: %s", err.Error())
	}
	return key, nil
}

func BytesToPublicKey(pub []byte) (*rsa.PublicKey, error) {

	var pubkey *rsa.PublicKey

	block, _ := pem.Decode(pub)
	enc := x509.IsEncryptedPEMBlock(block)
	b := block.Bytes
	var err error
	if enc {
		fmt.Println("is encrypted pem block")
		b, err = x509.DecryptPEMBlock(block, nil)
		if err != nil {
			return pubkey, fmt.Errorf("failed to decode bytes to pub key: %s", err.Error())
		}
	}
	ifc, err := x509.ParsePKIXPublicKey(b)
	if err != nil {
		return pubkey, fmt.Errorf("failed to decode bytes to pub key: %s", err.Error())
	}
	key, ok := ifc.(*rsa.PublicKey)
	if !ok {
		return pubkey, fmt.Errorf("failed to decode bytes to pub key: %s", err.Error())
	}
	return key, nil
}

func BytesToCert(cert []byte) (*x509.Certificate, error) {

	var ret_crt *x509.Certificate

	block, _ := pem.Decode(cert)

	ifc, err := x509.ParseCertificate(block.Bytes)

	if err != nil {

		return ret_crt, fmt.Errorf("failed to decode bytes to certificate: %s", err.Error())
	}

	ret_crt = ifc

	return ret_crt, nil
}

func EncryptWithPublicKey(msg []byte, pub *rsa.PublicKey) ([]byte, error) {
	hash := sha512.New()
	ciphertext, err := rsa.EncryptOAEP(hash, rand.Reader, pub, msg, nil)
	if err != nil {
		return ciphertext, fmt.Errorf("failed to encrypt with public key: %s", err.Error())
	}
	return ciphertext, nil
}

func DecryptWithPrivateKey(ciphertext []byte, priv *rsa.PrivateKey) ([]byte, error) {
	hash := sha512.New()
	plaintext, err := rsa.DecryptOAEP(hash, rand.Reader, priv, ciphertext, nil)
	if err != nil {
		return plaintext, fmt.Errorf("failed to decrypt with private key: %s", err.Error())
	}
	return plaintext, nil
}

func EncryptWithSymmetricKey(key []byte, file_byte []byte) ([]byte, error) {

	var ret_byte []byte

	c, err := aes.NewCipher(key)

	if err != nil {
		return ret_byte, fmt.Errorf("failed to encrypt with symmetric key: %s", err.Error())

	}

	gcm, err := cipher.NewGCM(c)

	if err != nil {
		return ret_byte, fmt.Errorf("failed to encrypt with symmetric key: %s", err.Error())

	}

	nonce := make([]byte, gcm.NonceSize())

	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return ret_byte, fmt.Errorf("failed to encrypt with symmetric key: %s", err.Error())
	}

	enc_file := gcm.Seal(nonce, nonce, file_byte, nil)

	ret_byte = enc_file

	return ret_byte, nil

}

func DecryptWithSymmetricKey(key []byte, file_byte []byte) ([]byte, error) {

	var ret_byte []byte

	c, err := aes.NewCipher(key)
	if err != nil {
		return ret_byte, fmt.Errorf("failed to decrypt with symmetric key: %s", err.Error())
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return ret_byte, fmt.Errorf("failed to decrypt with symmetric key: %s", err.Error())
	}

	nonceSize := gcm.NonceSize()
	if len(file_byte) < nonceSize {
		return ret_byte, fmt.Errorf("failed to decrypt with symmetric key: %s", err.Error())
	}

	nonce, ciphertext := file_byte[:nonceSize], file_byte[nonceSize:]
	plain_file, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return ret_byte, fmt.Errorf("failed to decrypt with symmetric key: %s", err.Error())
	}

	ret_byte = plain_file

	return ret_byte, nil
}

func GetRandIntInRange(min int, max int) int {

	mathrand.Seed(time.Now().UnixNano())

	return mathrand.Intn(max-min+1) + min
}

func GenKeyPair(keylen int) (*rsa.PrivateKey, *rsa.PublicKey) {

	bitSize := keylen

	key, err := rsa.GenerateKey(rand.Reader, bitSize)
	if err != nil {
		panic(err)
	}

	pub := key.Public()

	return key, pub.(*rsa.PublicKey)
}

func CertTemplate(cn string, validYear int) (*x509.Certificate, error) {
	// generate a random serial number (a real cert authority would have some logic behind this)
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, errors.New("failed to generate serial number: " + err.Error())
	}

	if err != nil {

		return nil, fmt.Errorf("failed to gen cert tmpl: %s", err.Error())
	}

	tmpl := x509.Certificate{
		SerialNumber:          serialNumber,
		Subject:               pkix.Name{CommonName: cn},
		SignatureAlgorithm:    x509.SHA256WithRSA,
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour * 24 * 30 * 12 * time.Duration(validYear)),
		BasicConstraintsValid: true,
		DNSNames: []string{
			cn,
		},
	}
	return &tmpl, nil
}

func CreateCert(template, parent *x509.Certificate, pub interface{}, parentPriv interface{}) (cert *x509.Certificate, certPEM []byte, err error) {

	certDER, err := x509.CreateCertificate(rand.Reader, template, parent, pub, parentPriv)
	if err != nil {
		return
	}
	// parse the resulting certificate so we can use it again
	cert, err = x509.ParseCertificate(certDER)
	if err != nil {
		return
	}
	// PEM encode the certificate (this is a standard TLS encoding)
	b := pem.Block{Type: "CERTIFICATE", Bytes: certDER}
	certPEM = pem.EncodeToMemory(&b)

	return
}

func NewCertsPipeline(server_name string, client_name string) *CertSet {
	// generate a new key-pair
	rootKey, rootPub := GenKeyPair(4096)

	rootCertTmpl, err := CertTemplate("ca", 10)
	if err != nil {
		log.Fatalf("creating cert template: %v", err)
	}

	// this cert will be the CA that we will use to sign the server cert
	rootCertTmpl.IsCA = true
	// describe what the certificate will be used for
	rootCertTmpl.KeyUsage = x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature
	rootCertTmpl.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth}

	log.Println("----------root cert")

	rootCert, rootCertPEM, err := CreateCert(rootCertTmpl, rootCertTmpl, rootPub, rootKey)
	if err != nil {
		log.Fatalf("error creating cert: %v", err)
	}

	// provide the private key and the cert
	rootKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(rootKey),
	})

	rootPubPEM := pem.EncodeToMemory(&pem.Block{
		Type: "RSA PUBLIC KEY", Bytes: x509.MarshalPKCS1PublicKey(rootPub),
	})

	/*******************************************************************
	Server Cert
	*******************************************************************/

	// create a key-pair for the server
	servKey, servPub := GenKeyPair(4096)

	log.Println("----------server cert")
	// create a template for the server
	servCertTmpl, err := CertTemplate(server_name, 1)
	if err != nil {
		log.Fatalf("creating cert template: %v", err)
	}
	servCertTmpl.KeyUsage = x509.KeyUsageDigitalSignature
	servCertTmpl.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}

	// create a certificate which wraps the server's public key, sign it with the root private key
	_, servCertPEM, err := CreateCert(servCertTmpl, rootCert, servPub, rootKey)
	if err != nil {
		log.Fatalf("error creating cert: %v", err)
	}

	// provide the private key and the cert
	servKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(servKey),
	})

	servPubPEM := pem.EncodeToMemory(&pem.Block{
		Type: "RSA PUBLIC KEY", Bytes: x509.MarshalPKCS1PublicKey(servPub),
	})

	/*******************************************************************
	Client Cert
	*******************************************************************/

	// create a key-pair for the client
	clientKey, clientPub := GenKeyPair(4096)

	log.Println("----------client cert")

	// create a template for the client
	clientCertTmpl, err := CertTemplate(client_name, 1)
	if err != nil {
		log.Fatalf("creating cert template: %v", err)
	}
	clientCertTmpl.KeyUsage = x509.KeyUsageDigitalSignature
	clientCertTmpl.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}

	// the root cert signs the cert by again providing its private key
	_, clientCertPEM, err := CreateCert(clientCertTmpl, rootCert, clientPub, rootKey)
	if err != nil {
		log.Fatalf("error creating cert: %v", err)
	}

	// encode and load the cert and private key for the client
	clientKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(clientKey),
	})

	clientPubPEM := pem.EncodeToMemory(&pem.Block{
		Type: "RSA PUBLIC KEY", Bytes: x509.MarshalPKCS1PublicKey(clientPub),
	})

	cs := CertSet{
		RootCertPEM:   rootCertPEM,
		RootKeyPEM:    rootKeyPEM,
		RootPubPEM:    rootPubPEM,
		ClientCertPEM: clientCertPEM,
		ClientKeyPEM:  clientKeyPEM,
		ClientPubPEM:  clientPubPEM,
		ServCertPEM:   servCertPEM,
		ServKeyPEM:    servKeyPEM,
		ServPubPEM:    servPubPEM,
	}

	return &cs
}
