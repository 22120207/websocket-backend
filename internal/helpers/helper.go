package helpers

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

func InitLogger() {
	log.SetOutput(os.Stdout)
	log.SetLevel(log.DebugLevel)
}

func ExcuteCommand(s ...string) ([]byte, error) {

	// Gen command and Remove duplicate whitespace from a command
	space := regexp.MustCompile(`\s+`)
	command := space.ReplaceAllString(fmt.Sprint(strings.Join(s[:], " ")), " ")
	if len(command) == 0 {
		return nil, nil
	}
	log.Infof("_run: %s", command)

	// Chuyển CombinedOutput() => Output()
	str, err := exec.Command("bash", "-c", command).CombinedOutput()
	if err != nil {
		log.Info(err)
		outStr := strings.Trim(string(str), "\n")
		return nil, errors.New(outStr)
	}
	return str, err
	//return exec.Command("bash", "-c", command).CombinedOutput()
}

func IsValidIptablesTableName(tableName string) bool {
	if tableName == "filter" || tableName == "nat" || tableName == "mangle" || tableName == "raw" || tableName == "security" {
		return true
	} else {
		return false
	}
}

func ReadAllFileContent(filePath string) ([]byte, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func IsStrInStrLst(value string, arr []string) bool {
	for _, v := range arr {
		if v == value {
			return true
		}
	}
	return false
}

func WriteFileContent(filePath string, data []byte) error {
	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	// Clear file content
	err = file.Truncate(0)
	if err != nil {
		return err
	}

	_, err = file.Write(data)
	if err != nil {
		return err
	}

	return nil
}

func IsFileExists(filePath string) bool {
	if _, err := os.Stat(filePath); err == nil {
		return true
	}
	return false
}

func StringToMap(s string) (map[string]interface{}, error) {
	// Done
	// Convert string of json to map
	jsonMap := make(map[string]interface{})
	err := json.Unmarshal([]byte(s), &jsonMap)
	if err != nil {
		return nil, err
	}
	return jsonMap, nil
}

func IsIntInIntList(port int, portList []int) bool {
	for _, p := range portList {
		if p == port {
			return true
		}
	}
	return false
}

func RemoveStrInStrWithComma(str, strWithComma string) string {
	// Done
	strWithCommaLst := strings.Split(strWithComma, ",")
	for i, s := range strWithCommaLst {
		if s == str {
			strWithCommaLst[i] = ""
		}
	}
	strResult := strings.Join(strWithCommaLst, ",")
	strResult = strings.Replace(strResult, ",,", ",", -1)
	strResult = strings.Trim(strResult, ",")
	return strResult
}

func ReformatPortInProxyMap(portList string) (string, error) {
	portLst := strings.Split(portList, ",")

	portListTmp := make([]string, len(portLst))
	for i, port := range portLst {
		portListTmp[i] = strings.TrimSpace(port)
	}

	// Sort port list
	sort.Strings(portListTmp)

	log.Println("portLst: ", portListTmp)

	return strings.Join(portListTmp, ","), nil
}

func ReadYamlFile(filePath string) (map[string]interface{}, error) {
	data, err := ReadAllFileContent(filePath)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	err = yaml.Unmarshal(data, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func WriteYamlFile(filePath string, data map[string]interface{}) error {
	yamlData, err := yaml.Marshal(data)
	if err != nil {
		return err
	}

	err = WriteFileContent(filePath, yamlData)
	if err != nil {
		return err
	}
	return nil
}

func ConvertSliceInterfaceToSliceString(sliceInterface []interface{}) []string {
	var result []string
	for _, s := range sliceInterface {
		result = append(result, s.(string))
	}
	return result
}

func EncodeBase64(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

func DecodeBase64(s string) (string, error) {
	decoded, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}

func InterfaceToIntSlice(iface []interface{}) []int {
	var result []int
	for _, i := range iface {
		result = append(result, int(i.(float64)))
	}
	return result
}
