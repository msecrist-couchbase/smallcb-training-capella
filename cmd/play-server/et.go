package main                                                                                                                                                                                              
import (                                                                                                                                                                                                    
    "encoding/base64"                                                                                                                                                                                       
    "crypto/aes"                                                                                                                                                                                            
    "crypto/cipher" 
	"crypto/rand"
	"io"                                                                                                                                                                                                                                                                                                                                                                                      
)
                                                                                                                                                                                                                                                                                                                                                                                                                  
func encodeBase64(b []byte) string {                                                                                                                                                                        
    return base64.StdEncoding.EncodeToString(b)                                                                                                                                                             
}                                                                                                                                                                                                           
                                                                                                                                                                                                            
func decodeBase64(s string) []byte {                                                                                                                                                                        
    data, err := base64.StdEncoding.DecodeString(s)                                                                                                                                                         
    if err != nil { panic(err) }                                                                                                                                                                            
    return data                                                                                                                                                                                             
}                                                                                                                                                                                                           

// ref: https://www.melvinvivas.com/how-to-encrypt-and-decrypt-data-using-aes
func Encrypt(key, text string) string {
	block, err := aes.NewCipher([]byte(key))
	if err != nil { panic(err) }
	aesGCM, err := cipher.NewGCM(block)
	if err != nil { panic(err.Error())}
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil { panic(err.Error())}
	plaintext := []byte(text) 
	ciphertext := aesGCM.Seal(nonce, nonce, plaintext, nil)
	etstring := encodeBase64(ciphertext)
	return etstring
}
func Decrypt(key, text string) string {
	block, err := aes.NewCipher([]byte(key))
	if err != nil { panic(err) }
	aesGCM, err := cipher.NewGCM(block)
	if err != nil { panic(err.Error())}
	ciphertext := decodeBase64(text)
	nonceSize := aesGCM.NonceSize()
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil { panic(err)}
	return string(plaintext)
}
