package helpers

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
	"gopkg.in/yaml.v3"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
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

func FindSubnetMask(ip1, ip2 net.IP) net.IPMask {
	ip1 = ip1.To4()
	ip2 = ip2.To4()
	// for each bit in the IP address
	for i := 0; i < len(ip1)*8; i++ {
		// if ip1 and ip2 are different at this bit
		if (ip1[i/8]>>uint(7-i%8))&1 != (ip2[i/8]>>uint(7-i%8))&1 {
			// return the mask
			return net.CIDRMask(i, len(ip1)*8)
		}
	}
	// otherwise return a full mask
	return net.CIDRMask(len(ip1)*8, len(ip1)*8)
}

func IsIpInNetworkArray(ip net.IP, networks []string) bool {
	for _, network := range networks {
		_, subnet, _ := net.ParseCIDR(network)
		if subnet.Contains(ip) {
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

func GetPublicInterface() (string, error) {
	// DONE
	command := "ip route show default | gawk '/^default/ {print $5}' | cut -d'.' -f1"
	out, err := ExcuteCommand(command)
	outSplit := strings.Split(string(out), "\n")
	result := strings.TrimSpace(outSplit[0])
	if len(result) == 0 {
		return "", errors.New("cannot get public interface")
	}
	return result, err
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

func ParsePortListFromProxyMap(proxyMapPortLst string, excludePortLst []int) ([]int, error) {
	// DONE
	// Example: 20,6660-6669,7777 to 20, 6660, 6661, 6662, 6663, 6664, 6665, 6666, 6667, 6668, 6669, 7777
	if len(proxyMapPortLst) == 0 {
		return make([]int, 0), nil
	}

	portLst := strings.Split(proxyMapPortLst, ",")

	// Trimspace for each port
	for i, port := range portLst {
		portLst[i] = strings.TrimSpace(port)
	}

	var result []int
	for _, port := range portLst {
		tmp, err := ParsePortOrPortRangeToPortList(port)
		if err != nil {
			return nil, err
		}
		result = append(result, tmp...)

	}

	// Exclude port 22, 26, 27, 65432 and check valid port
	if result[0] <= 0 || result[len(result)-1] > 65535 {
		return nil, errors.New("port out of range")
	}
	for _, port := range result {
		for _, excludePort := range excludePortLst {
			if port == excludePort {
				return nil, errors.New("port is not allowed")
			}
		}
	}

	return result, nil
}

func ParsePortOrPortRangeToPortList(portRange string) ([]int, error) {
	var result []int
	if !regexp.MustCompile(`^[0-9\-]+$`).MatchString(portRange) {
		return nil, fmt.Errorf("invalid port format")
	}
	if strings.Contains(portRange, "-") {
		portRangeLst := strings.Split(portRange, "-")
		if len(portRangeLst) != 2 {
			return nil, fmt.Errorf("invalid port range")
		}
		startPort, _ := strconv.Atoi(portRangeLst[0])
		endPort, _ := strconv.Atoi(portRangeLst[1])

		for i := startPort; i <= endPort; i++ {
			if !IsIntInIntList(i, result) {
				result = append(result, i)
			} else {
				return nil, fmt.Errorf("duplicate port")
			}
		}
	} else {
		intPort, _ := strconv.Atoi(portRange)
		if !IsIntInIntList(intPort, result) {
			result = append(result, intPort)
		} else {
			return nil, fmt.Errorf("duplicate port")
		}
	}

	return result, nil
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

func CheckValidIPv4OrNetwork(ip string) bool {
	// DONE
	if !strings.Contains(ip, "/") {
		ip = ip + "/32"
	} else {
		prefix := strings.Split(ip, "/")[1]
		intPrefix, _ := strconv.Atoi(prefix)
		if intPrefix < 16 || intPrefix > 32 {
			return false
		}
	}
	_, _, err := net.ParseCIDR(ip)
	return err == nil
}

func MD5(s string) string {
	// DONE
	// Get MD5 hash of string
	return fmt.Sprintf("%x", md5.Sum([]byte(s)))
}

//func ContainInTwoRangePort(r1, r2 string, excludePortLst []int) bool {
//	// false if r1 not contain r2 and r2 not contain r1
//	r1Lst, err := ParsePortListFromProxyMap(r1, excludePortLst)
//	if err != nil {
//		return true
//	}
//	r2Lst, err := ParsePortListFromProxyMap(r2, excludePortLst)
//	if err != nil {
//		return true
//	}
//
//	//if len(r1Lst) != len(r2Lst) {
//	//	return false
//	//}
//	//result := true
//
//	//for i := range r1Lst {
//	//	if r1Lst[i] != r2Lst[i] {
//	//		result = false
//	//		break
//	//	}
//	//}
//
//	// NEW
//	result := false
//	for _, port1 := range r1Lst {
//		for _, port2 := range r2Lst {
//			if port1 == port2 {
//				result = true
//				return result
//			}
//		}
//	}
//
//	return result
//}

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

//func ReformatPortInProxyMap(portList string) (string, error) {
//	// Port format: 20,6660-6669,7777
//	portList = strings.TrimSpace(portList)
//	portLstComponents := strings.Split(portList, ",")
//	for i, portLstComponent := range portLstComponents {
//		portLstComponents[i] = strings.TrimSpace(portLstComponent)
//		// Port only contain number and -
//		if !regexp.MustCompile(`^[0-9\-]+$`).MatchString(portLstComponents[i]) {
//			return "", fmt.Errorf("invalid port format")
//		}
//	}
//	sort.Strings(portLstComponents)
//	// Remove duplicate port
//	portLstInt, err := ParsePortListFromProxyMap(strings.Join(portLstComponents, ","))
//	if err != nil {
//		return "", err
//	}
//	// Reformat port list
//	var result []string
//	buffer := ""
//	endBuffer := ""
//	for i := range portLstInt {
//		if i == 0 {
//			buffer = strconv.Itoa(portLstInt[i])
//		} else {
//			if portLstInt[i]-portLstInt[i-1] == 1 {
//				endBuffer = strconv.Itoa(portLstInt[i])
//				continue
//			} else {
//				if len(endBuffer) > 0 {
//					buffer = buffer + "-" + endBuffer
//					endBuffer = ""
//				}
//				result = append(result, buffer)
//				buffer = strconv.Itoa(portLstInt[i])
//			}
//		}
//	}
//	if len(endBuffer) > 0 {
//		buffer = buffer + "-" + endBuffer
//	}
//	result = append(result, buffer)
//
//	return strings.Join(result, ","), nil
//}

func CheckKnownAppPortLst(portLst string, excludePortLst []int) (string, error) {
	// Port list form 20,6660-6669,7777
	if len(portLst) == 0 {
		return "", nil
	}
	resultPortLst := ""
	portLstSplitRaw := strings.Split(portLst, ",")
	for _, portLstSplit := range portLstSplitRaw {
		portLstSplit = strings.TrimSpace(portLstSplit)
		tmp, err := ParsePortOrPortRangeToPortList(portLstSplit)
		if err != nil {
			return "", err
		}
		if tmp[0] <= 0 || tmp[len(tmp)-1] > 65535 {
			return "", errors.New("port out of range")
		}
		for _, port := range tmp {
			if IsIntInIntList(port, excludePortLst) {
				return "", errors.New("port is not allowed")
			}
		}
		resultPortLst = resultPortLst + portLstSplit + ","

	}

	resultPortLst = strings.Trim(resultPortLst, ",")

	return resultPortLst, nil
}

func ReadYamlFile(filePath string) (map[string]interface{}, error) {
	// Done
	// Read yaml file
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

func GetCurrentGateway() (string, error) {
	// Done
	command := "ip route show default | gawk '/^default/ {print $3}'"
	out, err := ExcuteCommand(command)
	if err != nil {
		return "", err
	}

	result := strings.TrimSpace(string(out))
	result = strings.Trim(result, "\n")

	return result, nil
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

func GetAllIPv4InHost() ([]string, error) {
	result := make([]string, 0)

	command := "ip -4 addr show | gawk '/inet/ {print $2}' | cut -d'/' -f1"
	out, err := ExcuteCommand(command)
	if err != nil {
		return nil, err
	}

	ips := strings.Split(string(out), "\n")
	for _, ip := range ips {
		ip = strings.TrimSpace(ip)
		if len(ip) == 0 {
			continue
		}
		if strings.HasPrefix(ip, "10.") || strings.HasPrefix(ip, "172.") || strings.HasPrefix(ip, "192.168") || strings.HasPrefix(ip, "127.0") {
			continue
		}
		result = append(result, ip)
	}

	return result, nil
}

//// ConvertCIDRToNetworkAddress converts an IP address in CIDR notation to its network address form.
//func ConvertCIDRToNetworkAddress(cidr string) (string, error) {
//	// Parse the CIDR notation to get the IP and mask
//	ip, ipNet, err := net.ParseCIDR(cidr)
//	if err != nil {
//		return "", err
//	}
//
//	// Get the network address as a string
//	networkAddress := ip.Mask(ipNet.Mask)
//
//	return networkAddress.String(), nil
//}

func DownloadFile(url, filePath string) error {
	// Done
	command := "curl -L -o " + filePath + " " + url
	_, err := ExcuteCommand(command)
	if err != nil {
		return err
	}
	return nil
}

func IsDedicatedFirewall() bool {
	// Done
	command := "lspci | grep -i vmware"
	out, _ := ExcuteCommand(command)

	if len(out) > 5 {
		// VMWare
		return false
	}

	return true
}

func SortStringArray(arr []string) []string {
	sort.Strings(arr)
	return arr
}

func IsStringArrayTheSame(arr1, arr2 []string) bool {
	if len(arr1) != len(arr2) {
		return false
	}

	// Sort the arrays
	arr1 = SortStringArray(arr1)
	arr2 = SortStringArray(arr2)

	for i := range arr1 {
		if arr1[i] != arr2[i] {
			return false
		}
	}
	return true
}

// containsNetwork checks if net1 contains net2
func ContainsNetwork(net1, net2 string) (bool, error) {
	_, ipnet1, err := net.ParseCIDR(net1)
	if err != nil {
		return false, err
	}

	_, ipnet2, err := net.ParseCIDR(net2)
	if err != nil {
		return false, err
	}

	// Check if the first IP of net2 is within net1
	if ipnet1.Contains(ipnet2.IP) {
		// Calculate the last IP of net2
		lastIP := lastAddress(ipnet2)
		// Check if the last IP of net2 is also within net1
		return ipnet1.Contains(lastIP), nil
	}

	return false, nil
}

// lastAddress calculates the last IP address in the given CIDR range
func lastAddress(cidr *net.IPNet) net.IP {
	var lastIP net.IP
	for i := range cidr.IP {
		lastIP = append(lastIP, cidr.IP[i]|^cidr.Mask[i])
	}
	return lastIP
}

func IsMountLoop99() (bool, error) {
	// Done
	command := "mount | grep loop99"
	out, err := ExcuteCommand(command)
	if err != nil {
		return false, err
	}
	if len(out) > 5 {
		return true, nil
	}
	return false, nil
}

func InterfaceToIntSlice(iface []interface{}) []int {
	var result []int
	for _, i := range iface {
		result = append(result, int(i.(float64)))
	}
	return result
}

func RequestToAPI(url, method string, reqHeader, reqBody map[string]string, timeOutSecond int) ([]byte, error) {
	timeout := time.Duration(timeOutSecond) * time.Second

	// Convert reqBody to io.Reader
	payload := io.Reader(nil)
	var err error

	if reqBody != nil {
		payload, err = MapToJSONReader(reqBody)
		if err != nil {
			return nil, err
		}
	}

	client := &http.Client{
		Timeout: timeout,
	}
	req, err := http.NewRequest(method, url, payload)

	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	// Add custom header
	if reqHeader != nil {
		for key, value := range reqHeader {
			if key == "Host" || key == "host" {
				req.Host = value
			} else {
				req.Header.Set(key, value)
			}
		}
	} else {
		req.Header.Set("Content-Type", "application/json")
	}

	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(res.Body)

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	return body, nil
}

func MapToJSONReader(m map[string]string) (io.Reader, error) {
	// Convert the map to a JSON byte slice
	jsonData, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("error marshalling map to JSON: %v", err)
	}

	// Create an io.Reader from the JSON byte slice
	return bytes.NewReader(jsonData), nil
}

func ConvertBytesToMap(data []byte) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	err := json.Unmarshal(data, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func GenerateLocalIPFirewallIPBackend(ipfw, ipbe string) (ipLocalFw string, ipLocalBe string) {
	ipfwSplit := strings.Split(ipfw, ".")
	ipbeSplit := strings.Split(ipbe, ".")

	ipLocalFw = "10." + ipfwSplit[2] + "." + ipfwSplit[3] + "." + ipfwSplit[3]
	ipLocalBe = "10." + ipfwSplit[2] + "." + ipfwSplit[3] + "."
	intOcta4Be, _ := strconv.Atoi(ipbeSplit[3])
	if strings.Compare(ipfwSplit[3], ipbeSplit[3]) == 0 || intOcta4Be == 0 || intOcta4Be > 254 {
		// Change IPLocal FW
		// Convert ipbeSplit[3] to int and +1
		if intOcta4Be < 254 {
			intOcta4Be++
		} else {
			intOcta4Be--
		}
	}
	ipLocalBe += strconv.Itoa(intOcta4Be)
	return ipLocalFw, ipLocalBe
}

func GetDistinctStringArray(arr []string) []string {
	// Remove duplicate string in array
	result := make([]string, 0)
	for _, s := range arr {
		if !IsStrInStrLst(s, result) {
			result = append(result, s)
		}
	}
	return result
}

func ConcatMaps(map1, map2 map[string]interface{}) map[string]interface{} {
	// Create a new map to hold the result
	result := make(map[string]interface{})

	// Add all key-value pairs from map1 to result
	if map1 != nil {
		for k, v := range map1 {
			result[k] = v
		}
	}

	// Add all key-value pairs from map2 to result
	if map2 != nil {
		for k, v := range map2 {
			result[k] = v
		}
	}

	return result
}

func GetAllIfaceName() []string {
	// Done
	command := "ip -o link show | awk -F': ' '{print $2}'"
	out, _ := ExcuteCommand(command)
	if len(out) == 0 {
		return nil
	}
	ifaceNameLst := strings.Split(string(out), "\n")
	// Remove lo and docker0
	for i, ifaceName := range ifaceNameLst {
		if ifaceName == "lo" || ifaceName == "docker0" || ifaceName == "" {
			ifaceNameLst = append(ifaceNameLst[:i], ifaceNameLst[i+1:]...)
		}
	}
	ifaceNameLst = ifaceNameLst[:len(ifaceNameLst)-1]
	return ifaceNameLst
}

func GetMainIPAddress(ifaceName string) ([]string, error) {
	// Done
	command := "ip -o -4 addr show | grep " + ifaceName + " | awk '{print $4}'"
	out, err := ExcuteCommand(command)
	if err != nil {
		return nil, err
	}
	//result := strings.Split(strings.TrimSpace(string(out)), "\n")[0]
	result := make([]string, 0)
	for _, ip := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if strings.HasPrefix(ip, "10.") || strings.HasPrefix(ip, "192.168") || strings.HasPrefix(ip, "172.17") || strings.HasPrefix(ip, "127.0") {
			continue
		}
		result = append(result, strings.Split(ip, "/")[0])
	}
	return result, nil
}

func GetHostName() string {
	// Done
	command := "hostname"
	out, _ := ExcuteCommand(command)
	return strings.TrimSpace(string(out))
}

func CheckRegexPattern(pattern, matchStr string) bool {
	// Done
	rg, err := regexp.Compile(pattern)
	if err != nil {
		return false
	}
	return rg.Match([]byte(matchStr))
}

func CheckPortRange(port string) bool {
	// Done
	portSplit := strings.Split(port, "-")
	startPort, err := strconv.Atoi(portSplit[0])
	if err != nil {
		return false
	}
	endPort, err := strconv.Atoi(portSplit[1])
	if err != nil {
		return false
	}

	if startPort < 0 || startPort > 65535 || endPort < 0 || endPort > 65535 {
		return false
	}

	if startPort > endPort {
		return false
	}

	return true
}

func ReplacePatternInFile(filePath, pattern, replace string) error {
	// Done
	command := "sed -i 's|" + pattern + "|" + replace + "|g' " + filePath
	_, err := ExcuteCommand(command)
	if err != nil {
		return err
	}
	return nil
}

func RandomString(n int) string {
	// Done
	var letter = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, n)
	for i := range b {
		b[i] = letter[rand.Intn(len(letter))]
	}
	return string(b)
}

func RandomEmail() string {
	// Random 5 character
	randStr := RandomString(5)
	emailServerList := []string{
		"gmail.com",
		"yahoo.com",
		"outlook.com",
		"hotmail.com",
		"protonmail.com",
	}
	return randStr + "@" + emailServerList[rand.Intn(len(emailServerList))]
}

func GetPrefixFromIP(ip string) string {
	// Done
	if strings.Contains(ip, "/") {
		ip = strings.Split(ip, "/")[0]
	}

	command := "ip a | grep " + ip + "| awk '{print $2}' | cut -d '/' -f2"
	out, _ := ExcuteCommand(command)
	return strings.TrimSpace(string(out))
}

func RunNetworkTuning() error {
	command := "/bin/bash /gen7-api/scripts/network_tuning.sh"
	_, _ = ExcuteCommand(command)
	return nil
}

func GetVlanFromIP(ip string) string {
	// Done
	if strings.Contains(ip, "/") {
		ip = strings.Split(ip, "/")[0]
	}

	command := "ip a | grep " + ip + " | awk '{print $7}' | cut -d '.' -f2"
	out, _ := ExcuteCommand(command)
	outStr := strings.Split(strings.TrimSpace(string(out)), "\n")[0]

	if strings.Contains(outStr, "secondary") {
		command = "ip a | grep " + ip + " | awk '{print $8}' | cut -d '.' -f2"
		out, _ = ExcuteCommand(command)
		return strings.Split(strings.TrimSpace(string(out)), "\n")[0]
	}
	return outStr
}

func GetHostname() string {
	// Done
	command := "hostname"
	out, _ := ExcuteCommand(command)
	result := strings.TrimSpace(string(out))

	pattern := "fw[4,5]{1}gen7-s[0-9]*"
	if !CheckRegexPattern(pattern, result) {
		return result
	}

	return result + ".vietnix.vn"
}

func InitFiles(dataFiles []string) {
	for _, file := range dataFiles {
		if !IsFileExists(file) {
			_, _ = os.Create(file)
		}
	}
}

func IsDirectoryExists(dirPath string) bool {
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		return false
	}
	return true
}

func GenerateGatewayIP(ipStr, prefixStr string) (string, error) {
	prefix, err := strconv.Atoi(prefixStr)
	if err != nil {
		return "", err
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		return "", fmt.Errorf("invalid IP address: %s", ipStr)
	}

	// Convert IP to 4-byte array
	ip = ip.To4()
	if ip == nil {
		return "", fmt.Errorf("invalid IPv4 address: %s", ipStr)
	}

	// Create subnet mask from prefix
	mask := net.CIDRMask(prefix, 32)

	// Calculate network address by ANDing IP with mask
	network := ip.Mask(mask)

	// Gateway IP is network address + 1
	gateway := make(net.IP, len(network))
	copy(gateway, network)
	gateway[3]++

	return gateway.String(), nil

}

func SplitAndRemoveExcludedPort(portRange string, excludePortList []string) ([]string, error) {
	// Valid input: 80-83
	regexPattern := `^([0-9]+)-([0-9]+)$`
	if !CheckRegexPattern(regexPattern, portRange) {
		return nil, fmt.Errorf("invalid port range: %s", portRange)
	}

	portLst, err := ParsePortOrPortRangeToPortList(portRange)
	if err != nil {
		return nil, err
	}

	portLstNew := make([]int, 0)
	for _, port := range portLst {
		if !IsStrInStrLst(strconv.Itoa(port), excludePortList) {
			portLstNew = append(portLstNew, port)
		}
	}

	// Convert port list to string
	result := make([]string, 0)
	// Write go func to convert list int like 78,79,81,82 to 78-79,81-82
	for i := 0; i < len(portLstNew); i++ {
		startPort := portLstNew[i]
		endPort := startPort
		for j := i + 1; j < len(portLstNew); j++ {
			if portLstNew[j]-portLstNew[j-1] == 1 {
				endPort = portLstNew[j]
				i++
			} else {
				break
			}
		}
		if startPort != endPort {
			result = append(result, strconv.Itoa(startPort)+"-"+strconv.Itoa(endPort))
		} else {
			result = append(result, strconv.Itoa(startPort))
		}
	}

	return result, nil
}

func CheckBackendPort(ip, port string) error {
	// Check port is not in use
	command := "nc -zv " + ip + " " + port
	out, err := ExcuteCommand(command)
	if err != nil {
		return err
	}
	outStr := strings.TrimSpace(string(out))
	if strings.Contains(outStr, "succeeded") {
		return nil
	}
	return errors.New(outStr)
}

func GetStateUpInterface() ([]string, error) {
	// Done
	command := "ip a | grep 'state UP' | awk -F '[ .:]' '{print $3}' | uniq"
	out, err := ExcuteCommand(command)
	if err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, errors.New("no interface is up")
	}

	ifaceNameLst := strings.Split(string(out), "\n")
	ifaceNameLst = ifaceNameLst[:len(ifaceNameLst)-1]
	log.Info("GetStateUpInterface: ", ifaceNameLst)

	return ifaceNameLst, nil
}

func ReconfigFilebeat() {
	cmd := "mv /etc/filebeat/filebeat.yml /etc/filebeat/filebeat.yml.bak 2>/dev/null && cp -f /gen7-api/templates/filebeat.yml /etc/filebeat/filebeat.yml && service filebeat restart"
	_, _ = ExcuteCommand(cmd)
}

func ResyncTime() {
	cmd := "apt-get install ntpdate -y && ntpdate 103.184.124.254"
	_, _ = ExcuteCommand(cmd)
}

func RunIRQ() {
	cmd := "/bin/bash /gen7-api/scripts/irq.sh"
	_, _ = ExcuteCommand(cmd)
}

func ExcuteRemoteCommand(client *ssh.Client, command string) (string, error) {
	// Create a new session
	session, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer func(session *ssh.Session) {
		_ = session.Close()
	}(session)

	// Execute command
	output, err := session.CombinedOutput(command)
	if err != nil {
		return "", err
	}

	return string(output), nil
}

func GenerateExportedFileNameByIP(ip string) (string, error) {
	if len(ip) == 0 {
		ifaces, err := GetPublicInterface()
		if err != nil {
			return "", err
		}

		ipLst, err := GetMainIPAddress(ifaces)
		if err != nil {
			return "", err
		}

		ip = ipLst[0]
	}

	return strings.TrimSpace(strings.Split(ip, "/")[0]) + "_config.zip", nil
}

func LogActionsTime(filePath string) {
	// Define the file path

	// Get the current time
	currentTime := time.Now()

	// Format the time as hh:mm dd/mm/yyyy
	formattedTime := currentTime.Format("15:04 31/01/2006")

	// Open the file for writing (create if it doesn't exist)
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		log.Println("LogActionsTime: Error opening file:", err)
		return
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	// Write the formatted time to the file
	_, err = file.WriteString(formattedTime)
	if err != nil {
		log.Println("LogActionsTime: Error writing to file:", err)
		return
	}

	log.Println("LogActionsTime: Successfully wrote current time to the file")
	return
}

func CreateNewFile(filePath string) error {
	// Create a new file or truncate it if it already exists
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("error creating file: %v", err)
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	log.Printf("CreateNewFile: Successfully created or truncated file: %s\n", filePath)
	return nil
}
