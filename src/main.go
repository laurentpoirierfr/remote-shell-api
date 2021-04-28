package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"text/template"

	b64 "encoding/base64"

	sse "github.com/alexandrevicenzi/go-sse"
	"github.com/google/uuid"
	"github.com/iancoleman/strcase"
	"github.com/spf13/viper"
)

func init() {
	app_name := "remote-shell-api"
	viper.SetConfigName("config")             // name of config file (without extension)
	viper.SetConfigType("yaml")               // REQUIRED if the config file does not have the extension in the name
	viper.AddConfigPath("/etc/" + app_name)   // path to look for the config file in
	viper.AddConfigPath("$HOME/." + app_name) // call multiple times to add many search paths
	viper.AddConfigPath(".")                  // optionally look for config in the working directory
	err := viper.ReadInConfig()               // Find and read the config file
	if err != nil {                           // Handle errors reading the config file
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}

}

func initialize() {

}

// Create SSE server
var s *sse.Server

func main() {
	s = sse.NewServer(nil)
	defer s.Shutdown()

	// Configure the console sse route
	http.Handle(viper.GetString("api.console_path"), s)

	// Static server
	fs := http.FileServer(http.Dir("./www"))
	http.Handle("/", fs)

	url := viper.GetString("api.command_path")
	http.HandleFunc(url, func(w http.ResponseWriter, r *http.Request) {
		uri := strings.Replace(r.URL.Path, url, "", 1)
		id := strings.Split(uri, "/")[0]
		shell := strings.Split(uri, "/")[1]

		command := "#! /bin/bash\n\n"
		for k, v := range r.URL.Query() {
			command += "export " + prepareParameter(k) + "=" + v[0] + "\n"
		}
		command += "\ncd " + viper.GetString("shell.id_entry_point") + "/" + id + "\n"
		command += "\n./" + shell + ".sh\n"

		uid := uuid.NewString()
		console := viper.GetString("api.console_path") + uid

		command64 := b64.StdEncoding.EncodeToString([]byte(command))

		response := Response{ID: uid, Command: command64, Console: console}

		w.Header().Set("Content-Type", "application/json")

		fmt.Fprintf(w, prepareResponse(response))

		go streamConsole(Response{ID: uid, Command: command, Console: console})

	})

	http.HandleFunc(viper.GetString("api.init_path"), func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		initialize()
		fmt.Fprintf(w, "{ \"status\": \"UP\" }")
	})

	// Start server
	port := ":" + viper.GetString("application.port")
	log.Println("Listening at " + port)
	http.ListenAndServe(port, nil)
}

func prepareParameter(str string) string {
	return strings.ToUpper(strcase.ToSnake(str))
}

type Response struct {
	ID      string
	Command string
	Console string
}

func prepareResponse(response Response) string {
	content, err := ioutil.ReadFile("./tpl/response.json.tpl")
	if err != nil {
		log.Fatal(err)
	}
	tmpl, err := template.New("response").Parse(string(content))
	if err != nil {
		panic(err)
	}
	var tpl bytes.Buffer
	if err = tmpl.Execute(&tpl, response); err != nil {
		panic(err)
	}

	return tpl.String()
}

func streamConsole(response Response) {
	filename := os.TempDir() + "/exec-" + response.ID + ".sh"
	fmt.Println(filename)

	command := []byte(response.Command)
	err := ioutil.WriteFile(filename, command, 0644)
	if err != nil {
		log.Fatal(err)
	}

	cmd := exec.Command("bash", filename)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Println(err)
	}

	err = cmd.Start()
	fmt.Println("The command is running")
	if err != nil {
		fmt.Println(err)
	}

	// print the output of the subprocess
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		m := scanner.Text()
		s.SendMessage(response.Console, sse.SimpleMessage(m))
	}
	cmd.Wait()

	// delete command file
	e := os.Remove(filename)
	if e != nil {
		log.Fatal(e)
	}
}
