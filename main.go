package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/rikatz/kubepug/pkg/utils"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	log "github.com/sirupsen/logrus"
)

var (
	version, swaggerDir, templateFile string
	force                             bool
	definitionsMap                    map[string]interface{}
)

// Represents an API taken from the Kubernetes schema
type kubeAPI struct {
	Description          string
	Group                string
	Kind                 string
	Version              string
	Name                 string
	Deprecated           bool
	DeprecatedProperties []kubeAPIProperty
}

// Represents an API property taken from the Kubernetes schema
type kubeAPIProperty struct {
	Name        string
	Description string
}

type kubernetesAPIs map[string]kubeAPI

func init() {
	flag.StringVar(&version, "version", "master", "Kubernetes version to check for deprecations")
	flag.StringVar(&swaggerDir, "path", "", "Path to read/download the Kubernetes API Swagger file (default same as 'version')")
	flag.StringVar(&templateFile, "template", "", "Path to the template to render")
	flag.BoolVar(&force, "force", false, "Whether to force download the Kubernetes API Swagger file")
	logLevel := flag.String("log-level", "info", "Log level. Should be: debug, info, warn, error")
	flag.Parse()

	if swaggerDir == "" {
		swaggerDir = version
	}

	switch *logLevel {
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "warn":
		log.SetLevel(log.WarnLevel)
	case "error":
		log.SetLevel(log.ErrorLevel)
	default:
		log.SetLevel(log.InfoLevel)
		log.Warnf("unknown log level %s. Setting log level to info", logLevel)
	}
}

func main() {
	k8sAPIs := kubernetesAPIs{}

	_ = os.Mkdir(swaggerDir, 0755)
	swagger, err := utils.DownloadSwaggerFile(version, swaggerDir, force)
	if err != nil {
		log.Fatalf("error downloading Swagger file: %v", err.Error())
	}

	if err := k8sAPIs.populateKubeAPIMap(swagger); err != nil {
		log.Fatalf("error populating API map from Swagger file: %v", err.Error())
	}

	rego, err := template.New(filepath.Base(templateFile)).Funcs(template.FuncMap{
		"extractDeprecation": func(s string) string {
			match := regexp.MustCompile("[D|d]eprecated in.*|DEPRECATED.*|Deprecated:.*").FindStringSubmatch(s)
			if len(match) == 1 {
				return match[0]
			}
			return s
		},
	}).ParseFiles(templateFile)

	if err != nil {
		log.Fatalf("error templating rego document: %v", err)
	}

	if err := rego.Execute(os.Stdout, k8sAPIs); err != nil {
		log.Fatalf("error templating deprecations: %v", err)
	}
}

func (kAPIs kubernetesAPIs) populateKubeAPIMap(swaggerfile string) (err error) {
	log.Debugf("Opening the swagger file for reading: %s", swaggerfile)
	jsonFile, err := os.Open(swaggerfile)
	if err != nil {
		return err
	}
	byteValue, _ := ioutil.ReadAll(jsonFile)

	err = jsonFile.Close()
	if err != nil {
		return err
	}
	err = json.Unmarshal(byteValue, &definitionsMap)
	if err != nil {
		return fmt.Errorf("error parsing the JSON, file might be invalid: %v", err)
	}
	definitions := definitionsMap["definitions"].(map[string]interface{})

	log.Debugf("Iterating through %d definitions", len(definitions))
	for k, value := range definitions {
		val := value.(map[string]interface{})
		log.Debugf("Getting API values from %s", k)
		if kubeapivalue, valid := getKubeAPIValues(k, val); valid {
			log.Debugf("Valid API object found for %s", k)
			var name string
			if kubeapivalue.Group != "" {
				name = fmt.Sprintf("%s/%s/%s", kubeapivalue.Group, kubeapivalue.Version, kubeapivalue.Kind)
			} else {
				name = fmt.Sprintf("%s/%s", kubeapivalue.Version, kubeapivalue.Kind)
			}
			log.Debugf("Adding %s to map. Deprecated: %t", name, kubeapivalue.Deprecated)
			kAPIs[name] = kubeapivalue
		}
	}
	return nil
}

func getGroupVersionKind(value map[string]interface{}) (group, version, kind string) {
	for k, v := range value {
		switch k {
		case "group":
			group = v.(string)
		case "version":
			version = v.(string)
		case "kind":
			kind = v.(string)
		}
	}
	return group, version, kind
}

func getKubeAPIValues(name string, value map[string]interface{}) (kubeAPI, bool) {
	var valid, deprecated bool
	var description, group, version, kind string
	var depProps []kubeAPIProperty

	// Does it look like a Spec?
	isSpec := strings.HasSuffix(name, "Spec")

	gvk, valid, err := unstructured.NestedSlice(value, "x-kubernetes-group-version-kind")
	if err != nil {
		return kubeAPI{}, false
	}

	if !isSpec && valid {
		gvkMap := gvk[0]
		group, version, kind = getGroupVersionKind(gvkMap.(map[string]interface{}))
	} else {
		group, version, kind, err = inferGVKFromSpecName(name)
		if err != nil {
			log.Warnf(err.Error())
			return kubeAPI{}, false
		}
	}

	description, found, err := unstructured.NestedString(value, "description")

	if !isSpec && (!found || err != nil || description == "") {
		log.Warnf("no description found for %s - ignoring...", name)
		return kubeAPI{}, false
	}

	if strings.Contains(strings.ToLower(description), "deprecated") {
		log.Debugf("%s description contains 'deprecated'", name)
		deprecated = true
	}

	properties, pfound, err := unstructured.NestedMap(value, "properties")
	if !pfound || err != nil {
		log.Debugf("no properties in %s", name)
	}

	if pfound {
		for k, v := range properties {
			desc, dfound, err := unstructured.NestedString(v.(map[string]interface{}), "description")
			if !dfound || err != nil || desc == "" {
				log.Debugf("property %s in %s has no description - ignoring...", k, name)

			} else if strings.Contains(strings.ToLower(desc), "deprecated") {
				log.Debugf("%s property in %s description contains 'deprecated'", k, name)
				depProps = append(depProps, kubeAPIProperty{Name: k, Description: removeNewlines(desc)})
			}
		}
	}

	if valid || pfound {
		return kubeAPI{
			Description:          removeNewlines(description),
			Group:                group,
			Kind:                 kind,
			Version:              version,
			Deprecated:           deprecated,
			DeprecatedProperties: depProps,
		}, true
	}

	return kubeAPI{}, false
}

func inferGVKFromSpecName(name string) (g, v, k string, err error) {
	if name[:11] != "io.k8s.api." {
		log.Debugf("%v is not part of io.k8s.api - ingoring...", name)
		return g, v, k, nil
	}

	parts := strings.Split(name[11:], ".")
	switch len(parts) {
	case 3:
		g = parts[0]
		v = parts[1]
		k = strings.TrimSuffix(parts[2], "Spec")
	case 2:
		v = parts[0]
		k = strings.TrimSuffix(parts[1], "Spec")
	default:
		return g, v, k, fmt.Errorf("cannot derive valid GVK from %s - ignoring...", name)

	}

	return g, v, k, nil
}

func removeNewlines(s string) string {
	re := regexp.MustCompile(`\r?\n`)
	return re.ReplaceAllString(s, " ")
}
