
package main

import (
    "bufio"
    "os"
    "fmt"
    "flag"
    "context"
    "strings"
    "encoding/base64"
    "compress/gzip"
    "log"
    "bytes"
    "io"
	"io/ioutil"
	"encoding/json"
	"path/filepath"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/client-go/kubernetes"
    _ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
    "k8s.io/client-go/tools/clientcmd"
    helm "helm.sh/helm/v3/pkg/release" // "helm.sh/release.v1"
    yaml "github.com/ghodss/yaml"
    //v1 "k8s.io/api/core/v1"
)


var helmCharts = make(map[string]helm.Release)
//helmCharts = make(map[string]helm.Release)


func main() {
	reader := bufio.NewReader(os.Stdin)
	
	source_kubeconfig := flag.String("source_kubeconfig", "/home/ec2-user/.kube/gke", "a string")
	//namespaces := flag.String("namespaces", "", "a string")
	
	flag.Parse()
	fmt.Println("source_kubeconfig:", *source_kubeconfig)
	
	
	// Accept input from user
	if *source_kubeconfig == "" {
		fmt.Print("Please pass the location of source kubernetes cluster kubeconfig file: ")
		*source_kubeconfig, _ = reader.ReadString('\n')
		//source_kubeconfig = "/home/ec2-user/.kube/gke"
	}
	
    kubeconfig := flag.String("kubeconfig", *source_kubeconfig, "kubeconfig file")
	flag.Parse()
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		fmt.Printf("The kubeconfig cannot be loaded: %v\n", err)
		os.Exit(1)
	}
	clientset, err := kubernetes.NewForConfig(config)
	
	Generate_secret_config(clientset)
}




// Scan source kubernetes cluster and generate the secret objects
func Generate_secret_config(clientset *kubernetes.Clientset) {
		// Loop through all the namespaces and get the list of services
	
	secret, err := clientset.CoreV1().Secrets("default").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Printf("Could not read kubernetes SVC using cluster client: %v\n", err)
		os.Exit(1)
	}
    
    for _, item := range secret.Items {
    	fmt.Println(item.ObjectMeta.Name)	
    }	
    
    // Remove default secret from each namespace
    for i, item := range secret.Items {
    	if strings.Contains(item.ObjectMeta.Name, "default-token-") {
    		secret.Items = append(secret.Items[:i], secret.Items[i+1:]...)
    		break
    	}
    }
    
    fmt.Println()
    // append list of services in this namespace to glabal services list
	for _, item := range secret.Items {
    	fmt.Println(item.ObjectMeta.Name)	
    }
}


func main1() {
    reader := bufio.NewReader(os.Stdin)
	
	source_kubeconfig := flag.String("source_kubeconfig", "/home/ec2-user/.kube/gke", "a string")
	//namespaces := flag.String("namespaces", "", "a string")
	
	flag.Parse()
	fmt.Println("source_kubeconfig:", *source_kubeconfig)
	
	
	// Accept input from user
	if *source_kubeconfig == "" {
		fmt.Print("Please pass the location of source kubernetes cluster kubeconfig file: ")
		*source_kubeconfig, _ = reader.ReadString('\n')
		//source_kubeconfig = "/home/ec2-user/.kube/gke"
	}
	
    kubeconfig := flag.String("kubeconfig", *source_kubeconfig, "kubeconfig file")
	flag.Parse()
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		fmt.Printf("The kubeconfig cannot be loaded: %v\n", err)
		os.Exit(1)
	}
	clientset, err := kubernetes.NewForConfig(config)
	clientset = clientset
	
	labelSelector := fmt.Sprintf("owner=helm")
	listOptions := metav1.ListOptions{
	    LabelSelector: labelSelector,
	}
	
	secretList, err := clientset.CoreV1().Secrets("default").List(context.TODO(), listOptions )
	if err != nil {
			fmt.Printf("Could not read kubernetes Secrets using cluster client: %v\n", err)
			os.Exit(1)
	}
	
	for _, secret := range secretList.Items {
	    status_tmp := secret.ObjectMeta.Labels["status"]
	    status := strings.Split(status_tmp,":")
	    if status[0] == "deployed"{
	        base64Text := make([]byte, base64.StdEncoding.DecodedLen(len(secret.Data["release"])))
	        base64.StdEncoding.Decode(base64Text, []byte(secret.Data["release"]))
	        base64.StdEncoding.Decode(base64Text, base64Text)
	        
	        var secret_uncompressed bytes.Buffer
	        err = gunzipWrite(&secret_uncompressed, base64Text)
	        if err != nil {
		        log.Fatal(err)
	        }
	        secret_string := secret_uncompressed.String()
	        
            
            //convert the release string into helm release struct 
	        var secret_data helm.Release//map[string]interface{}
	        json.Unmarshal([]byte(secret_string), &secret_data)
	        
	        //Add the chart name to the helmCharts global variable 
	        helmCharts[secret_data.Name] = secret_data
	    	
	    	writeChartToFile(helmCharts)   
	    }
	       
	}
	
}

func writeChartToFile (charts map[string]helm.Release) {
	// Create the directory locally to store the helm charts
	path := "/home/ec2-user/HelmCharts"
	if _, err := os.Stat(path); os.IsNotExist(err) {
	    err = os.Mkdir(path, 0700)
	    if err != nil {
	    	fmt.Println("Error creating the path for helm charts")
	    }
	}
	
	
	// Get the ctarts from release struct
	for k,v := range charts {
		// Create subdirectory to store the charts for this release
		if _, err := os.Stat(path+"/"+v.Name); os.IsNotExist(err) {
		   err = os.Mkdir(path+"/"+v.Name, 0700)
		   if err != nil {
	    		fmt.Println("Error creating the path for helm release: ", v.Name)
	    	}
		}
		
		helm_templates := v.Chart.Templates
    	fmt.Println("Chart Name:", k)
	    //fmt.Println("template:", secret.Data["release"])
	    for _, element := range helm_templates {
	        fmt.Println("secrets:", element.Name)
	        //fmt.Println("template:", string(element.Data))
	        if _, err := os.Stat(filepath.Dir(path+"/"+v.Name+"/"+element.Name)); os.IsNotExist(err) {
	        	os.MkdirAll(filepath.Dir(path+"/"+v.Name+"/"+element.Name), 0700)
	        }
			err := ioutil.WriteFile(path+"/"+v.Name+"/"+element.Name, element.Data, 0644)
    		if err != nil {
    			panic(err)
    		}
	    }
	    
	    helm_files := v.Chart.Files
	    for _, element := range helm_files {
	        fmt.Println("Files Name:", element.Name)
	        //fmt.Println("template:", string(element.Data))
	        if _, err := os.Stat(filepath.Dir(path+"/"+v.Name+"/"+element.Name)); os.IsNotExist(err) {
	        	os.MkdirAll(filepath.Dir(path+"/"+v.Name+"/"+element.Name), 0700)
	        }
			err := ioutil.WriteFile(path+"/"+v.Name+"/"+element.Name, element.Data, 0644)
    		if err != nil {
    			panic(err)
    		}
	    }
	    
	    //Write values file
	    if _, err := os.Stat(filepath.Dir(path+"/"+v.Name+"/"+"values.json")); os.IsNotExist(err) {
        	os.MkdirAll(filepath.Dir(path+"/"+v.Name+"/"+"values.json"), 0700)
        }
        
        jsonString, err := json.Marshal(v.Chart.Values)
        valuesyaml, err := yaml.JSONToYAML(jsonString)
		err = ioutil.WriteFile(path+"/"+v.Name+"/"+"values.yaml", valuesyaml, 0644)
		if err != nil {
			panic(err)
		}
		
		//Write Chart metadata to chart.json file
	    if _, err := os.Stat(filepath.Dir(path+"/"+v.Name+"/"+"Chart.yaml")); os.IsNotExist(err) {
        	os.MkdirAll(filepath.Dir(path+"/"+v.Name+"/"+"Chart.yaml"), 0700)
        }
        
        jsonString, err = json.Marshal(v.Chart.Metadata)
        chartyaml, err := yaml.JSONToYAML(jsonString)
		err = ioutil.WriteFile(path+"/"+v.Name+"/"+"Chart.yaml", chartyaml, 0644)
		if err != nil {
			panic(err)
		}		
	}
}

func gunzipWrite(w io.Writer, data []byte) error {
	// Write gzipped data to the client
	gr, err := gzip.NewReader(bytes.NewBuffer(data))
	defer gr.Close()
	data, err = ioutil.ReadAll(gr)
	if err != nil {
		return err
	}
	w.Write(data)
	return nil
}