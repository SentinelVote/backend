package foldpub

// Standard library on top, third-party packages below.
import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/goccy/go-json"
	"github.com/zbohm/lirisi/client"
	"github.com/zbohm/lirisi/ring"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

type fabricUserAuth struct {
	Id     string `json:"id"`
	Secret string `json:"secret"`
}

type fabricUserToken struct {
	Token string `json:"token"`
}

type fabricChaincodeRequest struct {
	Method string   `json:"method"`
	Args   []string `json:"args"`
}

const (
	fabricAdminUsername = "admin"
	fabricAdminPassword = "adminpw"
	fabricChaincodeURL  = "http://localhost:8801/invoke/vote-channel/SentinelVote"
	fabricUserEnrollURL = "http://localhost:8801/user/enroll"
)

func GetToken(httpClient *http.Client) (string, error) {
	jsonData, err := json.Marshal(
		fabricUserAuth{
			Id:     fabricAdminUsername,
			Secret: fabricAdminPassword,
		},
	)
	if err != nil {
		return "", err
	}

	request, err := http.NewRequest("POST", fabricUserEnrollURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer")
	response, err := httpClient.Do(request)
	if err != nil {
		return "", err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			panic(err)
		}
	}(response.Body)

	var t = fabricUserToken{}
	err = json.NewDecoder(response.Body).Decode(&t)
	if err != nil {
		return "", err
	}

	return t.Token, nil
}

func PutFoldedPublicKeys(conn *sqlite.Conn) (string, error) {

	// Get public keys.
	var publicKeys []string
	query := `SELECT public_key FROM users WHERE is_central_authority = FALSE AND public_key != '';`
	err := sqlitex.Execute(conn, query,
		&sqlitex.ExecOptions{
			ResultFunc: func(stmt *sqlite.Stmt) error {
				publicKeys = append(publicKeys, stmt.ColumnText(0))
				return nil
			},
		})
	if err != nil {
		return "", err
	}

	// Convert public keys to byte arrays.
	var publicKeysContent [][]byte
	for _, key := range publicKeys {
		publicKeysContent = append(publicKeysContent, []byte(key))
	}

	// Fold public keys.
	status, foldedPublicKeys := client.FoldPublicKeys(publicKeysContent, "sha3-256", "PEM", "hashes")
	if status != ring.Success {
		return "", fmt.Errorf("client.FoldPublicKeys() failed: status %v", status)
	}

	// Send folded public keys to the blockchain.
	httpClient := &http.Client{}
	token, err := GetToken(httpClient)
	jsonData, err := json.Marshal(fabricChaincodeRequest{
		Method: "KVContractGo:PutFoldedPublicKeys",
		Args:   []string{string(foldedPublicKeys)},
	})

	request, err := http.NewRequest("POST", fabricChaincodeURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+token)

	response, err := httpClient.Do(request)
	if err != nil {
		return "", err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			panic(err)
		}
	}(response.Body)

	return "OK", nil
}
