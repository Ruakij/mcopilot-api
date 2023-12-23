package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/pflag"
)

var (
	loginEmailFlag   = pflag.String("loginEmail", "", "Login Email")
	loginPasswordFlag   = pflag.String("loginPassword", "", "Login Password")
	loginTotpSecretFlag   = pflag.String("loginTotpSecret", "", "Login TOTP-Secret")
)

func init(){
	pflag.CommandLine.SortFlags = false
	pflag.Usage = func() {
		fmt.Fprintf(os.Stderr, "\nPrint path and content of all files recursively.\n\n %s [path ..]\n\n", filepath.Base(os.Args[0]))
		pflag.PrintDefaults()
	}

	pflag.Parse()

	checkFlags()
}

func checkFlags(){
	
}
