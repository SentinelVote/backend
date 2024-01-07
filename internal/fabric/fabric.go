package fabric

import (
	"bytes"
	"github.com/goccy/go-json"
	"net/http"
)

type fabricUserAuth struct {
	Id     string `json:"id"`
	Secret string `json:"secret"`
}

type fabricUserToken struct {
	Token string `json:"token"`
}

type fabricChaincode struct {
	Method string   `json:"method"`
	Args   []string `json:"args"`
}

const (
	fabricAdminUsername = "admin"
	fabricAdminPassword = "adminpw"
	fabricChaincodeURL  = "http://localhost:8801/invoke/vote-channel/SentinelVote"
	fabricUserEnrollURL = "http://localhost:8801/user/enroll"
)

func FabricGetToken(httpClient *http.Client) (string, error) {
	jsonData, err := json.Marshal(fabricUserAuth{
		Id:     fabricAdminUsername,
		Secret: fabricAdminPassword,
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", fabricUserEnrollURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer")
	res, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	var response = fabricUserToken{}
	err = json.NewDecoder(res.Body).Decode(&response)
	if err != nil {
		return "", err
	}

	return response.Token, nil
}

func FabricPutFoldedPublicKeys(value string) (string, error) {
	httpClient := &http.Client{}
	token, err := FabricGetToken(httpClient)
	jsonData, err := json.Marshal(fabricChaincode{
		Method: "KVContractGo:PutFoldedPublicKeys",
		Args:   []string{value},
	})

	reqFabricStore, err := http.NewRequest("POST", fabricChaincodeURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	reqFabricStore.Header.Set("Content-Type", "application/json")
	reqFabricStore.Header.Set("Authorization", "Bearer "+token)

	resFabricStore, err := httpClient.Do(reqFabricStore)
	if err != nil {
		return "", err
	}
	defer resFabricStore.Body.Close()

	return "OK", nil
}

func FabricPutVote(key string, value string) (string, error) {
	httpClient := &http.Client{}
	token, err := FabricGetToken(httpClient)
	jsonData, err := json.Marshal(fabricChaincode{
		Method: "KVContractGo:PutVote",
		Args:   []string{key, value},
	})

	reqFabricStore, err := http.NewRequest("POST", fabricChaincodeURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	reqFabricStore.Header.Set("Content-Type", "application/json")
	reqFabricStore.Header.Set("Authorization", "Bearer "+token)

	resFabricStore, err := httpClient.Do(reqFabricStore)
	if err != nil {
		return "", err
	}
	defer resFabricStore.Body.Close()

	return "OK", nil
}
