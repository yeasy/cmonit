package main

import (
	"github.com/op/go-logging"
	_ "github.com/spf13/cobra"
	_ "github.com/spf13/viper"
	"github.com/yeasy/cmonit/cmd"
	"fmt"
	"os"
)

var logger = logging.MustGetLogger("main")

const cmdRoot = "core"

func main() {
    if err := cmd.RootCmd.Execute(); err != nil {
        fmt.Println(err)
        os.Exit(-1)
    }
}