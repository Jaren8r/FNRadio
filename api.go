package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"golang.org/x/sys/windows/registry"
)

type APIClient struct {
	ID     string `json:"id"`
	Secret string `json:"secret"`
}

type APIStation struct {
	ID     string `json:"id"`
	Type   string `json:"type"`
	Source string `json:"source"`
}

type APIBinding struct {
	ID          string `json:"id"`
	StationUser string `json:"station_user"`
	StationID   string `json:"station_id"`
}

type APIUser struct {
	Stations map[string]*APIStation `json:"stations"`
	Bindings map[string]*APIBinding `json:"bindings"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type PartyResponse struct {
	Leader string `json:"leader"`
}

func (c *APIClient) generateAuthHeader() string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(c.ID+":"+c.Secret))
}

func (c *APIClient) Setup() {
	k, err := registry.OpenKey(registry.CURRENT_USER, `SOFTWARE\FNRadio`, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		panic(err)
	}

	value, _, err := k.GetStringValue("APICredentials:" + APIRoot)
	if err == nil {
		split := strings.Split(value, ":")

		c.ID = split[0]
		c.Secret = split[1]
	} else {
		request, err := http.NewRequest(http.MethodPost, APIRoot+"/users", nil)
		if err != nil {
			panic(err)
		}

		response, err := http.DefaultClient.Do(request)
		if err != nil {
			panic(err)
		}

		defer response.Body.Close()

		err = json.NewDecoder(response.Body).Decode(&c)
		if err != nil {
			panic(err)
		}

		err = k.SetStringValue("APICredentials:"+APIRoot, c.ID+":"+c.Secret)
		if err != nil {
			panic(err)
		}
	}

	err = k.Close()
	if err != nil {
		panic(err)
	}
}

func (c *APIClient) GetUser(id string) (*APIUser, error) {
	request, err := http.NewRequest(http.MethodGet, APIRoot+"/users/"+id, nil)
	if err != nil {
		panic(err)
	}

	request.Header.Add("Authorization", c.generateAuthHeader())

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		var errorResponse ErrorResponse

		err = json.NewDecoder(response.Body).Decode(&errorResponse)
		if err != nil {
			return nil, err
		}

		return nil, errors.New(errorResponse.Error)
	}

	user := APIUser{}

	err = json.NewDecoder(response.Body).Decode(&user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (c *APIClient) CreateStation(station *APIStation) error {
	data, err := json.Marshal(station)
	if err != nil {
		return err
	}

	request, err := http.NewRequest(http.MethodPut, APIRoot+"/users/@me/stations/"+url.PathEscape(station.ID), bytes.NewReader(data))
	if err != nil {
		return err
	}

	request.Header.Add("Authorization", c.generateAuthHeader())

	request.Header.Add("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusNoContent {
		var errorResponse ErrorResponse

		err := json.NewDecoder(response.Body).Decode(&errorResponse)
		if err != nil {
			return err
		}

		return errors.New(errorResponse.Error)
	}

	return nil
}

func (c *APIClient) DeleteStation(station *APIStation) error {
	request, err := http.NewRequest(http.MethodDelete, APIRoot+"/users/@me/stations/"+url.PathEscape(station.ID), nil)
	if err != nil {
		return err
	}

	request.Header.Add("Authorization", c.generateAuthHeader())

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusNoContent {
		var errorResponse ErrorResponse

		err := json.NewDecoder(response.Body).Decode(&errorResponse)
		if err != nil {
			return err
		}

		return errors.New(errorResponse.Error)
	}

	return nil
}

func (c *APIClient) AddToQueue(station *APIStation, source string) error {
	data, err := json.Marshal(map[string]string{"source": source})
	if err != nil {
		return err
	}

	request, err := http.NewRequest(http.MethodPut, APIRoot+"/users/@me/stations/"+url.PathEscape(station.ID)+"/queue", bytes.NewReader(data))
	if err != nil {
		return err
	}

	request.Header.Add("Authorization", c.generateAuthHeader())
	request.Header.Add("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusNoContent {
		var errorResponse ErrorResponse

		err := json.NewDecoder(response.Body).Decode(&errorResponse)
		if err != nil {
			return err
		}

		return errors.New(errorResponse.Error)
	}

	return nil
}

func (c *APIClient) CreateBinding(binding *APIBinding) error {
	data, err := json.Marshal(binding)
	if err != nil {
		return err
	}

	request, err := http.NewRequest(http.MethodPut, APIRoot+"/users/@me/bindings/"+url.PathEscape(binding.ID), bytes.NewReader(data))
	if err != nil {
		return err
	}

	request.Header.Add("Authorization", c.generateAuthHeader())
	request.Header.Add("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusNoContent {
		var errorResponse ErrorResponse

		err := json.NewDecoder(response.Body).Decode(&errorResponse)
		if err != nil {
			return err
		}

		return errors.New(errorResponse.Error)
	}

	return nil
}

func (c *APIClient) DeleteBinding(binding *APIBinding) error {
	request, err := http.NewRequest(http.MethodDelete, APIRoot+"/users/@me/bindings/"+url.PathEscape(binding.ID), nil)
	if err != nil {
		return err
	}

	request.Header.Add("Authorization", c.generateAuthHeader())

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusNoContent {
		return errors.New("status code " + strconv.Itoa(response.StatusCode))
	}

	return nil
}

func (c *APIClient) SetParty(party Party) (string, error) {
	data, err := json.Marshal(party)
	if err != nil {
		return "", err
	}

	request, err := http.NewRequest(http.MethodPost, APIRoot+"/users/@me/party", bytes.NewReader(data))
	if err != nil {
		return "", err
	}

	request.Header.Add("Authorization", c.generateAuthHeader())

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return "", err
	}

	defer response.Body.Close()

	if response.StatusCode == http.StatusNoContent {
		return "", nil
	}

	if response.StatusCode == http.StatusOK {
		var partyResponse PartyResponse

		err := json.NewDecoder(response.Body).Decode(&partyResponse)
		if err != nil {
			return "", err
		}

		return partyResponse.Leader, nil
	}

	var errorResponse ErrorResponse

	err = json.NewDecoder(response.Body).Decode(&errorResponse)
	if err != nil {
		return "", err
	}

	return "", errors.New(errorResponse.Error)
}
