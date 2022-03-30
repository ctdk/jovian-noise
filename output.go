package main

import (
	"encoding/json"
	"os"
)

func outputJSON(jData *jupiterData) error {
	if j, err := json.MarshalIndent(jData, "", "\t"); err != nil {
		return err
	} else {
		os.Stdout.Write(j)
	}
	return nil
}

func outputText(jData *jupiterData) error {

}
