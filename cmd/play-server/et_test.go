package main

import (
    "testing"
    "fmt"
    "strings"
)

type etTest struct {
    encryptkey, cleartext, expected string
}

var etTests = []etTest{
    etTest{"92345678901234567890123456789019", "MyPassword!", "MyPassword!"},
    etTest{"92345678901234567890123456789019", "", ""},
}


func TestET(t *testing.T) {
    for _, test := range etTests {
        encryptext := Encrypt(test.encryptkey, test.cleartext)
        decryptedtext := Decrypt(test.encryptkey, encryptext)
        if decryptedtext != test.expected {
            t.Fatalf(`Encrypt(key, cleartext) %s != Decrypt(key,encryptext) %s`, test.expected, decryptedtext)
        }
    }
}

type panicTest struct {
    encryptkey, decryptkey, cleartext, expected string
}

var panicTests = []panicTest{
    panicTest{"92345678901234567890123456789019", "92345678901234567890123456789010", "MyPassword!", "cipher: message authentication failed"},
    panicTest{"", "", "MyPassword!", "invalid key size 0"},
    panicTest{"92345678901234567890123456789019", "923456789012345678901234567890119", "MyPassword!", "invalid key size"},
}

func TestPanic(t *testing.T) {
    for _, test := range panicTests {
        assertPanic(t, EncDec, test.encryptkey, test.decryptkey, test.cleartext, test.expected)
    }
}

func assertPanic(t *testing.T, f func(string, string, string), encryptkey string, decryptkey string, cleartext string, expected string) {
    defer func() {
        if err := recover(); err == nil {
            t.Errorf("Expected code did not panic %q, %q, %q", encryptkey, decryptkey, cleartext)
        } else {
            if ! strings.Contains(fmt.Sprintf("%q",err), expected) {
                t.Errorf("Not matching expected panic %q != %q", err, expected)
            }
        }
    }()
    f(encryptkey, decryptkey, cleartext)
}

func EncDec(encryptkey string, decryptkey string, cleartext string) {
    encryptext := Encrypt(encryptkey, cleartext)
    decryptedtext := Decrypt(decryptkey, encryptext)
    if decryptedtext != cleartext { fmt.Println("No match!")}
}
