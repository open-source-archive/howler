package backend

import (
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"
	"text/template"

	"github.com/gin-gonic/gin"
	"github.com/golang/glog"
	"github.com/hashicorp/vault/api"
	"github.com/zalando-techmonkeys/gin-glog"
	"github.com/zalando-techmonkeys/howler/conf"
)

//FIXME: this should be a member of the vault structure, but the current use of values instead of pointers
//for methods makes it impossible. As long as this is not addressed, this variable will stay global.
var sharedSecret map[string]chan string

//due to plugin based architecture that has allows plugin to use a map[string]string to be used
//as configuration in the standard howler config.yaml, we have to check for presence of mandatory
//fields here manually
func mandatoryConfigCheck(config map[string]string) {
	if config["tokenTTL"] == "" {
		glog.Errorf("TTL configuration is empty, please provide a valid one.\n")
		os.Exit(1)
	}
	if config["vaultURI"] == "" {
		glog.Errorf("vaultURI is empty, please provide a valid one.\n")
		os.Exit(1)
	}
	if config["vaultToken"] == "" {
		glog.Errorf("vaultToken is empty, please provide a valid one.\n")
		os.Exit(1)
	}
}

//Vault is the basic type of the plugin
type Vault struct {
	config map[string]string
	name   string
}

//getSecret is the handler to read the secret from a channel based on the app id
func (v *Vault) getSecret(ginCtx *gin.Context) {
	appID := ginCtx.Params.ByName("appID")
	glog.Infof("App %s waiting to read cubbyhole token.\n", appID)
	createChannelIfNotExistent(appID)
	value := <-sharedSecret[appID]
	glog.Infof("Token for app %s will be sent\n", appID)
	ginCtx.JSON(http.StatusOK, gin.H{"secret": value})
}

//run starts the webserver
func (v *Vault) startServer() error {
	glog.Infof("Starting local server\n")
	router := gin.New()
	//TODO initialize configurations, correct middlewares, https/http
	router.Use(ginglog.Logger(5)) //5 seconds
	router.Use(gin.Recovery())

	//setting up https by default
	var tlsConfig = tls.Config{}
	keypair, err := tls.LoadX509KeyPair(v.config["tlsCertfilePath"], v.config["tlsKeyfilePath"])
	if err != nil {
		fmt.Printf("ERR: Could not load X509 KeyPair, caused by: %s\n", err)
		os.Exit(1) //exit explicitely as we choose a fail fast approach
	}
	tlsConfig.Certificates = []tls.Certificate{keypair}
	tlsConfig.NextProtos = []string{"http/1.1"}
	tlsConfig.Rand = rand.Reader

	router.GET("/secret/:appID", v.getSecret)
	serve := &http.Server{
		Addr:      fmt.Sprintf(":%s", v.config["serverPort"]),
		Handler:   router,
		TLSConfig: &tlsConfig,
	}
	err = serve.ListenAndServe()
	if err != nil {
		glog.Errorf("Cannot start server for Cubbyhole tokens distribution\n")
	}
	return err
}

//Register is used to register the vault plugin in howler
func (v *Vault) Register() error { //FIXME: error should always be the last error type
	v.name = "Vault"
	config := conf.New().Backends["vault"]
	mandatoryConfigCheck(config)
	v.config = config
	sharedSecret = make(map[string]chan string)
	go v.startServer()
	return nil
}

//pushValueToChannel pushes a value to a channel. It invokes "make" if this was not done before
func createChannelIfNotExistent(appID string) {
	if sharedSecret[appID] == nil {
		sharedSecret[appID] = make(chan string, 1)
	}
}

// HandleUpdate adds or removes container to loadbalancer pool
func (v *Vault) HandleUpdate(e StatusUpdateEvent) {
	switch e.Taskstatus {
	case "TASK_RUNNING":
		glog.Infof("Task is running, creating secrets\n")
		v.createSecrets(e)
	}
}

func (v *Vault) createSecrets(e StatusUpdateEvent) {
	vb := vaultBackend{}
	vb.appID = strings.TrimPrefix(e.Appid, "/") //Marathon specific, needed to remove initial "/" char
	createChannelIfNotExistent(vb.appID)
	//authenticate against vault using Th howler token
	err := vb.vaultAuthenticate(v.config["vaultURI"], v.config["vaultToken"])
	if err != nil {
		glog.Errorf("Cannot authenticate with Vault.\n")
		return
	}

	ttl := v.config["tokenTTL"]
	//create token T1 using howler policy (cubbyhole token)
	cubbyhole, err := vb.createToken(ttl)
	if err != nil {
		glog.Errorf("Cannot generate cubbyhole token.\n")
		return
	}

	teamName := vb.getTeamName(v.config["marathonEndpoint"], v.config["marathonUsername"], v.config["marathonPassword"])
	if teamName == "" {
		glog.Errorf("Cannot get team name\n")
		return
	}

	policy, err := vb.createNewPolicy(v.config["teamPolicyFile"], teamName)
	if err != nil {
		glog.Errorf("Cannot create new Policy\n")
		return
	}

	err = vb.usePolicy(policy)
	if err != nil {
		glog.Errorf("Cannot use generated policy:\n")
		return
	}

	//glog.Infof("created cubbyhole: " + cubbyhole) //TODO: uncomment line for debugging. Generated tokens must not be written to files.
	//create token T2 using app policy (secret token)
	secretToken, err := vb.createToken(ttl)
	if err != nil {
		glog.Errorf("Cannot generate secret token\n")
		return
	}
	//glog.Infof("created secret: " + secretToken) //TODO: uncomment line for debugging. Generated tokens must not be written to files.
	//authenticate with T1 => create a new client with that token
	err = vb.vaultAuthenticate(v.config["vaultURI"], cubbyhole) //after that "v" is fresh and ready to auth with cubbhyhole
	if err != nil {
		glog.Errorf("Cannot authenticate with cubbyhole token\n")
		return
	}
	//store secret T2 protected by cubbyhole token
	err = vb.storeInCubbyhole(secretToken)
	if err != nil {
		glog.Errorf("Error while storing in cubbyhole\n")
		return
	}
	//send token T1 in the channel (unlocks any possible waiting thread)
	sharedSecret[vb.appID] <- cubbyhole
	glog.Infof("Tokens creation done for %s", vb.appID)
	//TODO discard previous authentication
}

//HandleCreate does nothing in this case as we're not dealing with Create events
func (v *Vault) HandleCreate(e APIRequestEvent) {
	return //No need of actions in case of create requests
}

//HandleDestroy does nothing in this case as we're not dealing with Delete events
func (v *Vault) HandleDestroy(e AppTerminatedEvent) {
	return //No need of actions in case of destroy requests
}

//Name returns the backend service name
func (v *Vault) Name() string {
	return v.name
}

type vaultBackend struct {
	config *api.Config
	client *api.Client
	appID  string
}

func (vb *vaultBackend) vaultAuthenticate(vaultURI string, token string) error {
	vb.config = api.DefaultConfig()
	vb.config.Address = vaultURI
	client, err := api.NewClient(vb.config) //can probably be global
	if err != nil {
		glog.Errorf("Error authenticating %s\n", err.Error())
		return err
	}
	client.SetToken(token) //TODO put here the howler token to be read from file
	vb.client = client
	return nil
}

//teamTemplate is the structure used for template substitution
type teamTemplate struct {
	teamID string
	appID  string
}

func (vb *vaultBackend) createToken(ttl string) (string, error) {
	secret, err := vb.client.Auth().Token().Create(&api.TokenCreateRequest{
		Lease: ttl,
	})
	if err != nil {
		glog.Errorf("%s\n", err.Error())
		return "", err
	}
	return secret.Auth.ClientToken, nil
}

func (vb *vaultBackend) storeInCubbyhole(secretToken string) error {
	secretMap := map[string]interface{}{}
	secretMap["secret"] = secretToken
	_, err := vb.client.Logical().Write(fmt.Sprintf("/cubbyhole/%s", vb.appID), secretMap)
	if err != nil {
		glog.Errorf("Cannot write to logical: %s\n", err.Error())
		return err
	}
	return nil
}

//createNewPolicy creates a new policy for the given team name and app. It only returns an error if it failed
func (vb *vaultBackend) createNewPolicy(policyTemplate string, teamName string) (string, error) {
	t, err := template.ParseFiles(policyTemplate)
	if err != nil {
		glog.Errorf("Cannot parse file %s with error %s\n", policyTemplate, err.Error())
		return "", err
	}
	var out bytes.Buffer
	tpl, err := buildTemplate(teamName, vb.appID) //also does validation of parameters
	if err != nil {
		glog.Errorf("Cannot build a valid template: %s\n", err.Error())
		return "", err
	}
	err = t.Execute(&out, tpl)
	if err != nil {
		glog.Errorf("Cannot execute template with error %s\n", err.Error())
		return "", err
	}
	return out.String(), nil
}

//buildTemplate is a helper function that takes a teamName and an appID and returns a teamTemplate structure if parameters
//pass the validation
func buildTemplate(teamName string, appID string) (*teamTemplate, error) {
	if !isStringSafe(teamName) || !isStringSafe(appID) {
		return nil, errors.New("Invalid parameters for template.\n")
	}
	return &teamTemplate{teamID: teamName, appID: appID}, nil
}

//isStringSafe checks if a a string used for a template is safe
func isStringSafe(input string) bool {
	valid, err := regexp.MatchString("^[0-9a-zA-Z-. _\\/]+$", input)
	if err != nil {
		glog.Errorf("Error with regexp: %s\n", err.Error())
		return false
	}
	return valid
}

func (vb *vaultBackend) readPolicyFile(filename string) (string, error) {
	template, err := ioutil.ReadFile(filename)
	if err != nil {
		glog.Errorf("Cannot read policy from file %s, reason: %s.", filename, err.Error())
		return "", err
	}
	return string(template), err
}

func (vb *vaultBackend) usePolicy(template string) error {
	err := vb.client.Sys().PutPolicy(vb.appID, string(template))
	if err != nil {
		glog.Errorf("Error putting Vault policy: %s\n", err.Error())
		return err
	}
	return nil
}

//calls the marathon API back to get the team name
//assumes that the team is saved in the labels
func (vb *vaultBackend) getTeamName(endpoint string, username string, password string) string {
	//the call is just a plain rest call parsing for a specific field, no need to use the marathon go api here.
	client := &http.Client{}
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/%s", endpoint, vb.appID), nil)
	if err != nil {
		glog.Errorf("Cannot build request: %s\n", err.Error())
		return ""
	}
	if username != "" && password != "" {
		req.SetBasicAuth(username, password)
	}
	res, err := client.Do(req)
	if err != nil {
		glog.Errorf("Cannot GET app info from Marathon: %s\n", err.Error())
		defer res.Body.Close()
	}
	body, err := ioutil.ReadAll(res.Body)

	var data map[string]interface{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		glog.Errorf("Cannot unmarshal data: %s\n", err.Error())
		return ""
	}

	app := data["app"].(map[string]interface{})

	labels := app["labels"].(map[string]interface{})
	if labels["team"] != nil {
		return labels["team"].(string)
	}
	return ""
}
