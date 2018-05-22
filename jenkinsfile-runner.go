package main

import (
	"fmt"
	"os"
	"os/exec"
	home "github.com/mitchellh/go-homedir"
	"flag"
	"path/filepath"
)

var jenkinsfile string
var version string
var cache string
var workdir string
var configfile string

func main() {

	home, err := home.Dir()
	if err != nil {
		panic(err)
	}
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	flag.StringVar(&jenkinsfile, "file", filepath.Join(wd, "Jenkinsfile"), "Jenkinsfile to run")
	flag.StringVar(&version, "version", "latest", "Jenkins version to use")
	flag.StringVar(&cache, "cache", filepath.Join(home, ".jenkinsfile-runner"), "Directory used as download cache")
	flag.StringVar(&configfile, "config", filepath.Join(wd, "jenkins.yaml"), "Configuration as Code file to setup jenkins master matching pipeline requirements")

	flag.Parse()

	_, err = os.Stat(jenkinsfile)
	if os.IsNotExist(err) {
		fmt.Errorf("No such file %s", jenkinsfile)
		os.Exit(-1)
	}

	jenkinsfile, err = filepath.Abs(jenkinsfile)
	if err != nil {
		panic(err)
	}
	workdir = filepath.Join(filepath.Dir(jenkinsfile), ".jenkinsfile-runner")
	mkdir(workdir)

	mkdir(cache)

	if version == "latest" {
		version = getLatestCoreVersion()
	}
    fmt.Printf("Running Pipeline on jenkins %s\n", version)


	war, err := getJenkinsWar(version)
	if err != nil {
		panic(err)
	}

	mkdir(filepath.Join(workdir, "plugins"))
	installPlugins()
	InstallJenkinsfileRunner()

	writeFile(filepath.Join(workdir, "logging.properties"), `
.level = INFO
handlers= java.util.logging.ConsoleHandler
java.util.logging.ConsoleHandler.level=WARNING
java.util.logging.ConsoleHandler.formatter=java.util.logging.SimpleFormatter`)

    fmt.Println("Starting Jenkins...")

	cmd := exec.Command("java",
		// "-agentlib:jdwp=transport=dt_socket,server=y,suspend=n,address=5005",
		// disable setup wizard
		"-Djenkins.install.runSetupWizard=false",
		"-Djava.util.logging.config.file=.jenkinsfile-runner/logging.properties",
		"-jar", war, 
		// Disable http (so we can run in parallel without port collisions)
		"--httpPort=-1",
	)
	cmd.Env = append(os.Environ(),
		"JENKINS_HOME="+workdir,
		"JENKINSFILE="+jenkinsfile,
		"CASC_JENKINS_CONFIG="+configfile)

	cmd.Stdout = os.Stdout	
	cmd.Stderr = os.Stderr	

	if err := cmd.Run(); err != nil {
		fmt.Printf("cmd.Start() failed with %s\n", err)
		os.Exit(1)
	}
}




func InstallJenkinsfileRunner() {
	hpi := filepath.Join(workdir, "plugins", "jenkinsfile-runner.hpi")

	if _, err := os.Stat(hpi); err == nil {
		if err = os.Remove(hpi); err != nil {
			panic(err)
		}
	}	

	// TODO hpi file should be package within the jenkinsfile-runner binary as a "resource"
	home, err := home.Dir()
	if err != nil {
		panic(err)
	}
    if err := os.Link(home+"/.m2/repository/io/jenkins/plugins/jenkinsfile-runner/1.0-SNAPSHOT/jenkinsfile-runner-1.0-SNAPSHOT.hpi", hpi); err != nil {
        panic(err)
    }
}

