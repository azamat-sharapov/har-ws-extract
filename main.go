package main

import (
	"bufio"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
)

func main() {
	type entry struct {
		ResourceType string `json:"_resourceType"`
		Request      struct {
			Url string `json:"url"`
		}
		WSMessages json.RawMessage `json:"_webSocketMessages"`
	}

	harFile, err := os.Open("localhost.har")
	if err != nil {
		panic(err.Error())
	}

	var result map[string]string
	buf := bufio.NewReader(harFile)
	dec := json.NewDecoder(buf)

	for {
		t, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}

		if v, ok := t.(string); ok && v == "entries" {
			// opening bracket for entries
			_, err := dec.Token()
			if err != nil {
				log.Fatal(err)
				break
			}

			for dec.More() {
				var e entry

				err := dec.Decode(&e)
				if err != nil {
					log.Fatal(err)
				}

				// TODO: more check like request URL, response code, etc.
				if e.ResourceType == "websocket" {
					result, err = convertWsMessages(e.WSMessages)
					if err != nil {
						log.Fatal(err)
					}

					break
				}
			}

			break
		}
	}

	output, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%+v", string(output))
}

func convertWsMessages(wsMessages json.RawMessage) (map[string]string, error) {
	var messages []struct {
		Type string
		Data string
	}

	err := json.Unmarshal(wsMessages, &messages)
	if err != nil {
		return nil, err
	}

	// TODO, give capacity?
	pairs := make(map[string]string, 0)

	for i, msg := range messages {
		if msg.Type != "send" {
			continue
		}

		var data map[string]interface{}

		err := json.Unmarshal([]byte(msg.Data), &data)
		if err != nil {
			return pairs, err
		}

		id, ok := data["id"]
		if !ok {
			continue
		}

		str, err := serialize(data)
		if err != nil {
			return pairs, err
		}

		h := sha1.New()
		h.Write([]byte(str))
		hash := hex.EncodeToString(h.Sum(nil))

		pairs[hash] = ""

		for _, v := range messages[i+1:] {
			if v.Type != "receive" {
				continue
			}

			var receiveData map[string]interface{}

			err := json.Unmarshal([]byte(v.Data), &receiveData)
			if err != nil {
				return pairs, err
			}

			receiveId, ok := receiveData["id"]
			if !ok {
				continue
			}

			if receiveId == id {
				delete(receiveData, "id")

				jsonStr, err := json.Marshal(receiveData)
				if err != nil {
					return pairs, err
				}

				pairs[hash] = string(jsonStr)
				break
			}
		}
	}

	return pairs, err
}

func serialize(input interface{}) (string, error) {
	var str strings.Builder

	switch inputVal := input.(type) {
	case []interface{}:
		for _, v := range inputVal {
			val, err := serialize(v)
			if err != nil {
				return "", err
			}

			_, err = str.WriteString(val)
			if err != nil {
				return "", err
			}
		}
	case map[string]interface{}:
		inputLen := len(inputVal)
		if _, ok := inputVal["id"]; ok {
			inputLen--
		}

		keys := make([]string, inputLen)

		i := 0
		for k, _ := range inputVal {
			// exclude id from serialization
			if k == "id" {
				continue
			}
			keys[i] = k
			i++
		}

		sort.Slice(keys, func(i, j int) bool {
			return keys[i] < keys[j]
		})

		for _, k := range keys {
			v := inputVal[k]

			_, err := str.WriteString(k)
			if err != nil {
				return "", err
			}

			val, err := serialize(v)
			if err != nil {
				return "", err
			}

			_, err = str.WriteString(val)
			if err != nil {
				return "", err
			}
		}
	case float64:
		_, err := str.WriteString(strconv.FormatFloat(inputVal, 'f', -1, 64))
		if err != nil {
			return "", err
		}
	case bool:
		b := "false"
		if inputVal {
			b = "true"
		}
		_, err := str.WriteString(b)
		if err != nil {
			return "", err
		}
	case string:
		_, err := str.WriteString(inputVal)
		if err != nil {
			return "", err
		}
	default:
		_, err := str.WriteString("null")
		if err != nil {
			return "", err
		}
	}

	return str.String(), nil
}
