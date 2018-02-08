package configserver

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	str "strings"
	"time"

	"github.com/hokaccha/go-prettyjson"
)

type configServerJSON struct {
	Name            string   `json:"name"`
	Profiles        []string `json:"profiles"`
	PropertySources []struct {
		Name   string                 `json:"name"`
		Source map[string]interface{} `json:"source"`
	}
}

/*
* ThePropertiesMap is the global properties map that will be populated by
* values retrieved from the Config Server
 */
var ApplicationPropertiesMap = make(map[string]interface{})

var server = "localhost"
var port = "8888"

var configServerURL = "http://localhost:8888"

func getURL(application string, env string) string {
	return "http://" + server + ":" + port + "/" + application + "/" + env
}

/*
* This will take the Response and unmarshall into structure
* that can be utilized later
 */
func unmarshallMessage(response *http.Response) configServerJSON {

	body, readErr := ioutil.ReadAll(response.Body)
	if readErr != nil {
		log.Fatal(readErr)
	}
	var msg configServerJSON

	jsonErr := json.Unmarshal(body, &msg)
	if jsonErr != nil {
		log.Fatal("error unmarshalling the message body ... ", jsonErr)
	}
	return msg
}

var basicConfig = -1
var basicConfigEnv = -1
var appConfig = -1

/*
this will take the struct that has been marshalled and get us the map we are looking for
*/
func buildApplicationProperties(propStruct configServerJSON) {

	for i := 0; i < len(propStruct.PropertySources); i++ {
		if str.Compare(propStruct.PropertySources[i].Name, "file:config/application.yml") == 0 {
			if basicConfig == -1 {
				basicConfig = i
			}
		} else if str.Contains(propStruct.PropertySources[i].Name, "file:config/application.yml#"+ApplicationPropertiesMap["app_env"].(string)) {
			if basicConfigEnv == -1 {
				basicConfigEnv = i
			}
		} else if str.Contains(propStruct.PropertySources[i].Name, "file:config/"+ApplicationPropertiesMap["app_name"].(string)+".yml") {
			if appConfig == -1 {
				appConfig = i
			}
		}
	}
	fmt.Printf("Found baseConfig at index %d \n", basicConfig)
	fmt.Printf("Found baseConfigEnv at index %d \n", basicConfigEnv)
	fmt.Printf("Found appConfig at index %d \n", appConfig)

	//need to process in order so that properties do not get overwritten
	buildProperties(propStruct, basicConfig)
	buildProperties(propStruct, basicConfigEnv)
	if appConfig != -1 {
		buildProperties(propStruct, appConfig)
	}

	fmt.Printf("There are %d entries in ApplicationPropertiesMap \n", len(ApplicationPropertiesMap))

}

/*
This takes a specific index and then pulls from List of sources to get the
interface of values wanted.  This is because config-server returns to copies of the configs.
*/
func buildProperties(configs configServerJSON, idx int) {

	for k, v := range configs.PropertySources[idx].Source {
		ApplicationPropertiesMap[k] = v
	}
}

//Variable right now to use to print out more information. Must recompile library
var debug = 1

/*
set Debug for extra printing
*/
func SetDebug(value int) {
	debug = value
}

/*
 * GetProperties
 * get properties from Config Server
 */
func GetProperties(application string, environment string) {
	ApplicationPropertiesMap["app_name"] = application
	ApplicationPropertiesMap["app_env"] = environment
	fmt.Printf("Configuring for Application: %s \n", ApplicationPropertiesMap["app_name"])
	fmt.Printf("Configuring for Environment: %s \n", ApplicationPropertiesMap["app_env"])

	req, err := http.NewRequest(http.MethodGet, getURL(application, environment), nil)
	if err != nil {
		log.Fatal("error make new Request", err)
	}

	spaceClient := http.Client{
		Timeout: time.Second * 2, // Maximum of 2 secs
	}

	res, geErr := spaceClient.Do(req)
	if geErr != nil {
		log.Fatal("error making call", geErr)
	}

	msg := unmarshallMessage(res)

	if debug > 0 {
		s, _ := prettyjson.Marshal(msg)
		fmt.Println(string(s))
	}
	buildApplicationProperties(msg)

	if debug > 0 {
		for key, value := range ApplicationPropertiesMap {
			fmt.Println("Key:", key, " Value:", value)
		}
	}
}
