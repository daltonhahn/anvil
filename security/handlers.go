package security

import (
	"net/http"
	"io"
)

func EncData(plaintext string) ([]byte, error) {
	res1, err1 := EncDataSvc(plaintext, 0)
	if err1 != nil {
		if len(SecConf) < 2 {
			return []byte{}, err1
		} else {
			res2, err2 := EncDataSvc(plaintext, 1)
			if err2 != nil {
				return []byte{},err2
			}
			return res2, nil
		}
	}
	return res1, nil
}

func DecData(input_ciphertext string) ([]byte,error) {
	res1, err1 := DecDataSvc(input_ciphertext, 0)
	if err1 != nil {
		if len(SecConf) < 2 {
			return []byte{}, err1
		} else {
			res2, err2 := DecDataSvc(input_ciphertext, 1)
			if err2 != nil {
				return []byte{},err2
			}
			return res2, nil
		}
	}
	return res1, nil
}

func TLSGetReq(target string, path string, origin string) (*http.Response,error) {
	res1, err1 := TLSGetReqSvc(target, path, origin, 0)
	if err1 != nil {
		if len(SecConf) < 2 {
			return &http.Response{},err1
		} else {
			res2, err2 := TLSGetReqSvc(target, path, origin, 1)
			if err2 != nil {
				return &http.Response{},err2
			}
			return res2, nil
		}
	}
	return res1, nil
}

func TLSPostReq(target string, path string, origin string, options string, body io.Reader) (*http.Response, error) {
	res1, err1 := TLSPostReqSvc(target, path, origin, options, body, 0)
	if err1 != nil {
		if len(SecConf) < 2 {
			return &http.Response{},err1
		} else {
			res2, err2 := TLSPostReqSvc(target, path, origin, options, body, 1)
			if err2 != nil {
				return &http.Response{},err2
			}
			return res2, nil
		}
	}
	return res1, nil
}
