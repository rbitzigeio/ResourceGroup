package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	arg "github.com/Azure/azure-sdk-for-go/services/resourcegraph/mgmt/2021-03-01/resourcegraph"
	"github.com/Azure/go-autorest/autorest/azure/auth"
)

// Definition of subscriptions
// Name : String
// Id   : String
// VM   : Map of VirtualMachines (key is machine name)
type Subscription struct {
	Name string
	Id   string
	LS   string
	VM   map[string]VirtualMachine
	RG   map[string]ResourceGroup
}

// Definition of VirtualMachine
// Name : String
// Id   : String
// Size : String
type VirtualMachine struct {
	Name string
	Id   string
	Size string
}

// Definition of Resource Group
type ResourceGroup struct {
	Name string
	LS   string
}

func main() {
	fmt.Println("LS - By Subscription and Resource Groups")

	// get Subscription
	mapOfSubscriptions := getSubscriptions()

	f, err := os.Create("statisticLS.csv")
	check(err)
	defer f.Close()
	wr := bufio.NewWriter(f)
	nWrite, err := wr.WriteString("Subscription-Name, ID, LS, ResourceGroup, LS\n")
	check(err)
	fmt.Printf("wrote %d bytes\n", nWrite)

	// Iterate Subscriptions
	for s, w := range mapOfSubscriptions {
		ls := "?"
		if w.LS != "" {
			ls = w.LS
		}
		//if s == "cn-cccTest-01" || s == "mp-npi-AKOM" {
		//if s == "mp-npi-AKOM" {
		// Iterate ResourceGroups
		mapOfRG := getResourceGroups(w.Id)
		for s1, w1 := range mapOfRG {
			fmt.Println(s + ", " + w.Id + ", " + ls + ", " + s1 + ", " + w1.LS)
			_, err := wr.WriteString(s + ", " + w.Id + ", " + ls + ", " + s1 + ", " + w1.LS + "\n")
			check(err)
		}
		//}
	}
	wr.Flush()
}

// Request to receive all subscriptions which are visible for user
func getSubscriptions() map[string]Subscription {
	fmt.Println("- getSubscriptions")
	query := "resourcecontainers | where type == 'microsoft.resources/subscriptions' | project name, id, LS=tostring(tags.Leistungsschein)"
	//query := "resources | where type == 'microsoft.resources/subscriptions' | project name, id, LS=tostring(tags.Leistungsschein)"
	result := executeQuery(query)
	mapOfSubscriptions := extractSubscriptions(result)

	return mapOfSubscriptions
}

func getResourceGroups(subId string) map[string]ResourceGroup {
	fmt.Println("- getLSByResourceGroups")
	//id := strings.Replace(subId, "/subscriptions/", "", 1)
	query := "resourcecontainers | where type == 'microsoft.resources/subscriptions/resourcegroups' | project name, tags.Leistungsschein, subId=subscriptionId | where subId == '" + subId + "'"
	result := executeQuery(query)
	//fmt.Println(result)
	splits := splitResults(result)
	ls := ""
	name := ""
	mapOfRG := make(map[string]ResourceGroup)
	bName := false
	bLS := false
	for _, info := range splits {
		//fmt.Println(" - " + info)
		sName := strings.Split(info, "name:")
		if len(sName) == 2 {
			name = strings.Replace(sName[1], "]", "", 2)
			bName = true
		}
		saLS := strings.Split(info, "tags_Leistungsschein:")
		if len(saLS) == 2 {
			sls := strings.Replace(saLS[1], "]", "", 2)
			if ils, err := strconv.Atoi(sls); err == nil {
				ls = strconv.Itoa(ils)
			} else {
				ls = ""
			}
			bLS = true
		}
		if bName && bLS {
			var rg ResourceGroup
			rg.Name = name
			rg.LS = ls
			mapOfRG[name] = rg
			name = ""
			ls = ""
			bName = false
			bLS = false
		}
	}
	return mapOfRG
}

// Query to Azure
func executeQuery(query string) string {
	fmt.Println("  - execute query : " + query)
	// Create and authorize a ResourceGraph client
	argClient := arg.New()
	authorizer, err := auth.NewAuthorizerFromCLI()
	if err == nil {
		argClient.Authorizer = authorizer
	} else {
		fmt.Println(err.Error())
	}

	// Set options
	RequestOptions := arg.QueryRequestOptions{
		ResultFormat: "objectArray",
	}

	// Create the query request
	Request := arg.QueryRequest{
		Query:   &query,
		Options: &RequestOptions,
	}

	// Run the query and get the results
	var results, queryErr = argClient.Resources(context.Background(), Request)

	if queryErr == nil {
		//fmt.Println("Success")
		//fmt.Printf("Resources found: " + strconv.FormatInt(*results.TotalRecords, 10) + "\n")
		result := fmt.Sprint(results.Data)

		return result
	} else {
		fmt.Println(queryErr.Error())
	}
	return ""
}

// Extract subscriptions from response
// Identifier are:
// - name:
// - id:
func extractSubscriptions(result string) map[string]Subscription {
	fmt.Println("  - extract subscritions")
	splits := splitResults(result)

	name := ""
	bName := false
	var sName []string
	bSid := false
	sid := ""
	var sSid []string
	bLS := false
	sLS := ""
	var saLS []string
	i := 0
	//var mapOfSubs map[string]Subscription
	mapOfSubs := make(map[string]Subscription)
	for _, info := range splits {
		sName = strings.Split(info, "name:")
		if len(sName) == 2 {
			name = strings.Replace(sName[1], "]", "", 2)
			bName = true
		}
		sSid = strings.Split(info, "id:")
		if len(sSid) == 2 {
			sid = strings.Replace(sSid[1], "]", "", 2)
			sid = strings.Replace(sid, "/subscriptions/", "", 2)
			bSid = true
		}
		saLS = strings.Split(info, "LS:")
		if len(saLS) == 2 {
			sLS = strings.Replace(saLS[1], "]", "", 2)
			bLS = true
		}
		if bName && bSid && bLS {
			i = i + 1
			var sub Subscription
			sub.Name = name
			sub.Id = sid
			sub.LS = sLS
			mapOfSubs[name] = sub
			bName = false
			bSid = false
		}
	}

	return mapOfSubs
}

// Split response by seperator " " (blanc)
func splitResults(result string) []string {
	out, _ := json.Marshal(result)
	s := ""
	for _, asciiNum := range out {
		c := string(asciiNum)
		s = s + c
	}
	splits := strings.Split(s, " ")
	return splits
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}
