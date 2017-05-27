package main

import (
	"bufio"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/olekukonko/tablewriter"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

type xdebugLine struct {
	depth            int64
	functionNumber   int64
	entry            int64
	time             float64
	memoryUsage      int64
	name             string
	internalFunction int64
	include          string
	filename         string
	lineNumber       int64
	numOfParams      int64
}

type stackEntry struct {
	order        int
	calls        int
	funcName     string
	time         float64
	memory       int64
	nestedTime   float64
	nestedMemory int64
}

type functionEntry struct {
	name            string
	order           int
	calls           int
	timeInclusive   float64
	memoryInclusive int64
	timeChildren    float64
	memoryChildren  int64
	timeOwn         float64
	memoryOwn       int64
}

var functions map[string]stackEntry
var stackFunctions []string
var stack map[string]stackEntry
var sortKey string
var supportedKeys []string
var resultsLimit int
var csvFileName string
var useCSV bool

func main() {

	supportedKeys := []string{"calls", "flow", "time", "memory"}
	filenamePtr := flag.String("filename", "", "Path to xdebug log")
	sortKeyPtr := flag.String("sortKey", "flow", "Sort Key [ flow, calls, time, memory ] ")
	resultsLimitPtr := flag.Int("limit", 25, "Number of results returned")
	useCSVPtr := flag.Bool("useCSV", false, "Output to csv file")
	resultsLimit = *resultsLimitPtr
	useCSV = *useCSVPtr
	//sortKey = *sortKeyPtr

	flag.Parse()
	if len(*filenamePtr) == 0 {
		panic("Filename is invalid")
	}
	foundKey := false
	for _, supportedKey := range supportedKeys {
		if *sortKeyPtr == supportedKey {
			foundKey = true
		}
	}
	if foundKey == false {
		panic("Invalid Sort Key was used")
	}

	if fileInfo, err := os.Stat(*filenamePtr); os.IsNotExist(err) {
		panic("Filename does not exist")
	} else {
		csvFileName = fmt.Sprintf("%s.sorted_by_%s.csv", fileInfo.Name(), *sortKeyPtr)
	}

	f, err := os.Open(*filenamePtr)

	check(err)

	reader := bufio.NewReader(f)
	funcs := parse(reader, *sortKeyPtr)
	f.Close()
	sort.Sort(FunctionList(funcs))

	writeToStdout(funcs)
}
func writeToStdout(funcs []functionEntry) {

	table := tablewriter.NewWriter(os.Stdout)

	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetHeader([]string{"Name", "Calls", "Time Inclusive", "Memory", "Nested Time", "Nested Memory", "Order"})
	//check(err)
	//Convert to multi dimensional array
	var multidim [][]string
	var dim []string
	for _, v := range funcs {
		dim = []string{
			v.name,
			strconv.Itoa(v.calls),
			fmt.Sprintf("%f", v.timeInclusive),
			fmt.Sprintf("%d", v.memoryInclusive),
			fmt.Sprintf("%f", v.timeOwn),
			fmt.Sprintf("%d", v.memoryOwn),
			fmt.Sprintf("%d", v.order),
		}
		multidim = append(multidim, dim)
	}
	table.AppendBulk(multidim[:resultsLimit])
	table.Render()
	if useCSV {
		writeToCSV(multidim)
	}
}
func parse(reader *bufio.Reader, sortKey string) []functionEntry {
	lineCount := int(0)
	newlinebytes := byte('\n')
	stack = make(map[string]stackEntry)
	functions = make(map[string]stackEntry)

	stack["-1"] = stackEntry{0, 0, "", 0, 0, 0, 0}
	stack["0"] = stackEntry{0, 0, "", 0, 0, 0, 0}

	stackFunctions = []string{}

	tabRegexp := regexp.MustCompile("\t")
	for {
		line, err := reader.ReadString(newlinebytes)
		if err == io.EOF {
			break
		}
		check(err)
		lineCount++
		if lineCount < 4 {
			continue
		}

		parsedSlice := tabRegexp.Split(line, -1)
		//fmt.Println(line)

		if len(parsedSlice) < 5 {
			continue
		}

		xLine := xdebugLine{}
		for i, val := range parsedSlice {
			val = strings.Trim(val, "")
			re := regexp.MustCompile(`\r?\n`)
			val = re.ReplaceAllString(val, "")
			switch i {
			case 0:
				xLine.depth, _ = strconv.ParseInt(val, 0, 64)
			case 1:
				xLine.functionNumber, _ = strconv.ParseInt(val, 0, 64)
			case 2:
				xLine.entry, _ = strconv.ParseInt(val, 0, 64)
			case 3:
				xLine.time, _ = strconv.ParseFloat(val, 64)
			case 4:
				xLine.memoryUsage, _ = strconv.ParseInt(val, 10, 64)
			case 5:
				xLine.name = val
			case 6:
				xLine.internalFunction, _ = strconv.ParseInt(val, 10, 64)
			case 7:
				xLine.include = val
			case 8:
				xLine.filename = val
			case 9:
				xLine.lineNumber, _ = strconv.ParseInt(val, 10, 64)
			case 10:
				xLine.numOfParams, _ = strconv.ParseInt(val, 10, 64)
			}
		}
		depthKey := strconv.FormatInt(xLine.depth, 10)

		if xLine.entry == 0 {

			stackentry := stackEntry{
				funcName:     xLine.name,
				time:         xLine.time,
				memory:       xLine.memoryUsage,
				nestedMemory: 0,
				nestedTime:   0,
				order:        lineCount,
			}
			stack[depthKey] = stackentry
			stackFunctions = append(stackFunctions, xLine.name)
		} else if xLine.entry == 1 {
			//get Previous Line
			prevXLine := stack[depthKey]
			dTimeString := fmt.Sprintf("%f", xLine.time-prevXLine.time)
			dTime, _ := strconv.ParseFloat(dTimeString, 64)
			dMemory := xLine.memoryUsage - prevXLine.memory
			prevDepthKey := strconv.FormatInt(xLine.depth-1, 10)

			se2 := stack[prevDepthKey]
			se2.nestedTime += dTime
			se2.nestedMemory += dMemory
			stack[prevDepthKey] = se2

			stackFunctions = slicePop(stackFunctions)

			addToFunction(prevXLine.funcName,
				dTime,
				dMemory,
				prevXLine.nestedTime,
				prevXLine.nestedMemory,
				prevXLine.order)
		}
	}

	//sort by sortKey
	funcs := getFunctions()
	return funcs

}

func writeToCSV(data [][]string) {
	file, err := os.Create(csvFileName)
	check(err)
	defer file.Close()
	w := csv.NewWriter(file)
	for _, record := range data {
		if err := w.Write(record); err != nil {
			log.Fatalln("Error writing record to csv:", err)
		}
	}
	w.Flush()

	if err := w.Error(); err != nil {
		log.Fatal(err)
	}

}
func getFunctions() []functionEntry {
	var f []functionEntry
	for funcName, arr := range functions {
		fe := functionEntry{
			name:            funcName,
			calls:           arr.calls,
			timeInclusive:   arr.time,
			memoryInclusive: arr.memory,
			timeChildren:    arr.nestedTime,
			memoryChildren:  arr.nestedMemory,
			timeOwn:         arr.time - arr.nestedTime,
			memoryOwn:       arr.memory - arr.nestedMemory,
			order:           arr.order,
		}
		f = append(f, fe)
	}
	return f
}

//FunctionList is an array of function entries
type FunctionList []functionEntry

//Len gets function list length
func (f FunctionList) Len() int {
	return len(f)
}

// Swap just swaps elements of slice
func (f FunctionList) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}

func (f FunctionList) Less(i, j int) bool {
	var r bool
	switch sortKey {
	case "flow":
		r = f[i].order < f[j].order
	case "calls":
		r = f[i].calls > f[j].calls
	case "time", "time-inclusive":
		r = f[i].timeInclusive > f[j].timeInclusive
	case "memory-inclusive", "memory":
		r = f[i].memoryInclusive > f[j].memoryInclusive
	default:
		r = false
	}
	return r
}
func slicePop(slice []string) []string {
	sliceLength := len(slice)
	return slice[:sliceLength-1]
}

func addToFunction(functionName string, time float64, memory int64, nestedTime float64, nestedMemory int64, order int) {
	if _, prs := functions[functionName]; prs == false {
		functions[functionName] = stackEntry{}
	}
	elem := functions[functionName]
	elem.calls++
	found := false
	for _, v := range stackFunctions {
		if functionName == v {
			found = true
		}
	}
	if found == false {
		elem.time += time
		elem.memory += memory
		elem.nestedTime += nestedTime
		elem.nestedMemory += nestedMemory
		elem.order = order
	}
	functions[functionName] = elem
}
