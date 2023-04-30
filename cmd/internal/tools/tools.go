package tools

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"

	"github.com/nathanhack/ecc/benchmarking"
	"github.com/nathanhack/ecc/linearblock"
	mat "github.com/nathanhack/sparsemat"
)

type SimulationStats struct {
	TypeInfo string
	ECCInfo  string
	Stats    map[float64]benchmarking.Stats
}
type simulationStats struct {
	TypeInfo string
	ECCInfo  string
	Stats    map[string]benchmarking.Stats
}

func (s *SimulationStats) MarshalJSON() ([]byte, error) {
	ss := simulationStats{
		TypeInfo: s.TypeInfo,
		ECCInfo:  s.ECCInfo,
		Stats:    map[string]benchmarking.Stats{},
	}

	for f, stat := range s.Stats {
		ss.Stats[fmt.Sprintf("%v", f)] = stat
	}

	return json.Marshal(ss)
}

func (s *SimulationStats) UnmarshalJSON(bytes []byte) error {
	var ss simulationStats

	err := json.Unmarshal(bytes, &ss)
	if err != nil {
		return err
	}

	s.TypeInfo = ss.TypeInfo
	s.ECCInfo = ss.ECCInfo
	s.Stats = map[float64]benchmarking.Stats{}

	for fs, stat := range ss.Stats {
		f, err := strconv.ParseFloat(fs, 64)
		if err != nil {
			return err
		}
		s.Stats[f] = stat
	}
	return nil
}

func Md5Sum(H mat.SparseMat) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(H.String())))
}

func LoadLinearBlockECC(filepath string) (*linearblock.LinearBlock, error) {
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		return nil, fmt.Errorf("the ECC_JSON_FILE must exist")
	}

	bs, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("error while reading file %v: %v\n", filepath, err)
	}

	var ecc linearblock.LinearBlock
	err = json.Unmarshal(bs, &ecc)
	if err != nil {
		return nil, fmt.Errorf("error while reading file %v: %v\n", filepath, err)
	}

	return &ecc, nil
}

func LoadResults(filepath string) (*SimulationStats, error) {
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		return nil, nil
	}

	bs, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("error while reading file %v: %v\n", filepath, err)
	}

	var stat SimulationStats
	err = json.Unmarshal(bs, &stat)
	if err != nil {
		return nil, fmt.Errorf("error while unmarshalling file %v: %v\n", filepath, err)
	}
	return &stat, nil
}

func SaveResults(filepath string, data *SimulationStats) error {
	bs, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("error serializing csv: %v\n", err)
	}

	err = ioutil.WriteFile(filepath, bs, 0644)
	if err != nil {
		fmt.Errorf("error while saving csv to %v: %v\n", filepath, err)
	}
	return nil
}
